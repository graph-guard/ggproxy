package main

import (
	"io"
	"sync"
	"time"

	"github.com/graph-guard/gguard-proxy/cli"
	"github.com/graph-guard/gguard-proxy/server"
	"github.com/phuslu/log"
)

// serve turns the CLI process into a ggproxy server process
// on *nix systems.
func serve(w io.Writer, c cli.CommandServe) {
	conf := ReadConfig(w, c.ConfigDirPath)
	if conf == nil {
		return
	}

	l := log.Logger{
		Level:  log.InfoLevel,
		Writer: &log.IOWriter{Writer: w},
	}

	var s *server.Ingress
	{
		lServer := l
		lServer.Context = log.NewContext(nil).
			Str("server", "ingress").Value()
		s = server.NewIngress(
			conf,
			10*time.Second,
			10*time.Second,
			1024*1024*4,
			1024*1024*4,
			lServer,
			nil,
			nil,
		)
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)

	// explicitStop must be closed to trigger an explicit stop.
	explicitStop := make(chan struct{})
	// stopped will be closed once all components have terminated gracefuly.
	stopped := make(chan struct{})
	stopTriggered := RegisterStop(explicitStop)

	start := time.Now()

	var api *server.API
	{
		lServerAPI := l
		lServerAPI.Context = log.NewContext(nil).
			Str("server", "api").Value()

		if conf.API.Host != "" {
			wg.Add(1)
			api = server.NewAPI(
				conf,
				10*time.Second,
				10*time.Second,
				lServerAPI,
				nil,
				start,
			)
		}
	}

	cmdServerStarted := make(chan bool)

	// Start command server
	cleanup := createVarDir(w, l)
	if cleanup == nil {
		return
	}
	defer cleanup(l)
	go func() {
		runCmdSockServer(
			l,
			stopTriggered,
			stopped,
			explicitStop,
			cmdServerStarted,
		)
		wg.Done()
	}()

	if !<-cmdServerStarted {
		l.Info().Msg("aborting launch, the command server failed to start")
		return
	}

	if api != nil {
		// Start API server
		go func() {
			<-stopTriggered
			_ = api.Shutdown()
		}()
		go func() {
			defer wg.Done()
			api.Serve(nil)
		}()
	}

	// Start main ingress server
	go func() {
		<-stopTriggered
		_ = s.Shutdown()
	}()
	func() {
		defer wg.Done()
		s.Serve(nil)
	}()

	wg.Wait()
	close(stopped)
}
