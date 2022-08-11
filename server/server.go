package server

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/graph-guard/gguard-proxy/config"
	"github.com/graph-guard/gguard-proxy/engines/rmap"
	"github.com/graph-guard/gguard-proxy/gqlreduce"
	"github.com/graph-guard/gguard-proxy/server/internal/tokenwriter"
	"github.com/graph-guard/gqt"
	"github.com/phuslu/log"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

type Server struct {
	config   *config.Config
	server   *fasthttp.Server
	client   *fasthttp.Client
	services map[string]*service
	log      log.Logger
}

type ServerDebug struct {
	config   *config.Config
	server   *fasthttp.Server
	services map[string]*service
	log      log.Logger
}

type service struct {
	source         string
	destination    string
	forwardReduced bool
	log            log.Logger
	matcherpool    sync.Pool
}

type matcher struct {
	Reducer *gqlreduce.Reducer
	Engine  *rmap.RulesMap
}

func New(
	config *config.Config,
	readTimeout, writeTimeout time.Duration,
	readBufferSize, writeBufferSize int,
	log log.Logger,
	client *fasthttp.Client,
) *Server {
	services := make(map[string]*service)

	if client == nil {
		client = &fasthttp.Client{}
	}

	srv := &Server{
		config: config,
		server: &fasthttp.Server{
			ReadTimeout:                  readTimeout,
			WriteTimeout:                 writeTimeout,
			ReadBufferSize:               readBufferSize,
			WriteBufferSize:              writeBufferSize,
			DisablePreParseMultipartForm: false,
		},
		client:   client,
		log:      log,
		services: services,
	}
	srv.server.Handler = srv.handle

	for _, serviceEnabled := range config.ServicesEnabled {
		s := serviceEnabled
		services[s.ID] = &service{
			source:         s.ID,
			destination:    s.ForwardURL,
			forwardReduced: s.ForwardReduced,
			log:            log,
			matcherpool: sync.Pool{
				New: func() any {
					d := make(map[string]gqt.Doc, len(s.TemplatesEnabled))
					for _, t := range s.TemplatesEnabled {
						d[t.ID] = t.Document
					}
					engine, err := rmap.New(d, 0)
					if err != nil {
						panic(fmt.Errorf(
							"initializing engine for service %q: %w",
							s.ID, err,
						))
					}
					reducer := gqlreduce.NewReducer()
					return &matcher{
						Reducer: reducer,
						Engine:  engine,
					}
				},
			},
		}

		// Warm up matcher pool
		func() {
			// n := runtime.NumCPU()
			n := 1
			m := make([]*matcher, n)
			for i := 0; i < n; i++ {
				m[i] = services[s.ID].matcherpool.Get().(*matcher)
			}
			for i := 0; i < n; i++ {
				services[s.ID].matcherpool.Put(m[i])
			}
		}()
	}

	return srv
}

func NewDebug(
	config *config.Config,
	readTimeout, writeTimeout time.Duration,
	readBufferSize, writeBufferSize int,
	log log.Logger,
) *ServerDebug {
	services := make(map[string]*service)

	srv := &ServerDebug{
		config: config,
		server: &fasthttp.Server{
			ReadTimeout:                  readTimeout,
			WriteTimeout:                 writeTimeout,
			ReadBufferSize:               readBufferSize,
			WriteBufferSize:              writeBufferSize,
			DisablePreParseMultipartForm: false,
		},
		log:      log,
		services: services,
	}
	srv.server.Handler = srv.handle

	for _, serviceEnabled := range config.ServicesEnabled {
		s := serviceEnabled
		d := make([]gqt.Doc, len(s.TemplatesEnabled))
		for i := range s.TemplatesEnabled {
			d[i] = s.TemplatesEnabled[i].Document
		}
		services[s.ID] = &service{
			source:         s.ID,
			destination:    s.ForwardURL,
			forwardReduced: s.ForwardReduced,
			log:            log,
			matcherpool: sync.Pool{
				New: func() any {
					d := make(map[string]gqt.Doc, len(s.TemplatesEnabled)+
						len(s.TemplatesDisabled))
					for _, t := range s.TemplatesEnabled {
						d[t.ID] = t.Document
					}
					for _, t := range s.TemplatesDisabled {
						d[t.ID] = t.Document
					}

					engine, err := rmap.New(d, 0)
					if err != nil {
						panic(fmt.Errorf(
							"initializing engine for service %q: %w",
							s.ID, err,
						))
					}
					reducer := gqlreduce.NewReducer()
					return &matcher{
						Reducer: reducer,
						Engine:  engine,
					}
				},
			},
		}

		// Warm up matcher pool
		func() {
			// n := runtime.NumCPU()
			n := 1
			m := make([]*matcher, n)
			for i := 0; i < n; i++ {
				m[i] = services[s.ID].matcherpool.Get().(*matcher)
			}
			for i := 0; i < n; i++ {
				services[s.ID].matcherpool.Put(m[i])
			}
		}()
	}

	return srv
}

func (s *Server) handle(ctx *fasthttp.RequestCtx) {
	s.log.Info().
		Bytes("path", ctx.Path()).
		Msg("handling request")

	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error(fasthttp.StatusMessage(
			fasthttp.StatusMethodNotAllowed,
		), fasthttp.StatusMethodNotAllowed)
		return
	}

	id := getIDFromPath(ctx)

	service, ok := s.services[string(id)]
	if !ok {
		s.log.Debug().
			Bytes("path", ctx.Path()).
			Msg("endpoint not found")
		ctx.Error(fasthttp.StatusMessage(
			fasthttp.StatusNotFound,
		), fasthttp.StatusNotFound)
		return
	}
	s.log.Debug().
		Bytes("path", ctx.Path()).
		Bytes("query", ctx.Request.Body()).
		Msg("")

	query, operationName, variablesJSON, err := extractData(ctx)
	if err {
		return
	}

	m := service.matcherpool.Get().(*matcher)
	defer service.matcherpool.Put(m)

	m.Reducer.Reduce(
		query, operationName, variablesJSON,
		func(operation []gqlreduce.Token) {
			if !m.Engine.Match(operation) {
				ctx.Error(fasthttp.StatusMessage(
					fasthttp.StatusForbidden,
				), fasthttp.StatusForbidden)
				return
			}

			// Forward request
			freq := fasthttp.AcquireRequest()
			fresp := fasthttp.AcquireResponse()
			defer func() {
				fasthttp.ReleaseRequest(freq)
				fasthttp.ReleaseResponse(fresp)
			}()

			ctx.Request.CopyTo(freq)

			if service.forwardReduced {
				if err := tokenwriter.Write(
					ctx.Request.BodyWriter(),
					operation,
				); err != nil {
					s.log.Error().
						Err(err).
						Msg("writing reduced to forward request body")
					ctx.Error(fasthttp.StatusMessage(
						fasthttp.StatusInternalServerError,
					), fasthttp.StatusInternalServerError)
					return
				}
			} else {
				// Forward original
				freq.SetBody(ctx.Request.Body())
			}

			freq.SetHost(service.destination)
			if err := s.client.Do(freq, fresp); err != nil {
				s.log.Error().Err(err).Msg("forwarding")
				ctx.Error(fasthttp.StatusMessage(
					fasthttp.StatusInternalServerError,
				), fasthttp.StatusInternalServerError)
				return
			}
		},
		func(err error) {
			s.log.Error().Err(err).Msg("reducer error")
			ctx.Error(
				fasthttp.StatusMessage(fasthttp.StatusBadRequest),
				fasthttp.StatusBadRequest,
			)
		},
	)
}

func (s *ServerDebug) handle(ctx *fasthttp.RequestCtx) {
	s.log.Info().
		Bytes("path", ctx.Path()).
		Msg("handling debug request")

	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error(fasthttp.StatusMessage(
			fasthttp.StatusMethodNotAllowed,
		), fasthttp.StatusMethodNotAllowed)
		return
	}

	id := getIDFromPath(ctx)

	service, ok := s.services[string(id)]
	if !ok {
		s.log.Debug().
			Bytes("path", ctx.Path()).
			Msg("endpoint not found")
		ctx.Error(fasthttp.StatusMessage(
			fasthttp.StatusNotFound,
		), fasthttp.StatusNotFound)
		return
	}
	s.log.Debug().
		Bytes("path", ctx.Path()).
		Bytes("query", ctx.Request.Body()).
		Msg("")

	query, operationName, variablesJSON, err := extractData(ctx)
	if err {
		return
	}

	m := service.matcherpool.Get().(*matcher)
	defer service.matcherpool.Put(m)

	m.Reducer.Reduce(
		query, operationName, variablesJSON,
		func(operation []gqlreduce.Token) {
			var match []string
			m.Engine.MatchAll(
				operation,
				func(id string) {
					match = append(match, id)
				},
			)
			{
				resp, err := json.Marshal(match)
				if err != nil {
					s.log.Error().Err(err).Msg("marshalling debug response")
				}
				ctx.Response.SetBody(resp)
			}
		},
		func(err error) {

		},
	)
}

func (s *Server) Serve(listener net.Listener) {
	serviceIDs := make([]string, len(s.config.ServicesEnabled))
	for i := range s.config.ServicesEnabled {
		serviceIDs[i] = s.config.ServicesEnabled[i].ID
	}
	s.log.Info().
		Str("host", s.config.Host).
		Strs("services", serviceIDs).
		Msg("listening")

	var err error
	if listener != nil {
		err = s.server.Serve(listener)
	} else {
		err = s.server.ListenAndServe(s.config.Host)
	}
	if err != nil {
		s.log.Fatal().Err(err).Msg("listening")
	}
}

// Shutdown returns once the server was shutdown.
// Logs shutdown and errors.
func (s *Server) Shutdown() error {
	err := s.server.Shutdown()
	if err != nil {
		s.log.Error().Err(err).Msg("shutting down")
		return err
	}
	s.log.Info().Msg("shutdown")
	return nil
}

func (s *ServerDebug) Serve(listener net.Listener) {
	serviceIDs := make([]string, len(s.config.ServicesEnabled))
	for i := range s.config.ServicesEnabled {
		serviceIDs[i] = s.config.ServicesEnabled[i].ID
	}
	s.log.Info().
		Str("host", s.config.DebugAPIHost).
		Strs("services", serviceIDs).
		Msg("listening")

	var err error
	if listener != nil {
		err = s.server.Serve(listener)
	} else {
		err = s.server.ListenAndServe(s.config.DebugAPIHost)
	}
	if err != nil {
		s.log.Fatal().Err(err).Msg("listening")
	}
}

// Shutdown returns once the server was shutdown.
// Logs shutdown and errors.
func (s *ServerDebug) Shutdown() error {
	err := s.server.Shutdown()
	if err != nil {
		s.log.Error().Err(err).Msg("shutting down")
		return err
	}
	s.log.Info().Msg("shutdown")
	return nil
}

func extractData(ctx *fasthttp.RequestCtx) (
	query []byte,
	operationName []byte,
	variablesJSON []byte,
	err bool,
) {
	b := ctx.Request.Body()
	if v := gjson.GetBytes(b, "query"); v.Raw != "" {
		query = []byte(v.String())
	} else {
		ctx.Error(fasthttp.StatusMessage(
			fasthttp.StatusBadRequest,
		), fasthttp.StatusBadRequest)
		err = true
		return
	}
	if v := gjson.GetBytes(b, "operationName"); v.Raw != "" {
		operationName = b[v.Index+1 : v.Index+len(v.Raw)-1]
	}
	if v := gjson.GetBytes(b, "variables"); v.Raw != "" {
		variablesJSON = b[v.Index : v.Index+len(v.Raw)]
	}
	return
}

func getIDFromPath(ctx *fasthttp.RequestCtx) []byte {
	if p := ctx.Path(); len(p) > 0 && p[0] == '/' {
		return p[1:]
	}
	return nil
}
