package cli_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/graph-guard/gguard-proxy/cli"

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
	t.Run("default_config_path", func(t *testing.T) {
		out := new(bytes.Buffer)
		c := cli.Parse(out, []string{"ggproxy", "serve"})
		require.Equal(t, cli.CommandServe{
			ConfigDirPath: "./config",
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
				"serve flags available:",
				"-config <path>: "+
					"defines the configuration directory path "+
					"(default: ./config)",
			),
			out.String(),
		)
	})
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
