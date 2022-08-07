package main

import (
	"os"
	"os/signal"
)

// RegisterStop returns a channel that's closed once
// a termination signal is received.
func RegisterStop() (stop <-chan struct{}) {
	s := make(chan struct{})
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		close(s)
	}()
	return s
}
