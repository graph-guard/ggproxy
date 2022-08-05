package config_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/graph-guard/gguard-proxy/config"
	"github.com/graph-guard/gqt"
	"github.com/stretchr/testify/require"
)

type TestOK struct {
	Filesystem fstest.MapFS
	Expect     *config.Config
}

type TestError struct {
	Filesystem fstest.MapFS
	Check      func(*testing.T, error)
}

func TestReadConfig(t *testing.T) {
	for _, td := range []TestOK{
		{
			Filesystem: validFS(),
			Expect: &config.Config{
				ServicesEnabled: []*config.Service{
					{
						ID:             "service_a",
						Name:           "Service A",
						ForwardURL:     "http://localhost:8080/path",
						ForwardReduced: true,
						TemplatesEnabled: []*config.Template{
							{
								ID:     "template_b",
								Tags:   []string{"tag_b1", "tag_b2"},
								Source: lines(`query { bar }`),
								Document: gqt.DocQuery{
									Selections: []gqt.Selection{
										gqt.SelectionField{
											Name: "bar",
										},
									},
								},
							},
							{
								ID:     "template_a",
								Name:   "Template A",
								Tags:   []string{"tag_a"},
								Source: lines(`query { foo }`),
								Document: gqt.DocQuery{
									Selections: []gqt.Selection{
										gqt.SelectionField{
											Name: "foo",
										},
									},
								},
							},
						},
					},
				},
				ServicesDisabled: []*config.Service{
					{
						ID:             "service_b",
						ForwardURL:     "http://localhost:9090/",
						ForwardReduced: false,
						TemplatesDisabled: []*config.Template{
							{
								ID:     "template_c",
								Source: []byte(`query { maz }`),
								Document: gqt.DocQuery{
									Selections: []gqt.Selection{
										gqt.SelectionField{
											Name: "maz",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			c, err := config.ReadConfig(td.Filesystem, ".")
			require.NoError(t, err)
			require.Equal(t, td.Expect, c)
		})
	}
}

func TestReadConfigErrorMissingConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			err := testError(t, fstest.MapFS{
				filepath.Join(m, "service_a", "irrelevant_file.txt"): {
					Data: []byte(`this file only keeps the directory`),
				},
			})
			require.Equal(t, "service \"service_a\": "+
				"missing config.yml", err.Error())
		})
	}
}

func TestReadConfigErrorDuplicateTemplate(t *testing.T) {
	f := validFS()
	f[filepath.Join(
		config.ServicesEnabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"dup.gqt",
	)] = &fstest.MapFile{
		Data: []byte(`query { duplicate }`),
	}
	f[filepath.Join(
		config.ServicesEnabledDir,
		"service_a",
		config.TemplatesDisabledDir,
		"dup.gqt",
	)] = &fstest.MapFile{
		Data: []byte(`query { duplicate }`),
	}
	err := testError(t, f)
	require.Equal(t, `service "service_a": `+
		`template "dup" is `+
		"both enabled and disabled", err.Error())
}

func TestReadConfigErrorDuplicateService(t *testing.T) {
	err := testError(t, fstest.MapFS{
		filepath.Join(config.ServicesEnabledDir, "service_a", "config.yml"): {
			Data: []byte(`forward_url: localhost:8080/`),
		},
		filepath.Join(config.ServicesDisabledDir, "service_a", "config.yml"): {
			Data: []byte(`forward_url: localhost:8080/`),
		},
	})
	require.Equal(t, "service \"service_a\" is "+
		"both enabled and disabled", err.Error())
}

func TestReadConfigErrorDuplicateServiceConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			err := testError(t, fstest.MapFS{
				filepath.Join(m, "service_a", "config.yml"): {
					Data: []byte(`forward_url: localhost:8080`),
				},
				filepath.Join(m, "service_a", "config.yaml"): {
					Data: []byte(`forward_url: localhost:9090`),
				},
			})
			require.Equal(t, "service \"service_a\": conflicting files: "+
				fmt.Sprintf("%q", filepath.Join(
					m, "service_a", config.ServiceConfigFile1,
				))+
				" - "+
				fmt.Sprintf("%q", filepath.Join(
					m, "service_a", config.ServiceConfigFile2,
				)), err.Error())
		})
	}
}

func TestReadConfigErrorMissingForwardURL(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			err := testError(t, fstest.MapFS{
				filepath.Join(m, "service_a", "config.yml"): {
					Data: []byte(`forward_reduced: true`),
				},
			})
			require.Equal(t, "service \"service_a\":"+
				" reading \"config.yml\": "+
				"missing forward_url", err.Error())
		})
	}
}

func TestReadConfigErrorInvalidForwardURL(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			err := testError(t, fstest.MapFS{
				filepath.Join(m, "service_a", "config.yml"): {
					Data: []byte(`forward_url: not_a_url.`),
				},
			})
			require.Equal(t, "service \"service_a\":"+
				" reading \"config.yml\": "+
				"parse \"not_a_url.\": invalid URI for request", err.Error())
		})
	}
}

func TestReadConfigErrorInvalidTemplate(t *testing.T) {
	f := validFS()
	f[filepath.Join(
		config.ServicesDisabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"template_invalid.gqt",
	)] = &fstest.MapFile{
		Data: []byte(`invalid { template }`),
	}
	err := testError(t, f)
	require.Equal(t, "service \"service_a\": "+
		"parsing \"services_disabled/service_a/"+
		"templates_enabled/template_invalid.gqt\": "+
		"error at 0: unexpected definition", err.Error())
}

func TestReadConfigErrorInvalidTemplateID(t *testing.T) {
	f := validFS()
	f[filepath.Join(
		config.ServicesDisabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"template-invalid#.gqt",
	)] = &fstest.MapFile{
		Data: []byte(`invalid { template }`),
	}
	err := testError(t, f)
	require.Equal(t, "service \"service_a\": "+
		"validating \"services_disabled/service_a/"+
		"templates_enabled/template-invalid#.gqt\": "+
		"illegal identifier", err.Error())
}

func TestReadConfigErrorInvalidServiceID(t *testing.T) {
	f := validFS()
	f[filepath.Join(
		config.ServicesDisabledDir,
		"service_#1",
		config.ServiceConfigFile1,
	)] = &fstest.MapFile{
		Data: []byte(`forward_url: localhost:8080/`),
	}
	err := testError(t, f)
	require.Equal(t, "service \"service_#1\": "+
		"validating \"services_disabled/service_#1\": "+
		"illegal identifier", err.Error())
}

func lines(lines ...string) []byte {
	var b strings.Builder
	for i := range lines {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func testError(
	t *testing.T,
	filesystem fstest.MapFS,
) error {
	t.Helper()
	c, err := config.ReadConfig(filesystem, ".")
	require.Error(t, err)
	require.Nil(t, c)
	return err
}

func validFS() fstest.MapFS {
	return fstest.MapFS{
		filepath.Join(
			config.ServicesEnabledDir,
			"service_a",
			config.ServiceConfigFile1,
		): &fstest.MapFile{
			Data: lines(
				`name: "Service A"`,
				`forward_url: "http://localhost:8080/path"`,
				`forward_reduced: true`,
			),
		},
		filepath.Join(
			config.ServicesEnabledDir,
			"service_a",
			config.TemplatesEnabledDir,
			"template_a.gqt",
		): &fstest.MapFile{
			Data: lines(
				"---",
				`name: "Template A"`,
				"tags:",
				"  - tag_a",
				"---",
				`query { foo }`,
			),
		},
		filepath.Join(
			config.ServicesEnabledDir,
			"service_a",
			config.TemplatesEnabledDir,
			"Template_B.gqt",
		): &fstest.MapFile{
			Data: lines(
				"---",
				"tags:",
				"  - tag_b1",
				"  - tag_b2",
				"---",
				`query { bar }`,
			),
		},

		filepath.Join(
			config.ServicesDisabledDir,
			"service_b",
			config.ServiceConfigFile1,
		): &fstest.MapFile{
			Data: lines(
				`forward_url: "http://localhost:9090/"`,
			),
		},
		filepath.Join(
			config.ServicesDisabledDir,
			"service_b",
			config.TemplatesDisabledDir,
			"template_c.gqt",
		): &fstest.MapFile{
			Data: []byte(`query { maz }`),
		},

		// Ignored files
		filepath.Join(
			config.ServicesDisabledDir,
			"ignored_file1.txt",
		): &fstest.MapFile{
			Data: []byte(`this file should be ignored`),
		},
		filepath.Join(
			config.ServicesDisabledDir,
			"service_b",
			config.TemplatesDisabledDir,
			"ignored_file2.txt",
		): &fstest.MapFile{
			Data: []byte(`this file should be ignored`),
		},
		filepath.Join(
			config.ServicesDisabledDir,
			"service_b",
			config.TemplatesDisabledDir,
			"ignored_directory",
			"ignored_file3.txt",
		): &fstest.MapFile{
			Data: []byte(`this file should be ignored`),
		},
	}
}
