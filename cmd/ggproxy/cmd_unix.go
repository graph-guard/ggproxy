package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/phuslu/log"
)

const DirVarRun = "/var/run/ggproxy"
const FilePathPID = DirVarRun + "/ggproxy.pid"
const FilePathCmdSock = DirVarRun + "/ggproxy_cmd.sock"
const BufLenCmdSockRead = 64
const BufLenCmdSockWrite = 64 * 1024

// getPID reads the /var/run/ggproxy/ggproxy.pid file on *nix systems.
func getPID() string {
	b, err := os.ReadFile(FilePathPID)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	return string(b)
}

// runCmdSockServer starts listening on /var/run/ggproxy/ggproxy_cmd.sock
// and sends started<-true, otherwise sends started<-false if it failed.
func runCmdSockServer(
	l log.Logger,
	stopTriggered <-chan struct{},
	stopped <-chan struct{},
	explicitStop chan<- struct{},
	started chan<- bool,
) {
	lt, err := net.Listen("unix", FilePathCmdSock)
	if err != nil {
		l.Error().
			Err(err).
			Str("cmdSockFilePath", FilePathCmdSock).
			Msg("listening on file command socket")
		started <- false
	}
	go func() {
		<-stopTriggered
		if err := lt.Close(); err != nil {
			l.Error().
				Err(err).
				Msg("closing command socket listener")
		}
	}()
	started <- true
	for {
		c, err := lt.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				// Command server shutdown
				return
			}
			l.Error().
				Err(err).
				Str("cmdSockFilePath", FilePathCmdSock).
				Msg("accepting on file command socket")
			break
		}
		go handleCmdSockConn(c, l, explicitStop, stopped)
	}
}

// handleCmdSockConn starts listening for commands on the given connection.
func handleCmdSockConn(
	c net.Conn,
	l log.Logger,
	explicitStop chan<- struct{},
	stopped <-chan struct{},
) {
	bufRead := make([]byte, BufLenCmdSockRead)
	bufWrite := make([]byte, BufLenCmdSockWrite)
	for {
		nr, err := c.Read(bufRead)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return // Close connection
			}
			l.Error().
				Err(err).
				Str("cmdSockFilePath", FilePathCmdSock).
				Msg("reading command socket")
			return // Close connection
		}

		written := handleCmdSockMsg(
			bufRead[:nr],
			bufWrite[:0],
			l,
			explicitStop,
			stopped,
		)
		if written == nil {
			return // Close connection
		}
		if cap(written) > BufLenCmdSockWrite {
			l.Error().
				Str("cmdSockFilePath", FilePathCmdSock).
				Int("bufferCapacity", cap(written)).
				Int("capacityLimit", BufLenCmdSockWrite).
				Msg("command socket writer buffer capacity exceeds limit")
			return // Close connection
		}

		if _, err := c.Write(written); err != nil {
			l.Error().
				Err(err).
				Str("cmdSockFilePath", FilePathCmdSock).
				Msg("writing to command socket")
			return // Close connection
		}
	}
}

// handleCmdSockMsg handles a command socket message.
// Returns nil if the connection must be closed.
func handleCmdSockMsg(
	msg, buf []byte,
	l log.Logger,
	explicitStop chan<- struct{},
	stopped <-chan struct{},
) (written []byte) {
	if string(msg) == "reload" {
		l.Info().
			Str("command", "reload").
			Msg("command received")
		buf = append(buf, "err:reload is not yet supported"...)

	} else if string(msg) == "stats" {
		l.Info().
			Str("command", "stats").
			Msg("command received")
		buf = append(buf, "err:stats is not yet supported"...)

	} else if string(msg) == "stop" {
		l.Info().
			Str("command", "stop").
			Msg("command received")
		close(explicitStop)
		<-stopped
		buf = append(buf, "ok"...)

	} else {
		l.Error().
			Bytes("message", msg).
			Msg("unsupported command received")
		return nil

	}
	return buf
}

// createVarDir creates the following files:
//
//	/var/run/ggproxy/ggproxy_cmd.sock
//	/var/run/ggproxy/ggproxy.pid
func createVarDir(w io.Writer, l log.Logger) (cleanup func(log.Logger)) {
	if pid := getPID(); pid != "" {
		fmt.Fprintf(w, "another instance is already running "+
			"(process id: %s)\n", pid)
		return nil
	}

	pidFile, err := os.Create(FilePathPID)
	if err != nil {
		fmt.Fprintf(w, "creating %q: %s\n", FilePathPID, err)
		return nil
	}
	cleanup = func(l log.Logger) {
		if err := os.Remove(FilePathPID); err != nil {
			l.Error().
				Err(err).
				Str("filePath", FilePathPID).
				Msg("deleting PID file")
		}
	}

	pid := os.Getpid()
	if _, err := pidFile.WriteString(strconv.Itoa(pid)); err != nil {
		cleanup(l)
		fmt.Fprintf(w, "writing process ID to %q: %s\n", FilePathPID, err)
		return nil
	}
	return cleanup
}

// request connects to /var/run/ggproxy/ggproxy_cmd.sock and sends
// a command request to the running server instance.
func request(msg, buf []byte) ([]byte, error) {
	c, err := net.Dial("unix", FilePathCmdSock)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoInstanceRunning
		}
		return nil, fmt.Errorf("dialing %q: %w", FilePathCmdSock, err)
	}
	defer c.Close()

	if len(msg) > BufLenCmdSockRead {
		panic(fmt.Errorf(
			"message length (%d) exceeds limit (%d)",
			len(msg), BufLenCmdSockRead,
		))
	}

	if _, err := c.Write(msg); err != nil {
		return nil, fmt.Errorf("writing to %q: %w", FilePathCmdSock, err)
	}
	n, err := c.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("reading from %q: %w", FilePathCmdSock, err)
	}

	return buf[:n], nil
}

var ErrNoInstanceRunning = errors.New("no instance running")
