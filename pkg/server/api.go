package server

import (
	"context"
	"crypto/tls"
	"errors"
	stdlog "log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/graph-guard/ggproxy/pkg/api/graph"
	"github.com/graph-guard/ggproxy/pkg/api/graph/generated"
	"github.com/graph-guard/ggproxy/pkg/api/graph/model"
	"github.com/graph-guard/ggproxy/pkg/config"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon"
	gqt "github.com/graph-guard/gqt/v4"
	plog "github.com/phuslu/log"
	"github.com/valyala/fasthttp"
)

// API is the metrics, inspection and debug server
type API struct {
	auth         Auth
	config       *config.Config
	server       *http.Server
	log          plog.Logger
	graphHandler http.HandlerFunc

	lock  sync.Mutex
	graph *handler.Server
}

type Auth struct {
	Username string
	Password string
}

func NewAPI(
	auth Auth,
	conf *config.Config,
	readTimeout, writeTimeout time.Duration,
	log plog.Logger,
	tlsConfig *tls.Config,
	start time.Time, // When was the server started?
	proxyServer *Proxy,
) *API {
	lHTTPServer := log
	lHTTPServer.Context = plog.NewContext(nil).
		Str("server-module", "fasthttp").Value()

	graphServer := makeGraphServer(start, conf, proxyServer)

	srv := &API{
		auth:   auth,
		config: conf,
		server: &http.Server{
			Addr:         conf.API.Host,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			TLSConfig:    tlsConfig,
			ErrorLog: stdlog.New(&logWriter{
				Log: lHTTPServer,
				Msg: "http server log",
			}, "", 0),
		},
		log:   log,
		graph: graphServer,
	}
	srv.server.Handler = srv
	srv.graphHandler = makeBasicAuth(
		auth.Username,
		auth.Password,
		srv.handleGraph,
	)
	return srv
}

func (s *API) handleGraph(w http.ResponseWriter, r *http.Request) {
	func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		s.graph.ServeHTTP(w, r)
	}()
}

func (s *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch string(r.Method) {
	case fasthttp.MethodPost:
		switch r.URL.Path {
		case "/graph":
			s.graphHandler(w, r)
		default:
			const c = http.StatusNotFound
			http.Error(w, http.StatusText(c), c)
			return
		}
	default:
		const c = http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(c), c)
		return
	}
}

func (s *API) Serve(listener net.Listener) {
	serviceIDs := make([]string, len(s.config.ServicesEnabled))
	for i := range s.config.ServicesEnabled {
		serviceIDs[i] = s.config.ServicesEnabled[i].ID
	}
	s.log.Info().
		Str("host", s.config.API.Host).
		Bool("tls", s.config.API.TLS.CertFile != "").
		Strs("services", serviceIDs).
		Bool("auth", s.auth.Username != "").
		Msg("listening")

	var err error
	if s.config.API.TLS.CertFile != "" {
		// TLS enabled
		if listener != nil {
			err = s.server.ServeTLS(
				listener,
				s.config.API.TLS.CertFile,
				s.config.API.TLS.KeyFile,
			)
		} else {
			err = s.server.ListenAndServeTLS(
				s.config.API.TLS.CertFile,
				s.config.API.TLS.KeyFile,
			)
		}
	} else {
		// TLS disabled
		if listener != nil {
			err = s.server.Serve(listener)
		} else {
			err = s.server.ListenAndServe()
		}
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.log.Fatal().Err(err).Msg("listening")
	}
}

// Shutdown returns once the server was shutdown.
// Logs shutdown and errors.
func (s *API) Shutdown() error {
	err := s.server.Shutdown(context.Background())
	if err != nil {
		s.log.Error().Err(err).Msg("shutting down")
		return err
	}
	s.log.Info().Msg("shutdown")
	return nil
}

func makeBasicAuth(
	username, password string,
	next http.HandlerFunc,
) http.HandlerFunc {
	if username == "" {
		// No auth
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`,
		)
		u, p, ok := r.BasicAuth()
		if !ok {
			const c = http.StatusUnauthorized
			http.Error(w, http.StatusText(c), c)
			return
		}
		if u != username || p != password {
			const c = http.StatusForbidden
			http.Error(w, http.StatusText(c), c)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type logWriter struct {
	Log plog.Logger
	Msg string
}

func (w *logWriter) Write(data []byte) (int, error) {
	w.Log.Info().Bytes("data", data).Msg(w.Msg)
	return len(data), nil
}

func makeGraphServer(
	start time.Time,
	conf *config.Config,
	proxyServer *Proxy,
) *handler.Server {
	services := makeServices(conf, proxyServer)
	s := handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{Resolvers: &graph.Resolver{
				Start:    start,
				Conf:     conf,
				Services: services,
				Log:      proxyServer.log,
			}},
		),
	)

	s.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	s.AddTransport(transport.Options{})
	// gServer.AddTransport(transport.GET{})
	s.AddTransport(transport.POST{})
	s.AddTransport(transport.MultipartForm{})

	s.SetQueryCache(lru.New(1000))

	s.Use(extension.Introspection{})
	s.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	return s
}

func makeServices(
	conf *config.Config,
	proxyServer *Proxy,
) map[string]*model.Service {
	m := make(map[string]*model.Service, len(conf.Services))
	for _, s := range conf.Services {
		m[s.ID] = makeService(conf, s, proxyServer)
	}
	return m
}

func makeService(
	c *config.Config,
	s *config.Service,
	proxyServer *Proxy,
) *model.Service {
	stats := proxyServer.GetServiceStatistics(s.ID)
	service := &model.Service{
		Stats: stats,
		Templates: make(
			map[*config.Template]*model.Template,
			len(s.Templates),
		),
		ID:               s.ID,
		ForwardURL:       s.ForwardURL,
		Enabled:          s.Enabled,
		TemplatesEnabled: make([]*model.Template, len(s.TemplatesEnabled)),
		TemplatesDisabled: make(
			[]*model.Template,
			len(s.Templates)-len(s.TemplatesEnabled),
		),
	}

	{ // Initialize engine
		d := make(map[string]*gqt.Operation, len(s.Templates))
		for _, t := range s.Templates {
			d[t.ID] = t.GQTTemplate
			tm := &model.Template{
				Service: service,
				Stats:   proxyServer.GetTemplateStatistics(s.ID, t.ID),
				ID:      t.ID,
				Tags:    t.Tags,
				Source:  string(t.Source),
				Enabled: t.Enabled,
			}
			service.TemplatesEnabled = append(service.TemplatesEnabled, tm)
			service.Templates[t] = tm
		}

		service.Engine = playmon.New(s)
	}

	{ // Set proxy URL
		scheme := "http"
		if c.Proxy.TLS.CertFile != "" {
			scheme = "https"
		}
		u := url.URL{
			Scheme: scheme,
			Host:   c.Proxy.Host,
			Path:   s.ID,
		}
		service.ProxyURL = u.String()
	}

	return service
}
