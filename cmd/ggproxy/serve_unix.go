package main

import (
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/graph-guard/gguard-proxy/cli"
	"github.com/graph-guard/gguard-proxy/server"
	"github.com/phuslu/log"
)

func serve(w io.Writer, c cli.CommandServe) {
	conf := ReadConfig(w, c.ConfigDirPath)
	if conf == nil {
		return
	}

	{
		if pid := getPID(); pid != "" {
			_, _ = w.Write([]byte("another instance is already running "))
			_, _ = w.Write([]byte("(process id: "))
			_, _ = w.Write([]byte(pid))
			_, _ = w.Write([]byte(")\n"))
			return
		}

		// Create PID file
		pidFile, err := os.Create(PIDFilePath)
		if err != nil {
			_, _ = w.Write([]byte("creating PID file: "))
			_, _ = w.Write([]byte(err.Error()))
			_, _ = w.Write([]byte("\n"))
			return
		}
		defer func() {
			if err := os.Remove(PIDFilePath); err != nil {
				_, _ = w.Write([]byte("deleting PID file: "))
				_, _ = w.Write([]byte(err.Error()))
				_, _ = w.Write([]byte("\n"))
			}
		}()

		pid := os.Getpid()
		if _, err := pidFile.WriteString(strconv.Itoa(pid)); err != nil {
			_, _ = w.Write([]byte("writing process ID to "))
			_, _ = w.Write([]byte(PIDFilePath))
			_, _ = w.Write([]byte(": "))
			_, _ = w.Write([]byte(err.Error()))
			_, _ = w.Write([]byte("\n"))
			return
		}
	}

	// Turn the CLI process into a server process
	s := server.New(
		conf,
		10*time.Second,
		10*time.Second,
		1024*1024*4,
		1024*1024*4,
		log.Logger{
			Level:  log.InfoLevel,
			Writer: &log.IOWriter{Writer: w},
		},
		nil,
	)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	signal.Notify(interrupt, syscall.SIGTERM)

	go func() {
		<-interrupt
		_ = s.Shutdown()
	}()

	s.Serve(nil)
}
