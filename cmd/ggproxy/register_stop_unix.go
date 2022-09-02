package main

import (
	"os"
	"os/signal"
	"syscall"
)

// RegisterStop returns a channel that's closed once
// either a termination signal is received or explicitStop is triggered.
func RegisterStop(
	explicitStop <-chan struct{},
) (stopTriggered <-chan struct{}) {
	s := make(chan struct{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP)
	signal.Notify(interrupt, syscall.SIGINT)
	signal.Notify(interrupt, syscall.SIGQUIT)
	signal.Notify(interrupt, syscall.SIGABRT)
	signal.Notify(interrupt, syscall.SIGTERM)
	signal.Notify(interrupt, syscall.SIGPIPE)
	go func() {
		select {
		case <-explicitStop:
		case <-interrupt:
		}
		close(s)
	}()
	return s
}
