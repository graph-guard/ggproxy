package config_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/gqt"
	"github.com/stretchr/testify/require"
)

type TestOK struct {
	Filesystem fstest.MapFS
	DirPath    string
	Expect     *config.Config
}

type TestError struct {
	Filesystem fstest.MapFS
	Check      func(*testing.T, error)
}

func TestReadConfig(t *testing.T) {
	validFS, validFSDirPath, conf := validFS()
	for _, td := range []TestOK{
		{
			Filesystem: validFS,
			DirPath:    validFSDirPath,
			Expect:     conf,
		},
	} {
		t.Run("", func(t *testing.T) {
			c, err := config.ReadConfig(td.Filesystem, td.DirPath)
			require.NoError(t, err)
			require.Equal(t, td.Expect, c)
		})
	}
}

func TestReadConfigDefaultMaxReqBodySize(t *testing.T) {
	validFS, validFSDirPath, conf := validFS()

	validFS[filepath.Join(
		validFSDirPath,
		config.ServerConfigFile1,
	)] = &fstest.MapFile{
		Data: lines(
			`proxy:`,
			`  host: localhost:443`,
			`  tls:`,
			`    cert-file: proxy.cert`,
			`    key-file: proxy.key`,
			`  # max-request-body-size: 1234`,
			`api:`,
			`  host: localhost:3000`,
			`  tls:`,
			`    cert-file: api.cert`,
			`    key-file: api.key`,
		),
	}

	conf.Proxy.MaxReqBodySizeBytes = config.DefaultMaxReqBodySize

	for _, td := range []TestOK{
		{
			Filesystem: validFS,
			DirPath:    validFSDirPath,
			Expect:     conf,
		},
	} {
		t.Run("", func(t *testing.T) {
			c, err := config.ReadConfig(td.Filesystem, td.DirPath)
			require.NoError(t, err)
			require.Equal(t, td.Expect, c)
		})
	}
}

func TestReadConfigErrorMissingServerConfig(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServerConfigFile1,
	)
	delete(fs, p)
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
	}, err)
}

func TestReadConfigErrorMalformedServerConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs, path, _ := validFS()
			p := filepath.Join(
				path,
				config.ServiceConfigFile1,
			)
			fs[p].Data = lines(
				"not a valid config.yaml",
			)
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorIllegal{
				FilePath: p,
				Feature:  "syntax",
				Message: "yaml: unmarshal errors:\n  " +
					"line 1: cannot unmarshal !!str `not a v...` " +
					"into config.serverConfig",
			}, err)
		})
	}
}

func TestReadConfigErrorMissingProxyHostConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs, path, _ := validFS()
			p := filepath.Join(
				path,
				config.ServerConfigFile1,
			)
			fs[p].Data = lines(
				`proxy:`,
				`  host: `,
			)
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorMissing{
				FilePath: p,
				Feature:  "proxy.host",
			}, err)
		})
	}
}

func TestReadConfigErrorMissingAPIHostConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs, path, _ := validFS()
			p := filepath.Join(
				path,
				config.ServerConfigFile1,
			)
			fs[p].Data = lines(
				`proxy:`,
				`  host: localhost:8080`,
				`api:`,
				`  host: `,
			)
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorMissing{
				FilePath: p,
				Feature:  "api.host",
			}, err)
		})
	}
}

func TestReadConfigErrorMissingProxyTLSCert(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServerConfigFile1,
	)
	fs[p].Data = lines(
		`proxy:`,
		`  host: localhost:8080`,
		`  tls:`,
		`    key-file: proxy.key`,
		`api:`,
		`  host: localhost:9090`,
	)
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "proxy.tls.cert-file",
	}, err)
}

func TestReadConfigErrorMissingProxyTLSKey(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServerConfigFile1,
	)
	fs[p].Data = lines(
		`proxy:`,
		`  host: localhost:8080`,
		`  tls:`,
		`    cert-file: proxy.cert`,
		`api:`,
		`  host: localhost:9090`,
	)
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "proxy.tls.key-file",
	}, err)
}

func TestReadConfigErrorMissingAPITLSCert(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServerConfigFile1,
	)
	fs[p].Data = lines(
		`proxy:`,
		`  host: localhost:8080`,
		`api:`,
		`  host: localhost:9090`,
		`  tls:`,
		`    key-file: api.key`,
	)
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "api.tls.cert-file",
	}, err)
}

func TestReadConfigErrorMissingAPITLSKey(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServerConfigFile1,
	)
	fs[p].Data = lines(
		`proxy:`,
		`  host: localhost:8080`,
		`api:`,
		`  host: localhost:9090`,
		`  tls:`,
		`    cert-file: api.cert`,
	)
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "api.tls.key-file",
	}, err)
}

func TestReadConfigErrorIllegalProxyMaxReqBodySize(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServerConfigFile1,
	)
	fs[p].Data = lines(
		`proxy:`,
		`  host: localhost:8080`,
		`  max-request-body-size: 255`,
		`api:`,
		`  host: localhost:9090`,
	)
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: p,
		Feature:  "proxy.max-request-body-size",
		Message: fmt.Sprintf(
			"maximum request body size should not be smaller than %d B",
			config.MinReqBodySize,
		),
	}, err)
}

func TestReadConfigErrorMissingConfig(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs, path := minValidFS()
			p := filepath.Join(
				path,
				m,
				"service_a",
				"irrelevant_file.txt",
			)
			fs[p] = &fstest.MapFile{
				Data: []byte(`this file only keeps the directory`),
			}
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorMissing{
				FilePath: filepath.Join(
					path, m, "service_a", config.ServiceConfigFile1,
				),
			}, err)
		})
	}
}

func TestReadConfigErrorMalformedMetadata(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServicesEnabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"template_a.gqt",
	)
	fs[p] = &fstest.MapFile{
		Data: lines(
			"---",
			"malformed metadata",
			"---",
			`query { foo }`,
		),
	}
	err := testError(t, fs, path)
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
	fs, path := minValidFS()
	fs[filepath.Join(
		path,
		config.ServerConfigFile1,
	)] = &fstest.MapFile{
		Data: lines(
			`proxy:`,
			`  host: localhost:8080/`,
		),
	}
	fs[filepath.Join(
		path,
		config.ServerConfigFile2,
	)] = &fstest.MapFile{
		Data: lines(
			`proxy:`,
			`  host: localhost:9090/`,
		),
	}
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorConflict{Items: []string{
		config.ServerConfigFile1,
		config.ServerConfigFile2,
	}}, err)
}

func TestReadConfigErrorDuplicateTemplate(t *testing.T) {
	fs, path, _ := validFS()
	fs[filepath.Join(
		path,
		config.ServicesEnabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"dup.gqt",
	)] = &fstest.MapFile{
		Data: []byte(`query { duplicate }`),
	}
	fs[filepath.Join(
		path,
		config.ServicesEnabledDir,
		"service_a",
		config.TemplatesDisabledDir,
		"dup.gqt",
	)] = &fstest.MapFile{
		Data: []byte(`query { duplicate }`),
	}
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorConflict{
		Items: []string{
			"templates_enabled/dup",
			"templates_disabled/dup",
		},
	}, err)
}

func TestReadConfigErrorDuplicateService(t *testing.T) {
	fs, path := minValidFS()
	fs[filepath.Join(
		path,
		config.ServicesEnabledDir,
		"service_a",
		config.ServiceConfigFile1,
	)] = &fstest.MapFile{
		Data: lines(
			`path: /`,
			`forward_url: http://localhost:8080/`,
		),
	}
	fs[filepath.Join(
		path,
		config.ServicesDisabledDir,
		"service_a",
		config.ServiceConfigFile1,
	)] = &fstest.MapFile{
		Data: lines(
			`path: /`,
			`forward_url: http://localhost:8080/`,
		),
	}
	err := testError(t, fs, path)
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
			fs, path := minValidFS()
			fs[filepath.Join(
				path,
				m,
				"service_a",
				config.ServiceConfigFile1,
			)] = &fstest.MapFile{
				Data: lines(
					`path: /`,
					`forward_url: http://localhost:8080/`,
				),
			}
			fs[filepath.Join(
				path,
				m,
				"service_a",
				config.ServiceConfigFile2,
			)] = &fstest.MapFile{
				Data: lines(
					`path: /`,
					`forward_url: http://localhost:9090/`,
				),
			}
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorConflict{Items: []string{
				filepath.Join(path, m, "service_a", config.ServiceConfigFile1),
				filepath.Join(path, m, "service_a", config.ServiceConfigFile2),
			}}, err)
		})
	}
}

func TestReadConfigErrorMissingPath(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs, path := minValidFS()
			fs[filepath.Join(
				path,
				m,
				"service_a",
				config.ServiceConfigFile1,
			)] = &fstest.MapFile{
				Data: []byte(`forward_reduced: true`),
			}
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorMissing{
				FilePath: filepath.Join(
					path, m, "service_a", config.ServiceConfigFile1,
				),
				Feature: "path",
			}, err)
		})
	}
}

func TestReadConfigErrorMissingForwardURL(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs, path := minValidFS()
			fs[filepath.Join(
				path,
				m,
				"service_a",
				config.ServiceConfigFile1,
			)] = &fstest.MapFile{
				Data: []byte(`path: /`),
			}
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorMissing{
				FilePath: filepath.Join(
					path, m, "service_a", config.ServiceConfigFile1,
				),
				Feature: "forward_url",
			}, err)
		})
	}
}

func TestReadConfigErrorInvalidPath(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs, path := minValidFS()
			p := filepath.Join(
				path,
				m,
				"service_a",
				config.ServiceConfigFile1,
			)
			fs[p] = &fstest.MapFile{
				Data: lines(
					`path: invalid_path`,
					`forward_url: http://localhost:8080/`,
				),
			}
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorIllegal{
				FilePath: p,
				Feature:  "path",
				Message:  `path is not starting with /`,
			}, err)
		})
	}
}

func TestReadConfigErrorInvalidForwardURLInvalidScheme(t *testing.T) {
	for _, m := range [2]string{
		config.ServicesDisabledDir, config.ServicesEnabledDir,
	} {
		t.Run(m, func(t *testing.T) {
			fs, path := minValidFS()
			p := filepath.Join(
				path,
				m,
				"service_a",
				config.ServiceConfigFile1,
			)
			fs[p] = &fstest.MapFile{
				Data: lines(
					`path: /`,
					`forward_url: localhost:8080`,
				),
			}
			err := testError(t, fs, path)
			require.Equal(t, &config.ErrorIllegal{
				FilePath: p,
				Feature:  "forward_url",
				Message:  `protocol is not supported or undefined`,
			}, err)
		})
	}
}

func TestReadConfigErrorInvalidTemplate(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServicesDisabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"template_invalid.gqt",
	)
	fs[p] = &fstest.MapFile{
		Data: []byte(`invalid { template }`),
	}
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: p,
		Feature:  "template",
		Message:  `error at 0: unexpected definition`,
	}, err)
}

func TestReadConfigErrorInvalidTemplateID(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServicesDisabledDir,
		"service_a",
		config.TemplatesEnabledDir,
		"template-invalid#.gqt",
	)
	fs[p] = &fstest.MapFile{
		Data: []byte(`invalid { template }`),
	}
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: p,
		Feature:  "id",
		Message:  `contains illegal character at index 16`,
	}, err)
}

func TestReadConfigErrorInvalidServiceID(t *testing.T) {
	fs, path, _ := validFS()
	fs[filepath.Join(
		path,
		config.ServicesDisabledDir,
		"service_#1",
		config.ServiceConfigFile1,
	)] = &fstest.MapFile{
		Data: []byte(`forward_url: localhost:8080/`),
	}
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(
			path,
			config.ServicesDisabledDir,
			"service_#1",
		),
		Feature: "id",
		Message: `contains illegal character at index 8`,
	}, err)
}

func TestReadConfigErrorMalformedConfig(t *testing.T) {
	fs, path, _ := validFS()
	p := filepath.Join(
		path,
		config.ServicesEnabledDir,
		"service_a",
		config.ServiceConfigFile1,
	)
	fs[p] = &fstest.MapFile{
		Data: lines(
			`malformed yaml`,
		),
	}
	err := testError(t, fs, path)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: p,
		Feature:  "syntax",
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
	path string,
) error {
	t.Helper()
	c, err := config.ReadConfig(filesystem, path)
	require.Error(t, err)
	require.Nil(t, c)
	return err
}

func minValidFS() (filesystem fstest.MapFS, path string) {
	path = "testconfig"
	filesystem = fstest.MapFS{
		filepath.Join(
			path,
			config.ServerConfigFile1,
		): &fstest.MapFile{
			Data: lines(
				`proxy:`,
				`  host: localhost:443`,
			),
		},

		// Irrelevant files
		filepath.Join(
			"irrelevant_dir",
			"irrelevant_file.txt",
		): &fstest.MapFile{
			Data: lines(
				`this file is irrelevant and exists only for the purposes`,
				`of testing function ReadConfig.`,
			),
		},
		filepath.Join(
			"irrelevant_file.txt",
		): &fstest.MapFile{
			Data: lines(
				`this file is irrelevant and exists only for the purposes`,
				`of testing function ReadConfig.`,
			),
		},
	}
	return
}

func validFS() (filesystem fstest.MapFS, path string, conf *config.Config) {
	path = "testconfigroot"
	filesystem = fstest.MapFS{
		// Relevant files
		filepath.Join(
			path,
			config.ServerConfigFile1,
		): &fstest.MapFile{
			Data: lines(
				`proxy:`,
				`  host: localhost:443`,
				`  tls:`,
				`    cert-file: proxy.cert`,
				`    key-file: proxy.key`,
				fmt.Sprintf(
					`  max-request-body-size: %d`,
					config.MinReqBodySize+256,
				),
				`api:`,
				`  host: localhost:3000`,
				`  tls:`,
				`    cert-file: api.cert`,
				`    key-file: api.key`,
			),
		},

		filepath.Join(
			path,
			config.ServicesEnabledDir,
			"service_a",
			config.ServiceConfigFile1,
		): &fstest.MapFile{
			Data: lines(
				`path: "/path"`,
				`forward_url: "http://localhost:8080/path"`,
				`forward_reduced: true`,
			),
		},
		filepath.Join(
			path,
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
			path,
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
			path,
			config.ServicesDisabledDir,
			"service_b",
			config.ServiceConfigFile1,
		): &fstest.MapFile{
			Data: lines(
				`path: /`,
				`forward_url: "http://localhost:9090/"`,
			),
		},
		filepath.Join(
			path,
			config.ServicesDisabledDir,
			"service_b",
			config.TemplatesDisabledDir,
			"template_c.gqt",
		): &fstest.MapFile{
			Data: []byte(`query { maz }`),
		},

		// Irrelevant files
		filepath.Join(
			"irrelevant_dir",
			"irrelevant_file.txt",
		): &fstest.MapFile{
			Data: lines(
				`this file is irrelevant and exists only for the purposes`,
				`of testing function ReadConfig.`,
			),
		},
		filepath.Join(
			"irrelevant_file.txt",
		): &fstest.MapFile{
			Data: lines(
				`this file is irrelevant and exists only for the purposes`,
				`of testing function ReadConfig.`,
			),
		},

		// Ignored files
		filepath.Join(
			path,
			config.ServicesDisabledDir,
			"ignored_file1.txt",
		): &fstest.MapFile{
			Data: []byte(`this file should be ignored`),
		},
		filepath.Join(
			path,
			config.ServicesDisabledDir,
			"service_b",
			config.TemplatesDisabledDir,
			"ignored_file2.txt",
		): &fstest.MapFile{
			Data: []byte(`this file should be ignored`),
		},
		filepath.Join(
			path,
			config.ServicesDisabledDir,
			"service_b",
			config.TemplatesDisabledDir,
			"ignored_directory",
			"ignored_file3.txt",
		): &fstest.MapFile{
			Data: []byte(`this file should be ignored`),
		},
	}

	conf = &config.Config{
		Proxy: config.ProxyServerConfig{
			Host: "localhost:443",
			TLS: config.TLS{
				CertFile: "proxy.cert",
				KeyFile:  "proxy.key",
			},
			MaxReqBodySizeBytes: config.MinReqBodySize + 256,
		},
		API: &config.APIServerConfig{
			Host: "localhost:3000",
			TLS: config.TLS{
				CertFile: "api.cert",
				KeyFile:  "api.key",
			},
		},
		ServicesEnabled: []*config.Service{
			{
				ID:             "service_a",
				Path:           "/path",
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
				Path:           "/",
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
	}

	return
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
