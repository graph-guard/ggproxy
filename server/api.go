package server

import (
	"crypto/tls"
	"fmt"
	stdlog "log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/graph-guard/gguard-proxy/api/graph"
	"github.com/graph-guard/gguard-proxy/api/graph/generated"
	"github.com/graph-guard/gguard-proxy/api/graph/model"
	"github.com/graph-guard/gguard-proxy/config"
	"github.com/graph-guard/gguard-proxy/engines/rmap"
	"github.com/graph-guard/gguard-proxy/gqlreduce"
	"github.com/graph-guard/gqt"
	plog "github.com/phuslu/log"
	"github.com/valyala/fasthttp"
)

// API is the metrics, inspection and debug server
type API struct {
	lock           sync.Mutex
	config         *config.Config
	server         *http.Server
	log            plog.Logger
	graph          *handler.Server
	maxReqBodySize int
}

func NewAPI(
	conf *config.Config,
	readTimeout, writeTimeout time.Duration,
	log plog.Logger,
	tlsConfig *tls.Config,
	start time.Time, // When was the server started?
) *API {
	lHTTPServer := log
	lHTTPServer.Context = plog.NewContext(nil).
		Str("server-module", "fasthttp").Value()

	graphServer := makeGraphServer(start, conf)

	srv := &API{
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
		log:            log,
		graph:          graphServer,
		maxReqBodySize: conf.API.MaxReqBodySizeBytes,
	}
	srv.server.Handler = srv
	return srv
}

func (s *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.log.Info().
		Str("path", r.URL.Path).
		Msg("handling request")

	switch string(r.Method) {
	case fasthttp.MethodPost:
		switch r.URL.Path {
		case "/graph":
			func() {
				s.lock.Lock()
				defer s.lock.Unlock()
				s.graph.ServeHTTP(w, r)
			}()
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
) *handler.Server {
	reducer := gqlreduce.NewReducer()
	services := makeServices(conf)
	s := handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{Resolvers: &graph.Resolver{
				Start:    start,
				Conf:     conf,
				Reducer:  reducer,
				Services: services,
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

func makeServices(conf *config.Config) map[string]*model.Service {
	m := make(
		map[string]*model.Service,
		len(conf.ServicesEnabled)+len(conf.ServicesDisabled),
	)
	for _, s := range conf.ServicesEnabled {
		m[s.ID] = makeService(conf, s, true)
	}
	for _, s := range conf.ServicesDisabled {
		m[s.ID] = makeService(conf, s, false)
	}
	return m
}

func makeService(
	c *config.Config,
	s *config.Service,
	enabled bool,
) *model.Service {
	service := &model.Service{
		TemplatesByID: make(
			map[string]*model.Template,
			len(s.TemplatesEnabled)+len(s.TemplatesDisabled),
		),
		ID:                s.ID,
		ForwardURL:        s.ForwardURL,
		Enabled:           enabled,
		TemplatesEnabled:  make([]*model.Template, len(s.TemplatesEnabled)),
		TemplatesDisabled: make([]*model.Template, len(s.TemplatesDisabled)),
	}

	{ // Initialize matcher engine
		d := make(map[string]gqt.Doc, len(s.TemplatesEnabled)+
			len(s.TemplatesDisabled))
		for i, t := range s.TemplatesEnabled {
			d[t.ID] = t.Document
			tm := &model.Template{
				Service: service,

				ID:         t.ID,
				Tags:       t.Tags,
				Source:     string(t.Source),
				Statistics: nil, //TODO
				Enabled:    true,
			}
			service.TemplatesEnabled[i] = tm
			service.TemplatesByID[t.ID] = tm
		}
		for i, t := range s.TemplatesDisabled {
			d[t.ID] = t.Document
			tm := &model.Template{
				Service: service,

				ID:         t.ID,
				Tags:       t.Tags,
				Source:     string(t.Source),
				Statistics: nil, //TODO
				Enabled:    false,
			}
			service.TemplatesDisabled[i] = tm
			service.TemplatesByID[t.ID] = tm
		}

		var err error
		service.Matcher, err = rmap.New(d, 0)
		if err != nil {
			panic(fmt.Errorf(
				"initializing matcher for service %q: %w",
				s.ID, err,
			))
		}
	}

	{ // Set ingress URL
		scheme := "http"
		if c.Ingress.TLS.CertFile != "" {
			scheme = "https"
		}
		u := url.URL{
			Scheme: scheme,
			Host:   c.Ingress.Host,
			Path:   s.ID,
		}
		service.IngressURL = u.String()
	}

	return service
}
