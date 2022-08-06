package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
)

// Command can be any of:
//
//		CommandServe
//		CommandReload
//		CommandStop
//	 CommandHelp
type Command any

type CommandServe struct {
	ConfigDirPath string
}

type CommandReload struct{}

type CommandStop struct{}

func Parse(w io.Writer, args []string) (cmd Command) {
	executableName := "ggproxy"
	if len(args) > 0 {
		executableName = filepath.Base(args[0])
	}

	flags := flag.NewFlagSet("ggproxy", flag.ContinueOnError)
	flags.SetOutput(w)
	flags.Usage = func() {
		writeLines(w,
			fmt.Sprintf("usage: %s <command> [flags]", executableName),
			"",
			"commands available:",
			" serve - turns the CLI into a server and starts listening",
			" reload - reloads the server config",
			" stop - stops the server",
		)
	}

	parseFlags := func() (ok bool) {
		if err := flags.Parse(args[2:]); err != nil {
			// flags will automatically call .Usage()
			return false
		}
		return true
	}

	if len(args) < 2 {
		flags.Usage()
		return nil
	}

	switch args[1] {
	case "serve":
		flags.Usage = func() {
			writeLines(w,
				"",
				fmt.Sprintf("usage: %s serve [-config <path>]", executableName),
				"",
				"serve flags available:",
				"-config <path>: defines the configuration directory path "+
					"(default: ./config)",
			)
		}
		c := CommandServe{}
		flags.StringVar(&c.ConfigDirPath, "config", "./config", "")
		if !parseFlags() {
			return
		}
		cmd = c

	case "reload":
		if !parseFlags() {
			return
		}
		cmd = CommandReload{}

	case "stop":
		if !parseFlags() {
			return
		}
		cmd = CommandStop{}

	case "help":
		PrintHelp(w)
		return

	default:
		flags.Usage()
		return nil
	}
	return cmd
}

func writeLines(w io.Writer, lines ...string) {
	for i := range lines {
		_, _ = w.Write([]byte(lines[i]))
		_, _ = w.Write([]byte("\n"))
	}
}

func PrintHelp(w io.Writer) {
	_, _ = w.Write([]byte("ggproxy"))
}
