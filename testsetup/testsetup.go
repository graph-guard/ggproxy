package testsetup

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/graph-guard/ggproxy/config"
	"gopkg.in/yaml.v3"
)

/* SPECIAL NOTE:                                     *\
\* Symlinks are not allowed in embedded filesystems! */

//go:embed starwars
var fsStarwars embed.FS

//go:embed test1
var fsTest1 embed.FS

//go:embed inputs_schema
var fsInputsSchema embed.FS

func Starwars() Setup     { return read(fsStarwars, "starwars") }
func Test1() Setup        { return read(fsTest1, "test1") }
func InputsSchema() Setup { return read(fsInputsSchema, "inputs_schema") }

func read(fsys fs.FS, root string) Setup {
	c, err := config.Read(fsys, root, filepath.Join(root, "config.yml"))
	panicOnErr(err)
	t, err := readTests(fsys, root)
	panicOnErr(err)
	return Setup{
		Name:   root,
		Config: c,
		Tests:  t,
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

type TestModel struct {
	Client struct {
		Input struct {
			Method   string         `yaml:"method"`
			Endpoint string         `yaml:"endpoint"`
			Body     string         `yaml:"body"`
			BodyJSON map[string]any `yaml:"body(JSON)"`
		} `yaml:"input"`
		ExpectResponse struct {
			Status   int               `yaml:"status"`
			Headers  map[string]string `yaml:"headers"` // Key -> Regexp
			Body     string            `yaml:"body"`
			BodyJSON map[string]any    `yaml:"body(JSON)"`
		} `yaml:"expect-response"`
	} `yaml:"client"`
	Destination *struct {
		ExpectForwarded struct {
			Headers  map[string]string `yaml:"headers"` // Key -> Regexp
			Body     string            `yaml:"body"`
			BodyJSON map[string]any    `yaml:"body(JSON)"`
		} `yaml:"expect-forwarded"`
		Response struct {
			Status   int               `yaml:"status"`
			Headers  map[string]string `yaml:"headers"`
			Body     string            `yaml:"body"`
			BodyJSON map[string]any    `yaml:"body(JSON)"`
		}
	} `yaml:"destination"`
	Logs []map[string]any `yaml:"logs"`
}

type Setup struct {
	Name   string
	Config *config.Config
	Tests  []Test
}

type Test struct {
	Name string
	TestModel
}

func readTests(fsys fs.FS, root string) ([]Test, error) {
	var tests []Test
	d, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("reading test dir: %w", err)
	}
	for _, testDir := range d {
		n := testDir.Name()
		if !strings.HasPrefix(n, "test_") || !strings.HasSuffix(n, ".yaml") {
			continue
		}

		p := filepath.Join(root, n)
		f, err := fsys.Open(p)
		if err != nil {
			return nil, fmt.Errorf("reading test file %q: %w", p, err)
		}
		defer f.Close()

		var m TestModel
		d := yaml.NewDecoder(f)
		d.KnownFields(true)
		if err := d.Decode(&m); err != nil {
			return nil, fmt.Errorf("decoding YAML %q: %w", p, err)
		}

		if err := isXOR(
			m.Client.Input.Body,
			m.Client.Input.BodyJSON,
			"client.input.body",
			"client.input.body(JSON)",
		); err != nil {
			return nil, err
		}
		if err := isXOR(
			m.Client.ExpectResponse.Body,
			m.Client.ExpectResponse.BodyJSON,
			"client.expect-response.body",
			"client.expect-response.body(JSON)",
		); err != nil {
			return nil, err
		}
		if m.Destination != nil {
			if err := isXOR(
				m.Destination.ExpectForwarded.Body,
				m.Destination.ExpectForwarded.BodyJSON,
				"destination.expect-forwarded.body",
				"destination.expect-forwarded.body(JSON)",
			); err != nil {
				return nil, err
			}
			if err := isXOR(
				m.Destination.Response.Body,
				m.Destination.Response.BodyJSON,
				"destination.expect-forwarded.body",
				"destination.expect-forwarded.body(JSON)",
			); err != nil {
				return nil, err
			}
		}

		tests = append(tests, Test{
			Name:      n,
			TestModel: m,
		})
	}
	return tests, nil
}

func isXOR(a string, b map[string]any, aTitle, bTitle string) error {
	if (a != "" && b == nil) || (a == "" && b != nil) {
		return nil
	}
	return fmt.Errorf(`%q (%q) and %q (%v) are mutually exclusive, `+
		`make sure you're using either of them, not both at the same time!`,
		aTitle, a, bTitle, b,
	)
}
