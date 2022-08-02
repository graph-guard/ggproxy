package server

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	radix "github.com/armon/go-radix"
	"github.com/graph-guard/gguard/engines/rmap"
	"github.com/graph-guard/gguard/matcher"
	plog "github.com/phuslu/log"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

var ErrUnknownCtxType = errors.New("unknown context type")

type Server struct {
	Server  *fasthttp.Server
	log     plog.Logger
	workers *radix.Tree
	listen  string
	debug   bool
}

type ServiceConfig struct {
	Name        string
	Source      string
	Destination string
	Engine      matcher.Matcher
}

type worker struct {
	source      string
	destination string
	engine      matcher.Matcher
	log         plog.Logger
	debug       bool
}

func New(
	services []ServiceConfig, name string, listenAddress string, debug bool,
	readTimeOut time.Duration, log plog.Logger,
) *Server {
	w := radix.New()

	srv := &Server{
		Server: &fasthttp.Server{
			Name:        name,
			ReadTimeout: readTimeOut,
		},
		log:     log,
		listen:  listenAddress,
		debug:   debug,
		workers: w,
	}
	srv.Server.Handler = srv.handle

	for _, svc := range services {
		_, updated := w.Insert(svc.Source, worker{
			svc.Source,
			svc.Destination,
			svc.Engine,
			log,
			debug,
		})
		if updated {
			log.Info().Str("prefix", svc.Source).Msg("updated")
		}
	}

	return srv
}

func (w worker) process(ctx context.Context) error {
	c, ok := ctx.(*fasthttp.RequestCtx)

	if ok {
		body := gjson.GetManyBytes(c.Request.Body(), "query", "operationName", "variables")
		if !w.debug {
			if err := w.engine.Match(
				ctx, []byte(body[0].String()), []byte(body[1].String()), []byte(body[2].String())); err != nil {
				return err
			}
		} else {
			match := []string{}
			if err := w.engine.MatchAll(
				ctx, []byte(body[0].String()), []byte(body[1].String()), []byte(body[2].String()),
				func(templateIndex int) {
					match = append(match, strconv.Itoa(templateIndex))
				}); err != nil {
				return err
			}
			if len(match) > 0 {
				w.log.Debug().Str("source", w.source).Str("match", strings.Join(match, " ")).Msg("")
			} else {
				return errors.New("no match")
			}
		}
	} else {
		return ErrUnknownCtxType
	}

	return nil
}

func (s *Server) handle(ctx *fasthttp.RequestCtx) {
	s.log.Info().Bytes("path", ctx.Path()).Msg("handling request")

	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error(fasthttp.StatusMessage(
			fasthttp.StatusMethodNotAllowed,
		), fasthttp.StatusMethodNotAllowed)
		return
	}

	p, v, m := s.workers.LongestPrefix(string(ctx.Path()))

	switch m {
	case true:
		w := v.(worker)
		s.log.Debug().Bytes("path", ctx.Path()).Bytes("query", ctx.Request.Body()).Str("prefix", p).Msg("")

		err := w.process(ctx)
		if err == nil {
			s.log.Debug().Bytes("path", ctx.Path()).Msg("pass")
			ctx.Request.SetHost(w.destination)
			err := fasthttp.Do(&ctx.Request, &ctx.Response)
			if err != nil {
				s.log.Error().Err(err).Msg("")
				return
			}
		} else {
			switch err.(type) {
			case *rmap.ErrReducer:
				s.log.Debug().Bytes("path", ctx.Path()).Msg("malformed")
				ctx.Error(fasthttp.StatusMessage(
					fasthttp.StatusBadRequest,
				), fasthttp.StatusBadRequest)
			default:
				s.log.Debug().Bytes("path", ctx.Path()).Msg("filtered")
				ctx.Error(fasthttp.StatusMessage(
					fasthttp.StatusForbidden,
				), fasthttp.StatusForbidden)
			}
		}
	default:
		s.log.Debug().Bytes("path", ctx.Path()).Msg("not existing endpoint")
		ctx.Error(fasthttp.StatusMessage(
			fasthttp.StatusNotFound,
		), fasthttp.StatusNotFound)
	}
}

func (s *Server) Serve() {
	s.log.Info().Str("name", s.Server.Name).Str("listenAddress", s.listen).Msg("startig")

	if err := s.Server.ListenAndServe(s.listen); err != nil {
		s.log.Fatal().Err(err).Msg("")
	}
}
