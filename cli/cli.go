package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const EnvAPIUsername = "GGPROXY_API_USERNAME"
const EnvAPIPassword = "GGPROXY_API_PASSWORD"
const EnvLicense = "GGPROXY_LICENSE"
const LinkDashboardDownload = "https://graphguard.io/dashboard#download"

// Command can be any of:
//
//	CommandServe
//	CommandReload
//	CommandStop
//	CommandHelp
type Command any

type CommandServe struct {
	ConfigDirPath string
	LicenseToken  string
	APIUsername   string
	APIPassword   string
}

type CommandReload struct{}

type CommandStop struct{}

func Parse(
	w io.Writer,
	args []string,
	validateLicenseToken func(string) bool,
) (cmd Command) {
	fm := fmt.Sprintf

	executableName := "ggproxy"
	if len(args) > 0 {
		executableName = filepath.Base(args[0])
	}

	flags := flag.NewFlagSet("ggproxy", flag.ContinueOnError)
	flags.SetOutput(w)
	flags.Usage = func() {
		writeLines(w,
			fm("usage: %s <command> [flags]", executableName),
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
		c.APIUsername = os.Getenv(EnvAPIUsername)
		c.APIPassword = os.Getenv(EnvAPIPassword)
		c.LicenseToken = os.Getenv(EnvLicense)

		flags.Usage = func() {
			writeLines(w,
				"",
				fm("usage: %s serve [-config <path>]", executableName),
				"",
				"flags:",
				"-config <path>: defines the configuration directory path "+
					"(default: ./config)",
				"",
				"environment variables:",
				fm("%s: API basic auth username "+
					"(enables basic auth if set)", EnvAPIUsername),
				fm("%s: API basic auth password", EnvAPIPassword),
				fm("%s: License key", EnvLicense),
			)
		}

		flags.StringVar(&c.ConfigDirPath, "config", "./config", "")
		if !parseFlags() {
			return nil
		}

		if c.LicenseToken == "" {
			writeLines(w,
				EnvLicense+" isn't set.",
				fm("You can get the license key at %s", LinkDashboardDownload),
			)
			flags.Usage()
			return nil
		} else if !validateLicenseToken(c.LicenseToken) {
			writeLines(w,
				EnvLicense+" contains an invalid license key!",
				fm("You can get a valid license key at %s", LinkDashboardDownload),
			)
			flags.Usage()
			return nil
		}

		if c.APIUsername != "" && c.APIPassword == "" {
			writeLines(w,
				EnvAPIPassword+" isn't set.",
				"Make sure you provide it when "+EnvAPIUsername+" is defined.",
			)
			flags.Usage()
			return nil
		}

		cmd = c

	case "reload":
		if !parseFlags() {
			return nil
		}
		cmd = CommandReload{}

	case "stop":
		if !parseFlags() {
			return nil
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
