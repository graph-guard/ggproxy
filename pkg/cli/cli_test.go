package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/graph-guard/ggproxy/pkg/cli"

	"github.com/stretchr/testify/require"
)

func helpOutput(execName string) string {
	return lines(
		fmt.Sprintf("usage: %s <command> [flags]", execName),
		"",
		"commands available:",
		" serve - turns the CLI into a server and starts listening",
		" reload - reloads the server config",
		" stop - stops the server",
	)
}

func TestNoArgs(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(out, nil)
	require.Nil(t, c)
	require.Equal(t, helpOutput("ggproxy"), out.String())
}

func TestNoCommand(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(out, []string{"execname"})
	require.Nil(t, c)
	require.Equal(t, helpOutput("execname"), out.String())
}

func TestUnknownCommand(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(out, []string{"execname", "unknown-command"})
	require.Nil(t, c)
	require.Equal(t, helpOutput("execname"), out.String())
}

func TestCommandServe(t *testing.T) {
	os.Setenv(cli.EnvAPIUsername, "testusername")
	os.Setenv(cli.EnvAPIPassword, "testpassword")

	t.Run("default_config_path", func(t *testing.T) {
		out := new(bytes.Buffer)
		c := cli.Parse(out, []string{"ggproxy", "serve"})
		require.Equal(t, cli.CommandServe{
			ConfigDirPath: "/etc/ggproxy",
			APIUsername:   "testusername",
			APIPassword:   "testpassword",
		}, c)
		require.Equal(t, "", out.String())
	})

	t.Run("custom_config_path", func(t *testing.T) {
		out := new(bytes.Buffer)
		c := cli.Parse(out, []string{
			"ggproxy", "serve",
			"-config", "./custom_config",
		})
		require.Equal(t, cli.CommandServe{
			ConfigDirPath: "./custom_config",
			APIUsername:   "testusername",
			APIPassword:   "testpassword",
		}, c)
		require.Equal(t, "", out.String())
	})

	t.Run("unknown_flags", func(t *testing.T) {
		out := new(bytes.Buffer)
		c := cli.Parse(out, []string{
			"ggproxy", "serve",
			"-unknown", "foobar",
		})
		require.Nil(t, c)
		require.Equal(t,
			lines(
				"flag provided but not defined: -unknown",
				"",
				"usage: ggproxy serve [-config <path>]",
				"",
				"flags:",
				"-config <path>: "+
					"defines the configuration directory path "+
					"(default: /etc/ggproxy)",
				"",
				"environment variables:",
				"GGPROXY_API_USERNAME: API basic auth username "+
					"(enables basic auth if set)",
				"GGPROXY_API_PASSWORD: API basic auth password",
			),
			out.String(),
		)
	})
}

func TestAPIPasswordNotSet(t *testing.T) {
	out := new(bytes.Buffer)
	os.Setenv(cli.EnvAPIUsername, "testusername")
	os.Setenv(cli.EnvAPIPassword, "")
	c := cli.Parse(out, []string{"ggproxy", "serve"})
	require.Nil(t, c)
	require.Equal(t,
		lines(
			fmt.Sprintf("%s isn't set.", cli.EnvAPIPassword),
			fmt.Sprintf(
				"Make sure you provide it when %s is defined.",
				cli.EnvAPIUsername,
			),
			"",
			"usage: ggproxy serve [-config <path>]",
			"",
			"flags:",
			"-config <path>: "+
				"defines the configuration directory path "+
				"(default: /etc/ggproxy)",
			"",
			"environment variables:",
			"GGPROXY_API_USERNAME: API basic auth username "+
				"(enables basic auth if set)",
			"GGPROXY_API_PASSWORD: API basic auth password",
		),
		out.String(),
	)
}

func TestCommandReload(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(out, []string{"execname", "reload"})
	require.Equal(t, cli.CommandReload{}, c)
	require.Equal(t, "", out.String())
}

func TestCommandStop(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(out, []string{"execname", "stop"})
	require.Equal(t, cli.CommandStop{}, c)
	require.Equal(t, "", out.String())
}

func TestCommandHelp(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(out, []string{"execname", "help"})
	require.Nil(t, c)

	e := new(bytes.Buffer)
	cli.PrintHelp(e)
	require.Equal(t, e.String(), out.String())
}

func lines(lines ...string) string {
	var b strings.Builder
	for i := range lines {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	return b.String()
}
