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

	stop := RegisterStop()

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
	close := createVarDir(w, l)
	if close == nil {
		return
	}
	defer close(l)
	go func() {
		runCmdSockServer(l, stop, s, cmdServerStarted)
		wg.Done()
	}()

	if !<-cmdServerStarted {
		l.Info().Msg("aborting launch, the command server failed to start")
		return
	}

	if sDebug != nil {
		// Start debug server
		go func() {
			<-stop
			_ = sDebug.Shutdown()
		}()
		go func() {
			defer wg.Done()
			sDebug.Serve(nil)
		}()
	}

	// Start main ingress server
	go func() {
		<-stop
		_ = s.Shutdown()
	}()
	func() {
		defer wg.Done()
		s.Serve(nil)
	}()

	wg.Wait()
}
