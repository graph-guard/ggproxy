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

	var s *server.Server
	{
		lServer := l
		lServer.Context = log.NewContext(nil).
			Str("server", "ingress").Value()
		s = server.New(
			conf,
			10*time.Second,
			10*time.Second,
			1024*1024*4,
			1024*1024*4,
			lServer,
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

	var sDebug *server.ServerDebug
	{
		lServerDebug := l
		lServerDebug.Context = log.NewContext(nil).
			Str("server", "debug").Value()

		if conf.DebugAPIHost != "" {
			wg.Add(1)
			sDebug = server.NewDebug(
				conf,
				10*time.Second,
				10*time.Second,
				1024*1024*4,
				1024*1024*4,
				lServerDebug,
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

	if sDebug != nil {
		// Start debug server
		go func() {
			<-stopTriggered
			_ = sDebug.Shutdown()
		}()
		go func() {
			defer wg.Done()
			sDebug.Serve(nil)
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
