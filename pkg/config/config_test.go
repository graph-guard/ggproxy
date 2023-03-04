package config_test

import (
	"crypto/md5"
	"encoding/base32"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/graph-guard/ggproxy/pkg/config"
	"github.com/graph-guard/ggproxy/pkg/config/metadata"
	"github.com/graph-guard/gqt/v4"
	"github.com/stretchr/testify/require"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
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

func TestRead(t *testing.T) {
	basePath, expect := validFS(t)
	actual, err := config.Read(
		os.DirFS(basePath),
		basePath,
		ServerConfigFileName,
	)
	require.NoError(t, err)
	require.Equal(t, expect, actual)
}

func TestReadDefaultMaxReqBodySize(t *testing.T) {
	basePath, conf := validFS(t)
	createFiles(t, map[string]any{
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
	}, nil, basePath)
	conf.Proxy.MaxReqBodySizeBytes = config.DefaultMaxReqBodySize
}

func TestErrMissingServerConfig(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	err := os.Remove(p)
	require.NoError(t, err)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "server config",
	}, err)
	require.Nil(t, c)
}

func TestErrMalformedServerConfig(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	createFiles(t, map[string]any{
		ServerConfigFileName: lines("not a valid config"),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: p,
		Feature:  "syntax",
		Message: "yaml: unmarshal errors:\n  " +
			"line 1: cannot unmarshal !!str `not a v...` " +
			"into config.serverConfig",
	}, err)
	require.Nil(t, c)
}

func TestErrMissingProxyHostConfig(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	createFiles(t, map[string]any{
		ServerConfigFileName: lines(
			`proxy:`,
			`  host: `,
		),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "proxy.host",
	}, err)
	require.Nil(t, c)
}

func TestErrMissingAPIHostConfig(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	createFiles(t, map[string]any{
		ServerConfigFileName: lines(
			`proxy:`,
			`  host: localhost:8080`,
			`api:`,
			`  host: `,
		),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "api.host",
	}, err)
	require.Nil(t, c)
}

func TestErrMissingProxyTLSCert(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	createFiles(t, map[string]any{
		ServerConfigFileName: lines(
			`proxy:`,
			`  host: localhost:8080`,
			`  tls:`,
			`    key-file: proxy.key`,
			`api:`,
			`  host: localhost:9090`,
		),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "proxy.tls.cert-file",
	}, err)
	require.Nil(t, c)
}

func TestErrMissingProxyTLSKey(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	createFiles(t, map[string]any{
		ServerConfigFileName: lines(
			`proxy:`,
			`  host: localhost:8080`,
			`  tls:`,
			`    cert-file: proxy.cert`,
			`api:`,
			`  host: localhost:9090`,
		),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "proxy.tls.key-file",
	}, err)
	require.Nil(t, c)
}

func TestErrMissingAPITLSCert(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	createFiles(t, map[string]any{
		ServerConfigFileName: lines(
			`proxy:`,
			`  host: localhost:8080`,
			`api:`,
			`  host: localhost:9090`,
			`  tls:`,
			`    key-file: api.key`,
		),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "api.tls.cert-file",
	}, err)
	require.Nil(t, c)
}

func TestErrMissingAPITLSKey(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	createFiles(t, map[string]any{
		ServerConfigFileName: lines(
			`proxy:`,
			`  host: localhost:8080`,
			`api:`,
			`  host: localhost:9090`,
			`  tls:`,
			`    cert-file: api.cert`,
		),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "api.tls.key-file",
	}, err)
	require.Nil(t, c)
}

func TestErrIllegalProxyMaxReqBodySize(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, ServerConfigFileName)
	createFiles(t, map[string]any{
		ServerConfigFileName: lines(
			`proxy:`,
			`  host: localhost:8080`,
			`  max-request-body-size: 255`,
			`api:`,
			`  host: localhost:9090`,
		),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: p,
		Feature:  "proxy.max-request-body-size",
		Message: fmt.Sprintf(
			"maximum request body size should not be smaller than %d B",
			config.MinReqBodySize,
		),
	}, err)
	require.Nil(t, c)
}

func TestErrNoServices(t *testing.T) {
	basePath := minValidFS(t)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"irrelevant_file.txt": `this file only keeps the directory`,
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, config.ErrNoServices, err)
	require.Nil(t, c)
}

func TestErrNoServicesEnabled(t *testing.T) {
	basePath := minValidFS(t)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a.yml": lines(
				`path: /`,
				`forward-url: http://localhost:8080/`,
				`all-templates: ../all-templates/a`,
				`enabled-templates: ../enabled-templates/a`,
			),
		},
		"all-templates": map[string]any{
			"a": map[string]any{
				"a.gqt": `query { foo }`,
			},
		},
		"enabled-templates": map[string]any{
			"a": map[string]any{
				"a.gqt": `query { foo }`,
			},
		},
		"enabled-services": map[string]any{
			"placeholder.txt": "no services here",
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, config.ErrNoServicesEnabled, err)
	require.Nil(t, c)
}

func TestErrNoTemplates(t *testing.T) {
	basePath := minValidFS(t)
	serviceAConf := lines(
		`path: /`,
		`forward-url: http://localhost:8080/`,
		`all-templates: ../all-templates/a`,
		`enabled-templates: ../enabled-templates/a`,
	)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a.yml": serviceAConf,
		},
		"all-templates": map[string]any{
			"placeholder.txt": "no templates here",
		},
		"enabled-services": map[string]any{
			"a.yml": serviceAConf,
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, config.ErrNoTemplates, err)
	require.Nil(t, c)
}

func TestErrMalformedMetadata(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join("all-templates", "a", "a.gqt")
	createFiles(t, map[string]any{
		p: lines(
			"---",
			"malformed metadata",
			"---",
			`query { foo }`,
		),
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(basePath, p),
		Feature:  "metadata",
		Message: "decoding yaml: yaml: " +
			"unmarshal errors:\n  " +
			"line 1: cannot unmarshal !!str `malform...` " +
			"into metadata.Metadata",
	}, err)
	require.Nil(t, c)
}

func TestErrDuplicateTemplate(t *testing.T) {
	basePath, _ := validFS(t)
	t1 := filepath.Join("all-templates", "a", "d1.gqt")
	t2 := filepath.Join("all-templates", "a", "d2.gqt")
	createFiles(t, map[string]any{
		t1: `query { foo }`,
		t2: `query { foo }`,
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorDuplicate{
		Original:  filepath.Join(basePath, t1),
		Duplicate: filepath.Join(basePath, t2),
	}, err)
	require.Nil(t, c)
}

func TestErrDuplicateService(t *testing.T) {
	basePath := minValidFS(t)
	createFiles(t, map[string]any{
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
		"all-templates": map[string]any{
			"a": map[string]any{"a.gqt": `query {foo}`},
		},
		"enabled-templates": map[string]any{
			"a": map[string]any{"a.gqt": `query {foo}`},
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorDuplicate{
		Original:  filepath.Join(basePath, "all-services", "a.yml"),
		Duplicate: filepath.Join(basePath, "all-services", "b.yml"),
	}, err)
	require.Nil(t, c)
}

func TestErrConflictServicePath(t *testing.T) {
	basePath := minValidFS(t)
	serviceAConf := lines(
		`path: /`,
		`forward-url: http://localhost:8080/`,
		`all-templates: ../all-templates/a`,
		`enabled-templates: ../enabled-templates/a`,
	)
	serviceBConf := lines(
		`path: /`,
		`forward-url: http://localhost:8080/`,
		`all-templates: ../all-templates/b`,
		`enabled-templates: ../enabled-templates/b`,
	)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a.yml": serviceAConf,
			"b.yml": serviceBConf,
		},
		"enabled-services": map[string]any{
			"a.yml": serviceAConf,
			"b.yml": serviceBConf,
		},
		"all-templates": map[string]any{
			"a": map[string]any{"a.gqt": `query {foo}`},
			"b": map[string]any{"b.gqt": `query {foo}`},
		},
		"enabled-templates": map[string]any{
			"a": map[string]any{"a.gqt": `query {foo}`},
			"b": map[string]any{"b.gqt": `query {foo}`},
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Error(t, err)
	require.Equal(t, &config.ErrorConflict{
		Feature:  "path",
		Value:    "/",
		Subject1: filepath.Join(basePath, "all-services", "b.yml"),
		Subject2: filepath.Join(basePath, "all-services", "a.yml"),
	}, err)
	require.Nil(t, c)
}

func TestErrMissingPath(t *testing.T) {
	basePath := minValidFS(t)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a.yml": lines(
				`forward-url: http://localhost:8080/`,
			),
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: filepath.Join(
			basePath, "all-services", "a.yml",
		),
		Feature: "path",
	}, err)
	require.Nil(t, c)
}

func TestErrMissingForwardURL(t *testing.T) {
	basePath := minValidFS(t)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a.yml": lines(
				`path: /`,
			),
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: filepath.Join(
			basePath, "all-services", "a.yml",
		),
		Feature: "forward-url",
	}, err)
	require.Nil(t, c)
}

func TestErrInvalidPath(t *testing.T) {
	basePath := minValidFS(t)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a.yml": lines(
				`path: invalid_path`,
				`forward-url: http://localhost:8080/`,
			),
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(basePath, "all-services", "a.yml"),
		Feature:  "path",
		Message:  `path is not starting with /`,
	}, err)
	require.Nil(t, c)
}

func TestErrInvalidForwardURLInvalidScheme(t *testing.T) {
	basePath := minValidFS(t)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a.yml": lines(
				`path: /`,
				`forward-url: localhost:8080`,
			),
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(basePath, "all-services", "a.yml"),
		Feature:  "forward-url",
		Message:  `protocol is not supported or undefined`,
	}, err)
	require.Nil(t, c)
}

func TestErrInvalidSchema(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join("all-services", "schema_a.graphqls")
	createFiles(t, map[string]any{p: `type Query{ invalid }`}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(basePath, p),
		Feature:  "schema",
		Message: fmt.Sprintf(
			`invalid schema: %s:1: Expected :, found }`,
			filepath.Join(basePath, p),
		),
	}, err)
	require.Nil(t, c)
}

func TestErrMissingSchema(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join(basePath, "all-services", "schema_a.graphqls")
	err := os.Remove(p)
	require.NoError(t, err)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorMissing{
		FilePath: p,
		Feature:  "schema",
	}, err)
	require.Nil(t, c)
}

func TestErrInvalidTemplate(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join("all-templates", "a", "invalid_template.gqt")
	createFiles(t, map[string]any{p: `invalid { template }`}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(basePath, p),
		Feature:  "template",
		Message: `1:1: unexpected token, expected ` +
			`query, mutation, or subscription operation definition`,
	}, err)
	require.Nil(t, c)
}

func TestErrInvalidTemplateID(t *testing.T) {
	basePath, _ := validFS(t)
	p := filepath.Join("all-templates", "a", "invalid_template#.gqt")
	createFiles(t, map[string]any{p: `invalid { template }`}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(basePath, p),
		Feature:  "id",
		Message:  `contains illegal character at index 16`,
	}, err)
	require.Nil(t, c)
}

func TestErrInvalidServiceID(t *testing.T) {
	basePath, _ := validFS(t)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a#.yml": lines(
				`path: /`,
				`forward-url: http://localhost:8080/`,
				`all-templates: ../all-templates/a`,
				`enabled-templates: ../enabled-templates/a`,
			),
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(
			basePath,
			"all-services",
			"a#.yml",
		),
		Feature: "id",
		Message: `contains illegal character at index 1`,
	}, err)
	require.Nil(t, c)
}

func TestErrMalformedConfig(t *testing.T) {
	basePath, _ := validFS(t)
	createFiles(t, map[string]any{
		"all-services": map[string]any{
			"a.yml": lines(
				`malformed yaml`,
			),
		},
	}, nil, basePath)
	c, err := config.Read(os.DirFS(basePath), basePath, ServerConfigFileName)
	require.Equal(t, &config.ErrorIllegal{
		FilePath: filepath.Join(basePath, "all-services", "a.yml"),
		Feature:  "syntax",
		Message: "yaml: unmarshal errors:\n  " +
			"line 1: cannot unmarshal !!str `malform...` " +
			"into config.serviceConfig",
	}, err)
	require.Nil(t, c)
}

func TestErrorString(t *testing.T) {
	for _, td := range []struct {
		name   string
		input  error
		expect string
	}{
		{
			name: "missing_feature_in",
			input: config.ErrorMissing{
				FilePath: "path/to/file.txt",
				Feature:  "some_feature",
			},
			expect: "missing some_feature in path/to/file.txt",
		},
		{
			name: "missing_file",
			input: config.ErrorMissing{
				FilePath: "path/to/file.txt",
			},
			expect: "missing path/to/file.txt",
		},
		{
			name: "illegal_feature_in",
			input: config.ErrorIllegal{
				FilePath: "path/to/file.txt",
				Feature:  "some_feature",
				Message:  "some message",
			},
			expect: "illegal some_feature in path/to/file.txt: some message",
		},
		{
			name: "duplicate",
			input: config.ErrorDuplicate{
				Original:  "path/to/file_a.txt",
				Duplicate: "path/to/file_b.txt",
			},
			expect: "path/to/file_b.txt is a duplicate of path/to/file_a.txt",
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			require.Equal(t, td.expect, td.input.Error())
		})
	}
}

func minValidFS(t *testing.T) (base string) {
	base = t.TempDir()
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
			`of testing function Read.`,
		),
		"irrelevant-dir": map[string]any{
			"irrelevant_file.txt": lines(
				`this file is irrelevant and exists only for the purposes`,
				`of testing function Read.`,
			),
		},
	}

	hashes := make(map[string]string)

	createDirs(t, dirs, base)
	createFiles(t, files, hashes, base)

	return base
}

// validFS calls fn providing a valid setup filesystem.
func validFS(t *testing.T) (base string, conf *config.Config) {
	base = t.TempDir()
	type M = map[string]any

	dirs := M{
		"all-services":     nil,
		"enabled-services": nil,
		"all-templates": M{
			"a": nil,
			"b": nil,
		},
		"enabled-templates": M{
			"a": nil,
			"b": nil,
		},
		"irrelevant-dir": nil,
	}
	files := M{
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
		"all-services": M{
			"a.yml": lines(
				`path: "/path"`,
				`forward-url: "http://localhost:8080/path"`,
				`forward-reduced: true`,
				`schema: "schema_a.graphqls"`,
				`all-templates: "../all-templates/a"`,
				`enabled-templates: "../enabled-templates/a"`,
			),
			"schema_a.graphqls": lines(`type Query { foo:Int bar:String! }`),
			"b.yml": lines(
				`path: /`,
				`forward-url: "http://localhost:9090/"`,
				// Schemaless.
				`all-templates: "../all-templates/b"`,
				`enabled-templates: "../enabled-templates/b"`,
			),
			"ignored_file.txt": `this file should be ignored`,
		},
		"all-templates": M{
			"a": M{
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
			"b": M{
				"c.gqt":            `query { maz }`,
				"ignored_file.txt": `this file should be ignored`,
			},
		},
		"irrelevant-file.txt": lines(
			`this file is irrelevant and exists only for the purposes`,
			`of testing function Read.`,
		),
		"irrelevant-dir": M{
			"irrelevant_file.txt": lines(
				`this file is irrelevant and exists only for the purposes`,
				`of testing function Read.`,
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

	hashes := make(map[string]string)

	createDirs(t, dirs, base)
	createFiles(t, files, hashes, base)
	createSymlinks(t, links, base)

	serviceASchema, err := gqlparser.LoadSchema(&ast.Source{
		Name:  filepath.Join(base, "all-services", "schema_a.graphqls"),
		Input: files["all-services"].(M)["schema_a.graphqls"].(string),
	})
	require.NoError(t, err)

	serviceASchemaParser, err := gqt.NewParser([]gqt.Source{{
		Name:    filepath.Join(base, "all-services", "schema_a.graphqls"),
		Content: files["all-services"].(M)["schema_a.graphqls"].(string),
	}})
	require.NoError(t, err)

	services := make(map[string]*config.Service)
	templatesA := map[string]*config.Template{}
	templatesB := map[string]*config.Template{}

	{
		p := filepath.Join(base, "all-templates", "a", "a.gqt")
		templatesA[hashes[p]] = &config.Template{
			ID:     "a",
			Name:   "Template A",
			Tags:   []string{"tag_a"},
			Source: []byte(lines(`query { foo }`)),
			GQTTemplate: func() *gqt.Operation {
				_, body, err := metadata.Parse(
					[]byte(files["all-templates"].(M)["a"].(M)["a.gqt"].(string)),
				)
				require.NoError(t, err)
				template, _, errs := serviceASchemaParser.Parse(body)
				require.Nil(t, errs)
				return template
			}(),
			Enabled:  true,
			FilePath: p,
		}
	}

	{
		p := filepath.Join(base, "all-templates", "a", "b.gqt")
		templatesA[hashes[p]] = &config.Template{
			ID:     "b",
			Tags:   []string{"tag_b1", "tag_b2"},
			Source: []byte(lines(`query { bar }`)),
			GQTTemplate: func() *gqt.Operation {
				_, body, err := metadata.Parse(
					[]byte(files["all-templates"].(M)["a"].(M)["b.gqt"].(string)),
				)
				require.NoError(t, err)
				templates, _, errs := serviceASchemaParser.Parse(body)
				require.Nil(t, errs)
				return templates
			}(),
			Enabled:  true,
			FilePath: p,
		}
	}

	{
		p := filepath.Join(base, "all-services", "a.yml")
		services[hashes[p]] = &config.Service{
			ID:             "a",
			Path:           "/path",
			ForwardURL:     "http://localhost:8080/path",
			ForwardReduced: true,
			Schema:         serviceASchema,
			Templates:      templatesA,
			TemplatesEnabled: []*config.Template{
				templatesA[hashes[filepath.Join(base, "all-templates", "a", "a.gqt")]],
				templatesA[hashes[filepath.Join(base, "all-templates", "a", "b.gqt")]],
			},
			Enabled:  true,
			FilePath: p,
		}
	}

	{
		p := filepath.Join(base, "all-templates", "b", "c.gqt")
		templatesB[hashes[p]] = &config.Template{
			ID:     "c",
			Source: []byte(`query { maz }`),
			GQTTemplate: func() *gqt.Operation {
				_, body, err := metadata.Parse(
					[]byte(files["all-templates"].(M)["b"].(M)["c.gqt"].(string)),
				)
				require.NoError(t, err)
				template, _, errs := gqt.Parse(body)
				require.Nil(t, errs)
				return template
			}(),
			Enabled:  true,
			FilePath: p,
		}
	}

	{
		p := filepath.Join(base, "all-services", "b.yml")
		services[hashes[p]] = &config.Service{
			ID:             "b",
			Path:           "/",
			ForwardURL:     "http://localhost:9090/",
			ForwardReduced: false,
			Templates:      templatesB,
			TemplatesEnabled: []*config.Template{
				templatesB[hashes[filepath.Join(base, "all-templates", "b", "c.gqt")]],
			},
			Enabled:  true,
			FilePath: p,
		}
	}

	return base, &config.Config{
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
		Services: services,
		ServicesEnabled: []*config.Service{
			services[hashes[filepath.Join(base, "all-services", "a.yml")]],
			services[hashes[filepath.Join(base, "all-services", "b.yml")]],
		},
	}
}

func createDirs(t *testing.T, dirs map[string]any, basePath string) {
	for k, v := range dirs {
		p := filepath.Join(basePath, k)
		err := os.Mkdir(p, 0o775)
		require.NoError(t, err)
		if v != nil {
			switch vt := v.(type) {
			case map[string]any:
				createDirs(t, vt, p)
			default:
				panic(fmt.Errorf("unsupported dir content type: %v", v))
			}
		}
	}
}

func createFiles(
	t *testing.T,
	files map[string]any,
	hashes map[string]string,
	basePath string,
) {
	for k, v := range files {
		p := filepath.Join(basePath, k)
		switch vt := v.(type) {
		case string:
			f, err := os.Create(p)
			require.NoError(t, err)
			_, err = f.Write([]byte(vt))
			require.NoError(t, err)
			if hashes != nil {
				hashes[p] = calculateHash(t, f)
			}
		case map[string]any:
			createFiles(t, vt, hashes, p)
		default:
			panic(fmt.Errorf("unsupported file content type: %#v", v))
		}
	}
}

func createSymlinks(t *testing.T, links map[string]string, path string) {
	for k, v := range links {
		err := os.Symlink(filepath.Join(path, k), filepath.Join(path, v))
		require.NoError(t, err)
	}
}

func lines(lines ...string) string {
	var b strings.Builder
	for i := range lines {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	return b.String()
}

func calculateHash(t *testing.T, file *os.File) string {
	_, err := file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	h := md5.New()
	_, err = io.Copy(h, file)
	require.NoError(t, err)
	sum := h.Sum(nil)
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum)
	return s
}
