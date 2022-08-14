package main

import (
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/graph-guard/gguard-proxy/config"
	"github.com/graph-guard/gguard-proxy/server"
	plog "github.com/phuslu/log"
	"github.com/valyala/fasthttp"
)

func main() {
	s := server.NewIngress(
		&config.Config{
			Ingress: config.ServerConfig{
				Host: "localhost:8080",
				TLS: config.TLS{
					CertFile: "localhost.crt",
					KeyFile:  "localhost.key",
				},
			},
			API: &config.ServerConfig{
				Host: "localhost:8081",
			},
			ServicesEnabled: []*config.Service{
				{
					ID:   "testservice",
					Name: "Test Service",
					TemplatesEnabled: []*config.Template{
						{
							ID:       "sometemplate",
							Name:     "Some Template",
							Source:   []byte("query { foo }"),
							Document: nil,
						},
					},
				},
			},
		},
		10*time.Second, 10*time.Second,
		64*1024, 64*1024,
		plog.Logger{
			Level:      plog.DebugLevel,
			TimeField:  "time",
			TimeFormat: "23:59:59",
			Writer: &plog.ConsoleWriter{
				ColorOutput: true,
			},
		},
		&fasthttp.Client{},
		nil,
	)

	var wg sync.WaitGroup
	wg.Add(1)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		defer wg.Done()
		<-sig
		_ = s.Shutdown()
	}()

	s.Serve(nil)
	wg.Wait()
}
