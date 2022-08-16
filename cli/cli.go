package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const APIUsernameEnv = "GGPROXY_API_USERNAME"
const APIPasswordEnv = "GGPROXY_API_PASSWORD"

// Command can be any of:
//
//	CommandServe
//	CommandReload
//	CommandStop
//	CommandHelp
type Command any

type CommandServe struct {
	APIUsername   string
	APIPassword   string
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
		err := flags.Parse(args[2:])
		// flags will automatically call .Usage()
		return err == nil
	}

	if len(args) < 2 {
		flags.Usage()
		return nil
	}

	switch args[1] {
	case "serve":
		c := CommandServe{}
		c.APIUsername = os.Getenv(APIUsernameEnv)
		c.APIPassword = os.Getenv(APIPasswordEnv)

		flags.Usage = func() {
			writeLines(w,
				"",
				fmt.Sprintf("usage: %s serve [-config <path>]", executableName),
				"",
				"flags:",
				"-config <path>: defines the configuration directory path "+
					"(default: ./config)",
				"",
				"environment variables:",
				fmt.Sprintf("%s: API basic auth username "+
					"(enables basic auth if set)", APIUsernameEnv),
				fmt.Sprintf("%s: API basic auth password", APIPasswordEnv),
			)
		}

		flags.StringVar(&c.ConfigDirPath, "config", "./config", "")
		if !parseFlags() {
			return
		}

		if c.APIUsername != "" && c.APIPassword == "" {
			writeLines(w,
				APIPasswordEnv+" isn't set.",
				"Make sure you provide it when "+APIUsernameEnv+" is defined.",
			)
			flags.Usage()
			os.Exit(1)
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
