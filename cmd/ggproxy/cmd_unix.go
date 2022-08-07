package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/graph-guard/gguard-proxy/server"
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
func runCmdSockServer(
	l log.Logger,
	stop <-chan struct{},
	s *server.Server,
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
		<-stop
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
		go handleCmdSockConn(c, l, s)
	}
}

// handleCmdSockConn starts listening for commands on the given connection.
func handleCmdSockConn(c net.Conn, l log.Logger, s *server.Server) {
	bufRead := make([]byte, BufLenCmdSockRead)
	bufWrite := make([]byte, BufLenCmdSockWrite)
	for {
		nr, err := c.Read(bufRead)
		if err != nil {
			l.Error().
				Err(err).
				Str("cmdSockFilePath", FilePathCmdSock).
				Msg("reading command socket")
			return // Close connection
		}

		written := handleCmdSockMsg(bufRead[:nr], bufWrite[:0], l, s)
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
	s *server.Server,
) (written []byte) {
	if string(msg) == "reload" {
		l.Error().
			Str("command", "reload").
			Msg("command received")
		buf = append(buf, "err:reload is not yet supported"...)

	} else if string(msg) == "stats" {
		l.Error().
			Str("command", "stats").
			Msg("command received")
		buf = append(buf, "err:stats is not yet supported"...)

	} else if string(msg) == "stop" {
		l.Error().
			Str("command", "stop").
			Msg("command received")
		_ = s.Shutdown()

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
func createVarDir(w io.Writer, l log.Logger) (close func(log.Logger)) {
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
	close = func(l log.Logger) {
		if err := os.Remove(FilePathPID); err != nil {
			l.Error().
				Err(err).
				Str("filePath", FilePathPID).
				Msg("deleting PID file")
		}
	}

	pid := os.Getpid()
	if _, err := pidFile.WriteString(strconv.Itoa(pid)); err != nil {
		close(l)
		fmt.Fprintf(w, "writing process ID to %q: %s\n", FilePathPID, err)
		return nil
	}
	return close
}
