package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/engines/rmap"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/statistics"
	"github.com/graph-guard/ggproxy/utilities/tokenwriter"
	"github.com/graph-guard/gqt"
	plog "github.com/phuslu/log"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

// Proxy is the server receiving incomming proxy traffic
type Proxy struct {
	config   *config.Config
	server   *fasthttp.Server
	client   *fasthttp.Client
	services map[string]*service
	log      plog.Logger
}

type service struct {
	id                 string
	forwardURL         string
	forwardReduced     bool
	log                plog.Logger
	matcherpool        sync.Pool
	statistics         *statistics.ServiceSync
	templateStatistics map[string]*statistics.TemplateSync
}

type matcher struct {
	Parser *gqlparse.Parser
	Engine *rmap.RulesMap
}

func NewProxy(
	conf *config.Config,
	readTimeout, writeTimeout time.Duration,
	readBufferSize, writeBufferSize int,
	log plog.Logger,
	client *fasthttp.Client,
	tlsConfig *tls.Config,
) *Proxy {
	services := make(map[string]*service)

	if client == nil {
		client = &fasthttp.Client{}
	}

	lFasthttp := log
	lFasthttp.Context = plog.NewContext(nil).
		Str("server-module", "fasthttp").Value()

	srv := &Proxy{
		config: conf,
		server: &fasthttp.Server{
			ReadTimeout:                  readTimeout,
			WriteTimeout:                 writeTimeout,
			ReadBufferSize:               readBufferSize,
			WriteBufferSize:              writeBufferSize,
			DisablePreParseMultipartForm: false,
			TLSConfig:                    tlsConfig,
			Logger:                       &lFasthttp,
			MaxRequestBodySize:           conf.Proxy.MaxReqBodySizeBytes,
		},
		client:   client,
		log:      log,
		services: services,
	}
	srv.server.Handler = srv.handle

	for _, s := range conf.ServicesEnabled {
		templateStatistics := make(
			map[string]*statistics.TemplateSync,
			len(s.TemplatesEnabled),
		)
		for _, t := range s.TemplatesEnabled {
			templateStatistics[t.ID] = statistics.NewTemplateSync()
		}

		services[s.Path] = &service{
			id:             s.ID,
			forwardURL:     s.ForwardURL,
			forwardReduced: s.ForwardReduced,
			log:            log,
			matcherpool: sync.Pool{
				New: func() any {
					d := make(map[string]*gqt.Operation, len(s.TemplatesEnabled))
					for _, t := range s.TemplatesEnabled {
						d[t.ID] = t.Operation
					}
					engine, err := rmap.New(d, 0)
					if err != nil {
						panic(fmt.Errorf(
							"initializing engine for service %q: %w",
							s.ID, err,
						))
					}
					parser := gqlparse.NewParser()
					return &matcher{
						Parser: parser,
						Engine: engine,
					}
				},
			},
			statistics:         statistics.NewServiceSync(),
			templateStatistics: templateStatistics,
		}

		// Warm up matcher pool
		func() {
			// n := runtime.NumCPU()
			n := 1
			m := make([]*matcher, n)
			for i := 0; i < n; i++ {
				m[i] = services[s.Path].matcherpool.Get().(*matcher)
			}
			for i := 0; i < n; i++ {
				services[s.Path].matcherpool.Put(m[i])
			}
		}()
	}

	return srv
}

func (s *Proxy) GetServiceStatistics(id string) *statistics.ServiceSync {
	for _, s := range s.services {
		if s.id == id {
			return s.statistics
		}
	}
	return nil
}

func (s *Proxy) GetTemplateStatistics(
	serviceID, templateID string,
) *statistics.TemplateSync {
	if s, ok := s.services[serviceID]; ok {
		if s, ok := s.templateStatistics[templateID]; ok {
			return s
		}
	}
	return nil
}

func (s *Proxy) handle(ctx *fasthttp.RequestCtx) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error().Msg(r.(error).Error())
		}
	}()
	start := time.Now()
	s.log.Info().
		Bytes("path", ctx.Path()).
		Msg("handling request")

	if string(ctx.Method()) != fasthttp.MethodPost {
		const c = fasthttp.StatusMethodNotAllowed
		ctx.Error(fasthttp.StatusMessage(c), c)
		return
	}

	service, ok := s.services[string(ctx.Path())]
	if !ok {
		s.log.Debug().
			Bytes("path", ctx.Path()).
			Msg("endpoint not found")
		const c = fasthttp.StatusNotFound
		ctx.Error(fasthttp.StatusMessage(c), c)
		return
	}
	body := ctx.Request.Body()
	s.log.Debug().
		Bytes("path", ctx.Path()).
		Bytes("query", body).
		Msg("")

	query, operationName, variablesJSON, err := extractData(ctx)
	if err {
		return
	}

	m := service.matcherpool.Get().(*matcher)
	defer service.matcherpool.Put(m)

	m.Parser.Parse(
		query, operationName, variablesJSON,
		func(
			varVals [][]gqlparse.Token,
			operation []gqlparse.Token,
			selectionSet []gqlparse.Token,
		) {
			templateID := m.Engine.Match(varVals, operation[0].ID, selectionSet)
			if templateID == "" {
				timeProcessing := time.Since(start)
				service.statistics.Update(
					len(body), 0,
					true,
					timeProcessing, 0,
				)
				ctx.Error(fasthttp.StatusMessage(
					fasthttp.StatusForbidden,
				), fasthttp.StatusForbidden)
				return
			}

			templateStatistics := service.templateStatistics[templateID]

			timeProcessing := time.Since(start)
			startForward := time.Now()

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
						Msg("writing parsed to forward request body")
					ctx.Error(fasthttp.StatusMessage(
						fasthttp.StatusInternalServerError,
					), fasthttp.StatusInternalServerError)
					return
				}
			} else {
				// Forward original
				freq.SetBody(ctx.Request.Body())
			}

			ctx.Request.Header.VisitAll(func(key, value []byte) {
				freq.Header.SetBytesKV(key, value)
			})

			// Setting proxy headers and the forward URL
			freq.Header.Add("X-Forwarded-Host", string(ctx.Host()))
			freq.Header.Add("X-Forwarded-For", ctx.RemoteIP().String())
			freq.Header.Add("X-Forwarded-Proto", string(ctx.Request.Header.Protocol()))
			freq.SetRequestURI(service.forwardURL)

			if err := s.client.Do(freq, fresp); err != nil {
				s.log.Error().Err(err).Msg("forwarding")
				ctx.Error(fasthttp.StatusMessage(
					fasthttp.StatusInternalServerError,
				), fasthttp.StatusInternalServerError)
				return
			}

			fresp.Header.VisitAll(func(key, value []byte) {
				ctx.Response.Header.SetBytesKV(key, value)
			})
			ctx.Response.SetStatusCode(fresp.StatusCode())
			ctx.Response.SetBody(fresp.Body())

			timeForwarding := time.Since(startForward)
			service.statistics.Update(
				len(body), 0,
				true,
				timeProcessing, timeForwarding,
			)
			templateStatistics.Update(
				timeProcessing, timeForwarding,
			)
		},
		func(err error) {
			s.log.Error().Err(err).Msg("parser error")

			timeProcessing := time.Since(start)
			service.statistics.Update(
				len(body), 0,
				true,
				timeProcessing, 0,
			)

			ctx.Error(
				fasthttp.StatusMessage(fasthttp.StatusBadRequest),
				fasthttp.StatusBadRequest,
			)
		},
	)
}

func (s *Proxy) Serve(listener net.Listener) {
	serviceIDs := make([]string, len(s.config.ServicesEnabled))
	for i := range s.config.ServicesEnabled {
		serviceIDs[i] = s.config.ServicesEnabled[i].ID
	}
	s.log.Info().
		Str("host", s.config.Proxy.Host).
		Bool("tls", s.config.Proxy.TLS.CertFile != "").
		Strs("services", serviceIDs).
		Msg("listening")

	var err error
	if s.config.Proxy.TLS.CertFile != "" {
		// TLS enabled
		if listener != nil {
			err = s.server.ServeTLS(
				listener,
				s.config.Proxy.TLS.CertFile,
				s.config.Proxy.TLS.KeyFile,
			)
		} else {
			err = s.server.ListenAndServeTLS(
				s.config.Proxy.Host,
				s.config.Proxy.TLS.CertFile,
				s.config.Proxy.TLS.KeyFile,
			)
		}
	} else {
		// TLS disabled
		if listener != nil {
			err = s.server.Serve(listener)

		} else {
			err = s.server.ListenAndServe(s.config.Proxy.Host)
		}
	}
	if err != nil {
		s.log.Fatal().Err(err).Msg("listening")
	}
}

// Shutdown returns once the server was shutdown.
// Logs shutdown and errors.
func (s *Proxy) Shutdown() error {
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
