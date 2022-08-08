package main

import (
	"os"
	"os/signal"
)

// RegisterStop returns a channel that's closed once
// either a termination signal is received or explicitStop is triggered.
func RegisterStop(
	explicitStop <-chan struct{},
) (stopTriggered <-chan struct{}) {
	s := make(chan struct{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		select {
		case <-explicitStop:
		case <-interrupt:
		}
		close(s)
	}()
	return s
}
