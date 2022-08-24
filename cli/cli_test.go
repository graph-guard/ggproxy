package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/graph-guard/ggproxy/cli"

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
	c := cli.Parse(out, nil, func(s string) bool { return true })
	require.Nil(t, c)
	require.Equal(t, helpOutput("ggproxy"), out.String())
}

func TestNoCommand(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(
		out,
		[]string{"execname"},
		func(s string) bool { return true },
	)
	require.Nil(t, c)
	require.Equal(t, helpOutput("execname"), out.String())
}

func TestUnknownCommand(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(
		out,
		[]string{"execname", "unknown-command"},
		func(s string) bool { return true },
	)
	require.Nil(t, c)
	require.Equal(t, helpOutput("execname"), out.String())
}

func TestCommandServe(t *testing.T) {
	os.Setenv(cli.EnvLicense, "TESTLICENSETOKEN")
	os.Setenv(cli.EnvAPIUsername, "testusername")
	os.Setenv(cli.EnvAPIPassword, "testpassword")

	t.Run("default_config_path", func(t *testing.T) {
		out := new(bytes.Buffer)
		c := cli.Parse(
			out,
			[]string{"ggproxy", "serve"},
			func(s string) bool { return true },
		)
		require.Equal(t, cli.CommandServe{
			ConfigDirPath: "/etc/ggproxy",
			LicenseToken:  "TESTLICENSETOKEN",
			APIUsername:   "testusername",
			APIPassword:   "testpassword",
		}, c)
		require.Equal(t, "", out.String())
	})

	t.Run("custom_config_path", func(t *testing.T) {
		out := new(bytes.Buffer)
		c := cli.Parse(
			out,
			[]string{
				"ggproxy", "serve",
				"-config", "./custom_config",
			},
			func(s string) bool { return true },
		)
		require.Equal(t, cli.CommandServe{
			LicenseToken:  "TESTLICENSETOKEN",
			ConfigDirPath: "./custom_config",
			APIUsername:   "testusername",
			APIPassword:   "testpassword",
		}, c)
		require.Equal(t, "", out.String())
	})

	t.Run("unknown_flags", func(t *testing.T) {
		out := new(bytes.Buffer)
		c := cli.Parse(
			out,
			[]string{
				"ggproxy", "serve",
				"-unknown", "foobar",
			},
			func(s string) bool { return true },
		)
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
				"GGPROXY_LICENSE: License key",
			),
			out.String(),
		)
	})
}

func TestAPIPasswordNotSet(t *testing.T) {
	out := new(bytes.Buffer)
	os.Setenv(cli.EnvAPIUsername, "testusername")
	os.Setenv(cli.EnvAPIPassword, "")
	c := cli.Parse(
		out,
		[]string{"ggproxy", "serve"},
		func(s string) bool { return true },
	)
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
			"GGPROXY_LICENSE: License key",
		),
		out.String(),
	)
}

func TestLicenseTokenNotSet(t *testing.T) {
	out := new(bytes.Buffer)
	os.Setenv(cli.EnvAPIUsername, "testusername")
	os.Setenv(cli.EnvAPIPassword, "testpassword")
	os.Setenv(cli.EnvLicense, "")
	c := cli.Parse(
		out,
		[]string{"ggproxy", "serve"},
		func(s string) bool { return true },
	)
	require.Nil(t, c)

	require.Equal(t,
		lines(
			fmt.Sprintf("%s isn't set.", cli.EnvLicense),
			fmt.Sprintf(
				"You can get the license key at %s",
				cli.LinkDashboardDownload,
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
			"GGPROXY_LICENSE: License key",
		),
		out.String(),
	)
}

func TestLicenseTokenInvalid(t *testing.T) {
	out := new(bytes.Buffer)
	os.Setenv(cli.EnvAPIUsername, "testusername")
	os.Setenv(cli.EnvAPIPassword, "testpassword")
	os.Setenv(cli.EnvLicense, "thiskeyisinvalid")
	c := cli.Parse(
		out,
		[]string{"ggproxy", "serve"},
		func(s string) bool { return s == "valid" },
	)
	require.Nil(t, c)

	require.Equal(t,
		lines(
			fmt.Sprintf("%s contains an invalid license key!", cli.EnvLicense),
			fmt.Sprintf(
				"You can get a valid license key at %s",
				cli.LinkDashboardDownload,
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
			"GGPROXY_LICENSE: License key",
		),
		out.String(),
	)
}

func TestCommandReload(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(
		out,
		[]string{"execname", "reload"},
		func(s string) bool { return true },
	)
	require.Equal(t, cli.CommandReload{}, c)
	require.Equal(t, "", out.String())
}

func TestCommandStop(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(
		out,
		[]string{"execname", "stop"},
		func(s string) bool { return true },
	)
	require.Equal(t, cli.CommandStop{}, c)
	require.Equal(t, "", out.String())
}

func TestCommandHelp(t *testing.T) {
	out := new(bytes.Buffer)
	c := cli.Parse(
		out,
		[]string{"execname", "help"},
		func(s string) bool { return true },
	)
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
