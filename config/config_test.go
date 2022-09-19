package config_test

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/utilities/container/hamap"
	"github.com/graph-guard/gqt"
	"github.com/stretchr/testify/require"
)

type TestOK struct {
	Path   string
	Expect *config.Config
}

type TestError struct {
	Filesystem fstest.MapFS
	Check      func(*testing.T, error)
}

var ServerConfigFileName = "config.yml"

func TestReadConfig(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		for _, td := range []TestOK{
			{
				Path:   filepath.Join(path, ServerConfigFileName),
				Expect: conf,
			},
		} {
			t.Run("", func(t *testing.T) {
				c, err := config.New(td.Path)
				require.NoError(t, err)
				require.True(t, td.Expect.Equal(c))
			})
		}
	})
}

func TestReadConfigDefaultMaxReqBodySize(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		err := createFiles(map[string]any{
			ServerConfigFileName: lines(
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
				`all-services: all-services`,
				`enabled-services: enabled-services`,
			),
		}, nil, path)
		require.NoError(t, err)
		conf.Proxy.MaxReqBodySizeBytes = config.DefaultMaxReqBodySize
		for _, td := range []TestOK{
			{
				Path:   filepath.Join(path, ServerConfigFileName),
				Expect: conf,
			},
		} {
			t.Run("", func(t *testing.T) {
				c, err := config.New(td.Path)
				require.NoError(t, err)
				require.True(t, td.Expect.Equal(c))
			})
		}
	})
}

func TestReadConfigErrorMissingServerConfig(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := os.Remove(p)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestReadConfigErrorMalformedServerConfig(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			ServerConfigFileName: lines("not a valid config"),
		}, nil, path)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Equal(t, &config.ErrorIllegal{
			FilePath: p,
			Feature:  "syntax",
			Message: "yaml: unmarshal errors:\n  " +
				"line 1: cannot unmarshal !!str `not a v...` " +
				"into config.serverConfig",
		}, err)
	})
}

func TestReadConfigErrorMissingProxyHostConfig(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			ServerConfigFileName: lines(
				`proxy:`,
				`  host: `,
			),
		}, nil, path)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Equal(t, &config.ErrorMissing{
			FilePath: p,
			Feature:  "proxy.host",
		}, err)
	})
}

func TestReadConfigErrorMissingAPIHostConfig(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			ServerConfigFileName: lines(
				`proxy:`,
				`  host: localhost:8080`,
				`api:`,
				`  host: `,
			),
		}, nil, path)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Equal(t, &config.ErrorMissing{
			FilePath: p,
			Feature:  "api.host",
		}, err)
	})
}

func TestReadConfigErrorMissingProxyTLSCert(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			ServerConfigFileName: lines(
				`proxy:`,
				`  host: localhost:8080`,
				`  tls:`,
				`    key-file: proxy.key`,
				`api:`,
				`  host: localhost:9090`,
			),
		}, nil, path)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Equal(t, &config.ErrorMissing{
			FilePath: p,
			Feature:  "proxy.tls.cert-file",
		}, err)
	})
}

func TestReadConfigErrorMissingProxyTLSKey(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			ServerConfigFileName: lines(
				`proxy:`,
				`  host: localhost:8080`,
				`  tls:`,
				`    cert-file: proxy.cert`,
				`api:`,
				`  host: localhost:9090`,
			),
		}, nil, path)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Equal(t, &config.ErrorMissing{
			FilePath: p,
			Feature:  "proxy.tls.key-file",
		}, err)
	})
}

func TestReadConfigErrorMissingAPITLSCert(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			ServerConfigFileName: lines(
				`proxy:`,
				`  host: localhost:8080`,
				`api:`,
				`  host: localhost:9090`,
				`  tls:`,
				`    key-file: api.key`,
			),
		}, nil, path)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Equal(t, &config.ErrorMissing{
			FilePath: p,
			Feature:  "api.tls.cert-file",
		}, err)
	})
}

func TestReadConfigErrorMissingAPITLSKey(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			ServerConfigFileName: lines(
				`proxy:`,
				`  host: localhost:8080`,
				`api:`,
				`  host: localhost:9090`,
				`  tls:`,
				`    cert-file: api.cert`,
			),
		}, nil, path)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Equal(t, &config.ErrorMissing{
			FilePath: p,
			Feature:  "api.tls.key-file",
		}, err)
	})
}

func TestReadConfigErrorIllegalProxyMaxReqBodySize(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			ServerConfigFileName: lines(
				`proxy:`,
				`  host: localhost:8080`,
				`  max-request-body-size: 255`,
				`api:`,
				`  host: localhost:9090`,
			),
		}, nil, path)
		require.NoError(t, err)
		c, err := config.New(p)
		require.Nil(t, c)
		require.Equal(t, &config.ErrorIllegal{
			FilePath: p,
			Feature:  "proxy.max-request-body-size",
			Message: fmt.Sprintf(
				"maximum request body size should not be smaller than %d B",
				config.MinReqBodySize,
			),
		}, err)
	})
}

func TestReadServiceConfigErrorMissingConfig(t *testing.T) {
	minValidFS(func(path string) {
		p := filepath.Join(path, "all-services", "a.yml")
		err := createFiles(map[string]any{
			"all-services": map[string]any{
				"irrelevant_file.txt": []byte(`this file only keeps the directory`),
			},
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(p)
		require.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestReadConfigErrorMalformedMetadata(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join("all-templates", "a", "a.gqt")
		err := createFiles(map[string]any{
			p: lines(
				"---",
				"malformed metadata",
				"---",
				`query { foo }`,
			),
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(filepath.Join(path, ServerConfigFileName))
		require.Equal(t, &config.ErrorIllegal{
			FilePath: filepath.Join(path, p),
			Feature:  "metadata",
			Message: "decoding yaml: yaml: " +
				"unmarshal errors:\n  " +
				"line 1: cannot unmarshal !!str `malform...` " +
				"into metadata.Metadata",
		}, err)
	})
}

func TestReadConfigErrorDuplicateTemplate(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		t1 := filepath.Join("all-templates", "a", "d1.gqt")
		t2 := filepath.Join("all-templates", "a", "d2.gqt")
		err := createFiles(map[string]any{
			t1: []byte(
				`query { duplicate }`,
			),
			t2: []byte(
				`query { duplicate }`,
			),
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(filepath.Join(path, ServerConfigFileName))
		require.Equal(t, &config.ErrorDuplicate{
			Original:  filepath.Join(path, t1),
			Duplicate: filepath.Join(path, t2),
		}, err)
	})
}

func TestReadConfigErrorDuplicateService(t *testing.T) {
	minValidFS(func(path string) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			"all-services": map[string]any{
				"a.yml": lines(
					`path: /`,
					`forward-url: http://localhost:8080/`,
					`all-templates: ../all-templates/a`,
					`enabled-templates: ../enabled-templates/a`,
				),
				"b.yml": lines(
					`path: /`,
					`forward-url: http://localhost:8080/`,
					`all-templates: ../all-templates/a`,
					`enabled-templates: ../enabled-templates/a`,
				),
			},
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(p)
		require.Equal(t, &config.ErrorDuplicate{
			Original:  filepath.Join(path, "all-services", "a.yml"),
			Duplicate: filepath.Join(path, "all-services", "b.yml"),
		}, err)
	})
}

func TestReadConfigErrorMissingPath(t *testing.T) {
	minValidFS(func(path string) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			"all-services": map[string]any{
				"a.yml": lines(
					`forward-url: http://localhost:8080/`,
				),
			},
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(p)
		require.Equal(t, &config.ErrorMissing{
			FilePath: filepath.Join(
				path, "all-services", "a.yml",
			),
			Feature: "path",
		}, err)
	})

}

func TestReadConfigErrorMissingForwardURL(t *testing.T) {
	minValidFS(func(path string) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			"all-services": map[string]any{
				"a.yml": lines(
					`path: /`,
				),
			},
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(p)
		require.Equal(t, &config.ErrorMissing{
			FilePath: filepath.Join(
				path, "all-services", "a.yml",
			),
			Feature: "forward-url",
		}, err)
	})
}

func TestReadConfigErrorInvalidPath(t *testing.T) {
	minValidFS(func(path string) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			"all-services": map[string]any{
				"a.yml": lines(
					`path: invalid_path`,
					`forward-url: http://localhost:8080/`,
				),
			},
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(p)
		require.Equal(t, &config.ErrorIllegal{
			FilePath: filepath.Join(path, "all-services", "a.yml"),
			Feature:  "path",
			Message:  `path is not starting with /`,
		}, err)
	})
}

func TestReadConfigErrorInvalidForwardURLInvalidScheme(t *testing.T) {
	minValidFS(func(path string) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			"all-services": map[string]any{
				"a.yml": lines(
					`path: /`,
					`forward-url: localhost:8080`,
				),
			},
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(p)
		require.Equal(t, &config.ErrorIllegal{
			FilePath: filepath.Join(path, "all-services", "a.yml"),
			Feature:  "forward-url",
			Message:  `protocol is not supported or undefined`,
		}, err)
	})
}

func TestReadConfigErrorInvalidTemplate(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join("all-templates", "a", "invalid_template.gqt")
		err := createFiles(map[string]any{
			p: []byte(
				`invalid { template }`,
			),
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(filepath.Join(path, ServerConfigFileName))
		require.Equal(t, &config.ErrorIllegal{
			FilePath: filepath.Join(path, p),
			Feature:  "template",
			Message:  `error at 0: unexpected definition`,
		}, err)
	})
}

func TestReadConfigErrorInvalidTemplateID(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join("all-templates", "a", "invalid_template#.gqt")
		err := createFiles(map[string]any{
			p: []byte(
				`invalid { template }`,
			),
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(filepath.Join(path, ServerConfigFileName))
		require.Equal(t, &config.ErrorIllegal{
			FilePath: filepath.Join(path, p),
			Feature:  "id",
			Message:  `contains illegal character at index 16`,
		}, err)
	})
}

func TestReadConfigErrorInvalidServiceID(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			"all-services": map[string]any{
				"a#.yml": lines(
					`path: /`,
					`forward-url: http://localhost:8080/`,
					`all-templates: ../all-templates/a`,
					`enabled-templates: ../enabled-templates/a`,
				),
			},
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(p)
		require.Equal(t, &config.ErrorIllegal{
			FilePath: filepath.Join(
				path,
				"all-services",
				"a#.yml",
			),
			Feature: "id",
			Message: `contains illegal character at index 1`,
		}, err)
	})
}

func TestReadConfigErrorMalformedConfig(t *testing.T) {
	validFS(func(path string, conf *config.Config) {
		p := filepath.Join(path, ServerConfigFileName)
		err := createFiles(map[string]any{
			"all-services": map[string]any{
				"a.yml": lines(
					`malformed yaml`,
				),
			},
		}, nil, path)
		require.NoError(t, err)
		_, err = config.New(p)
		require.Equal(t, &config.ErrorIllegal{
			FilePath: filepath.Join(path, "all-services", "a.yml"),
			Feature:  "syntax",
			Message: "yaml: unmarshal errors:\n  " +
				"line 1: cannot unmarshal !!str `malform...` " +
				"into config.serviceConfig",
		}, err)
	})
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
			input: config.ErrorDuplicate{
				Original:  "path/to/file_a.txt",
				Duplicate: "path/to/file_b.txt",
			},
			expect: "path/to/file_b.txt is a duplicate of path/to/file_a.txt",
		},
	} {
		t.Run("", func(t *testing.T) {
			require.Equal(t, td.expect, td.input.Error())
		})
	}
}

func minValidFS(fn func(path string)) {
	base, err := os.MkdirTemp("", "ggproxy-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(base)

	dirs := map[string]any{
		"all-services":     nil,
		"enabled-services": nil,
		"all-templates": map[string]any{
			"a": nil,
			"b": nil,
		},
		"enabled-templates": map[string]any{
			"a": nil,
			"b": nil,
		},
		"irrelevant-dir": nil,
	}
	files := map[string]any{
		ServerConfigFileName: lines(
			`proxy:`,
			`  host: localhost:443`,
			`all-services: all-services`,
			`enabled-services: enabled-services`,
		),
		"irrelevant-file.txt": lines(
			`this file is irrelevant and exists only for the purposes`,
			`of testing function ReadConfig.`,
		),
		"irrelevant-dir": map[string]any{
			"irrelevant_file.txt": lines(
				`this file is irrelevant and exists only for the purposes`,
				`of testing function ReadConfig.`,
			),
		},
	}

	var hashes = make(map[string][]byte)

	if err := createDirs(dirs, base); err != nil {
		panic(err)
	}
	if err := createFiles(files, hashes, base); err != nil {
		panic(err)
	}

	fn(base)
}

func validFS(fn func(path string, conf *config.Config)) {
	base, err := os.MkdirTemp("", "ggproxy-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(base)

	dirs := map[string]any{
		"all-services":     nil,
		"enabled-services": nil,
		"all-templates": map[string]any{
			"a": nil,
			"b": nil,
		},
		"enabled-templates": map[string]any{
			"a": nil,
			"b": nil,
		},
		"irrelevant-dir": nil,
	}
	files := map[string]any{
		ServerConfigFileName: lines(
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
			`all-services: all-services`,
			`enabled-services: enabled-services`,
		),
		"all-services": map[string]any{
			"a.yml": lines(
				`path: "/path"`,
				`forward-url: "http://localhost:8080/path"`,
				`forward-reduced: true`,
				`all-templates: "../all-templates/a"`,
				`enabled-templates: "../enabled-templates/a"`,
			),
			"b.yml": lines(
				`path: /`,
				`forward-url: "http://localhost:9090/"`,
				`all-templates: "../all-templates/b"`,
				`enabled-templates: "../enabled-templates/b"`,
			),
			"ignored_file.txt": []byte(`this file should be ignored`),
		},
		"all-templates": map[string]any{
			"a": map[string]any{
				"a.gqt": lines(
					"---",
					`name: "Template A"`,
					"tags:",
					"  - tag_a",
					"---",
					`query { foo }`,
				),
				"b.gqt": lines(
					"---",
					"tags:",
					"  - tag_b1",
					"  - tag_b2",
					"---",
					`query { bar }`,
				),
			},
			"b": map[string]any{
				"c.gqt":            []byte(`query { maz }`),
				"ignored_file.txt": []byte(`this file should be ignored`),
			},
		},
		"irrelevant-file.txt": lines(
			`this file is irrelevant and exists only for the purposes`,
			`of testing function ReadConfig.`,
		),
		"irrelevant-dir": map[string]any{
			"irrelevant_file.txt": lines(
				`this file is irrelevant and exists only for the purposes`,
				`of testing function ReadConfig.`,
			),
		},
	}
	links := map[string]string{
		"all-services/a.yml":               "enabled-services/a.yml",
		"all-services/b.yml":               "enabled-services/b.yml",
		"all-templates/a/a.gqt":            "enabled-templates/a/a.gqt",
		"all-templates/a/b.gqt":            "enabled-templates/a/b.gqt",
		"all-templates/b/c.gqt":            "enabled-templates/b/c.gqt",
		"all-services/ignored-file.txt":    "enabled-services/ignored-file.txt",
		"all-templates/b/ignored-file.txt": "enabled-templates/b/ignored-file.txt",
	}

	var hashes = make(map[string][]byte)

	if err := createDirs(dirs, base); err != nil {
		panic(err)
	}
	if err := createFiles(files, hashes, base); err != nil {
		panic(err)
	}
	if err := createSymlinks(links, base); err != nil {
		panic(err)
	}

	path := ""
	services := hamap.New[[]byte, *config.Service](0, nil)
	serviceATemplates := hamap.New[[]byte, *config.Template](0, nil)
	serviceBTemplates := hamap.New[[]byte, *config.Template](0, nil)
	path = filepath.Join(base, "all-templates", "a", "a.gqt")
	serviceATemplates.Set(hashes[path],
		&config.Template{
			ID:     "a",
			Name:   "Template A",
			Tags:   []string{"tag_a"},
			Source: lines(`query { foo }`),
			Document: gqt.Doc{
				Query: []gqt.Selection{
					gqt.SelectionField{
						Name: "foo",
					},
				},
			},
			Enabled:  true,
			FilePath: path,
		},
	)
	path = filepath.Join(base, "all-templates", "a", "b.gqt")
	serviceATemplates.Set(hashes[path],
		&config.Template{
			ID:     "b",
			Tags:   []string{"tag_b1", "tag_b2"},
			Source: lines(`query { bar }`),
			Document: gqt.Doc{
				Query: []gqt.Selection{
					gqt.SelectionField{
						Name: "bar",
					},
				},
			},
			Enabled:  true,
			FilePath: path,
		},
	)
	path = filepath.Join(base, "all-services", "a.yml")
	services.Set(hashes[path],
		&config.Service{
			ID:               "a",
			Path:             "/path",
			ForwardURL:       "http://localhost:8080/path",
			ForwardReduced:   true,
			Templates:        serviceATemplates,
			TemplatesEnabled: serviceATemplates.Values(),
			Enabled:          true,
			FilePath:         path,
		},
	)
	path = filepath.Join(base, "all-templates", "b", "c.gqt")
	serviceBTemplates.Set(hashes[path],
		&config.Template{
			ID:     "c",
			Source: []byte(`query { maz }`),
			Document: gqt.Doc{
				Query: []gqt.Selection{
					gqt.SelectionField{
						Name: "maz",
					},
				},
			},
			Enabled:  true,
			FilePath: path,
		},
	)
	path = filepath.Join(base, "all-services", "b.yml")
	services.Set(hashes[path],
		&config.Service{
			ID:               "b",
			Path:             "/",
			ForwardURL:       "http://localhost:9090/",
			ForwardReduced:   false,
			Templates:        serviceBTemplates,
			TemplatesEnabled: serviceBTemplates.Values(),
			Enabled:          true,
			FilePath:         path,
		},
	)

	conf := &config.Config{
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
		Services:        services,
		ServicesEnabled: services.Values(),
	}

	fn(base, conf)
}

func createDirs(dirs map[string]any, path string) error {
	for k, v := range dirs {
		p := filepath.Join(path, k)
		if err := os.Mkdir(p, 0775); err != nil {
			return err
		}
		if v != nil {
			switch vt := v.(type) {
			case map[string]any:
				if err := createDirs(vt, p); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func createFiles(files map[string]any, hashes map[string][]byte, path string) error {
	for k, v := range files {
		p := filepath.Join(path, k)
		switch vt := v.(type) {
		case []byte:
			f, err := os.Create(p)
			if err != nil {
				return err
			} else {
				if _, err := f.Write(vt); err != nil {
					return err
				}
			}
			if hashes != nil {
				hashes[p] = calculateHash(f)
			}
		case map[string]any:
			if err := createFiles(vt, hashes, p); err != nil {
				return err
			}
		}
	}

	return nil
}

func createSymlinks(links map[string]string, path string) error {
	for k, v := range links {
		if err := os.Symlink(filepath.Join(path, k), filepath.Join(path, v)); err != nil {
			return err
		}
	}

	return nil
}

func lines(lines ...string) []byte {
	var b strings.Builder
	for i := range lines {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func calculateHash(file *os.File) []byte {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		panic(err)
	}
	h := md5.New()
	_, err = io.Copy(h, file)
	if err != nil {
		panic(err)
	}

	return h.Sum(nil)
}
