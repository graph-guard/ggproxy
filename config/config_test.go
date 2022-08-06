package config_test

import (
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
				Host: "localhost:443",
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

func TestReadConfigErrorMissingServerConfig(t *testing.T) {
	fs := validFS()
	delete(fs, config.ServerConfigFile1)
	err := testError(t, fs)
	require.Equal(t, &config.ErrorMissing{
		FilePath: config.ServerConfigFile1,
	}, err)
}

func TestReadConfigErrorMalformedServerConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs := validFS()
			fs[config.ServiceConfigFile1].Data = lines(
				"not a valid config.yaml",
			)
			err := testError(t, fs)
			require.IsType(t, &config.ErrorIllegal{}, err)
			e, _ := err.(*config.ErrorIllegal)
			require.Equal(t, &config.ErrorIllegal{
				FilePath: config.ServiceConfigFile1,
				Message: "yaml: unmarshal errors:\n  " +
					"line 1: cannot unmarshal !!str `not a v...` " +
					"into config.serverConfig",
			}, e)
		})
	}
}

func TestReadConfigErrorMissingHostConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs := validFS()
			fs[config.ServerConfigFile1].Data = lines(
				"host: ",
			)
			err := testError(t, fs)
			require.Equal(t, &config.ErrorMissing{
				FilePath: config.ServerConfigFile1,
				Feature:  "host",
			}, err)
		})
	}
}

func TestReadConfigErrorMissingConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs := minValidFS()
			fs[filepath.Join(
				m,
				"service_a",
				"irrelevant_file.txt",
			)] = &fstest.MapFile{
				Data: []byte(`this file only keeps the directory`),
			}
			err := testError(t, fs)
			require.Equal(t, &config.ErrorMissing{
				FilePath: filepath.Join(m, "service_a", config.ServiceConfigFile1),
			}, err)
		})
	}
}

func TestReadConfigErrorMalformedMetadata(t *testing.T) {
	f := validFS()
	p := filepath.Join(
		config.ServicesEnabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"template_a.gqt",
	)
	f[p] = &fstest.MapFile{
		Data: lines(
			"---",
			"malformed metadata",
			"---",
			`query { foo }`,
		),
	}
	err := testError(t, f)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: p,
		Feature:  "metadata",
		Message: "decoding yaml: yaml: " +
			"unmarshal errors:\n  " +
			"line 1: cannot unmarshal !!str `malform...` " +
			"into metadata.Metadata",
	}, err)
}

func TestReadConfigErrorDuplicateServerConfig(t *testing.T) {
	fs := minValidFS()
	fs[config.ServerConfigFile1] = &fstest.MapFile{
		Data: []byte(`host: localhost:8080/`),
	}
	fs[config.ServerConfigFile2] = &fstest.MapFile{
		Data: []byte(`host: localhost:9090/`),
	}
	err := testError(t, fs)
	require.Equal(t, &config.ErrorConflict{Items: []string{
		config.ServerConfigFile1,
		config.ServerConfigFile2,
	}}, err)
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
	require.Equal(t, &config.ErrorConflict{
		Items: []string{
			"templates_enabled/dup",
			"templates_disabled/dup",
		},
	}, err)
}

func TestReadConfigErrorDuplicateService(t *testing.T) {
	fs := minValidFS()
	fs[filepath.Join(
		config.ServicesEnabledDir,
		"service_a",
		config.ServiceConfigFile1,
	)] = &fstest.MapFile{
		Data: []byte(`forward_url: localhost:8080/`),
	}
	fs[filepath.Join(
		config.ServicesDisabledDir,
		"service_a",
		config.ServiceConfigFile1,
	)] = &fstest.MapFile{
		Data: []byte(`forward_url: localhost:8080/`),
	}
	err := testError(t, fs)
	require.Equal(t, &config.ErrorConflict{Items: []string{
		filepath.Join(config.ServicesEnabledDir, "service_a"),
		filepath.Join(config.ServicesDisabledDir, "service_a"),
	}}, err)
}

func TestReadConfigErrorDuplicateServiceConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs := minValidFS()
			fs[filepath.Join(
				m,
				"service_a",
				config.ServiceConfigFile1,
			)] = &fstest.MapFile{
				Data: []byte(`forward_url: localhost:8080`),
			}
			fs[filepath.Join(
				m,
				"service_a",
				config.ServiceConfigFile2,
			)] = &fstest.MapFile{
				Data: []byte(`forward_url: localhost:9090`),
			}
			err := testError(t, fs)
			require.Equal(t, &config.ErrorConflict{Items: []string{
				filepath.Join(m, "service_a", config.ServiceConfigFile1),
				filepath.Join(m, "service_a", config.ServiceConfigFile2),
			}}, err)
		})
	}
}

func TestReadConfigErrorMissingForwardURL(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs := minValidFS()
			fs[filepath.Join(
				m,
				"service_a",
				config.ServiceConfigFile1,
			)] = &fstest.MapFile{
				Data: []byte(`forward_reduced: true`),
			}
			err := testError(t, fs)
			require.Equal(t, &config.ErrorMissing{
				FilePath: filepath.Join(
					m, "service_a", config.ServiceConfigFile1,
				),
				Feature: "forward_url",
			}, err)
		})
	}
}

func TestReadConfigErrorInvalidForwardURL(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs := minValidFS()
			fs[filepath.Join(
				m,
				"service_a",
				config.ServiceConfigFile1,
			)] = &fstest.MapFile{
				Data: []byte(`forward_url: not_a_url.`),
			}
			err := testError(t, fs)
			require.Equal(t, &config.ErrorIllegal{
				FilePath: filepath.Join(
					m, "service_a", config.ServiceConfigFile1,
				),
				Feature: "forward_url",
				Message: `parse "not_a_url.": invalid URI for request`,
			}, err)
		})
	}
}

func TestReadConfigErrorInvalidTemplate(t *testing.T) {
	fs := validFS()
	fs[filepath.Join(
		config.ServicesDisabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"template_invalid.gqt",
	)] = &fstest.MapFile{
		Data: []byte(`invalid { template }`),
	}
	err := testError(t, fs)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(
			config.ServicesDisabledDir,
			"service_a",
			config.TemplatesEnabledDir,
			"template_invalid.gqt",
		),
		Feature: "template",
		Message: `error at 0: unexpected definition`,
	}, err)
}

func TestReadConfigErrorInvalidTemplateID(t *testing.T) {
	fs := validFS()
	fs[filepath.Join(
		config.ServicesDisabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"template-invalid#.gqt",
	)] = &fstest.MapFile{
		Data: []byte(`invalid { template }`),
	}
	err := testError(t, fs)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(
			config.ServicesDisabledDir,
			"service_a",
			config.TemplatesEnabledDir,
			"template-invalid#.gqt",
		),
		Feature: "id",
		Message: `contains illegal character at index 16`,
	}, err)
}

func TestReadConfigErrorInvalidServiceID(t *testing.T) {
	fs := validFS()
	fs[filepath.Join(
		config.ServicesDisabledDir,
		"service_#1",
		config.ServiceConfigFile1,
	)] = &fstest.MapFile{
		Data: []byte(`forward_url: localhost:8080/`),
	}
	err := testError(t, fs)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(
			config.ServicesDisabledDir,
			"service_#1",
		),
		Feature: "id",
		Message: `contains illegal character at index 8`,
	}, err)
}

func TestReadConfigErrorMalformedConfig(t *testing.T) {
	fs := validFS()
	p := filepath.Join(
		config.ServicesEnabledDir,
		"service_a",
		config.ServiceConfigFile1,
	)
	fs[p] = &fstest.MapFile{
		Data: lines(
			`malformed yaml`,
		),
	}
	err := testError(t, fs)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: p,
		Message: "yaml: unmarshal errors:\n  " +
			"line 1: cannot unmarshal !!str `malform...` " +
			"into config.serviceConfig",
	}, err)
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

func minValidFS() fstest.MapFS {
	return fstest.MapFS{
		config.ServerConfigFile1: &fstest.MapFile{
			Data: lines(
				`host: localhost:443`,
			),
		},
	}
}

func validFS() fstest.MapFS {
	return fstest.MapFS{
		config.ServerConfigFile1: &fstest.MapFile{
			Data: lines(
				`host: localhost:443`,
			),
		},

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

func TestErrorString(t *testing.T) {
	for _, td := range []struct {
		input  error
		expect string
	}{
		{
			input: config.ErrorMissing{
				FilePath: "path/to/file.txt",
				Feature:  "some_feature",
			},
			expect: "missing some_feature in path/to/file.txt",
		},
		{
			input: config.ErrorMissing{
				FilePath: "path/to/file.txt",
			},
			expect: "missing path/to/file.txt",
		},
		{
			input: config.ErrorIllegal{
				FilePath: "path/to/file.txt",
				Feature:  "some_feature",
				Message:  "some message",
			},
			expect: "illegal some_feature in path/to/file.txt: some message",
		},
		{
			input: config.ErrorConflict{
				Items: []string{
					"path/to/file_a.txt",
					"path/to/file_b.txt",
				},
			},
			expect: "conflict between: path/to/file_a.txt, path/to/file_b.txt",
		},
	} {
		t.Run("", func(t *testing.T) {
			require.Equal(t, td.expect, td.input.Error())
		})
	}
}
