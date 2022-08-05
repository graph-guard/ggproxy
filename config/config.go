package config

import (
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/graph-guard/gguard-proxy/config/metadata"
	"github.com/graph-guard/gqt"
	yaml "gopkg.in/yaml.v3"
)

const ServicesEnabledDir = "services_enabled"
const ServicesDisabledDir = "services_disabled"
const TemplatesEnabledDir = "templates_enabled"
const TemplatesDisabledDir = "templates_disabled"
const ServiceConfigFile1 = "config.yml"
const ServiceConfigFile2 = "config.yaml"
const FileExtGQT = ".gqt"

type Config struct {
	ServicesEnabled  []*Service
	ServicesDisabled []*Service
}

type Service struct {
	ID                string
	Name              string
	TemplatesEnabled  []*Template
	TemplatesDisabled []*Template
	ForwardURL        string
	ForwardReduced    bool
}

type Template struct {
	ID       string
	Source   []byte
	Document gqt.Doc
	Name     string
	Tags     []string
}

func ReadConfig(filesystem fs.FS, dirPath string) (*Config, error) {
	d, err := fs.ReadDir(filesystem, dirPath)
	if err != nil {
		return nil, fmt.Errorf("reading config directory: %w", err)
	}

	var servicesEnabledDir bool
	var servicesDisabledDir bool

	for _, o := range d {
		if o.IsDir() {
			switch o.Name() {
			case ServicesEnabledDir:
				servicesEnabledDir = true
			case ServicesDisabledDir:
				servicesDisabledDir = true
			}
			continue
		}
	}

	conf := &Config{}

	if servicesEnabledDir {
		s, err := readServicesDir(filesystem, ServicesEnabledDir)
		if err != nil {
			return nil, err
		}
		conf.ServicesEnabled = s
	}

	if servicesDisabledDir {
		s, err := readServicesDir(filesystem, ServicesDisabledDir)
		if err != nil {
			return nil, err
		}
		conf.ServicesDisabled = s
	}

	if d := duplicate(
		conf.ServicesEnabled,
		conf.ServicesDisabled,
		func(a, b *Service) bool { return a.ID == b.ID },
	); d != nil {
		return nil, fmt.Errorf(
			"service %q is both enabled and disabled", d.ID,
		)
	}

	return conf, nil
}

func readServicesDir(filesystem fs.FS, path string) ([]*Service, error) {
	d, err := fs.ReadDir(filesystem, path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}
	var services []*Service
	for _, o := range d {
		if !o.IsDir() {
			// Ignore files
			continue
		}
		s, err := readServiceDir(
			filesystem, filepath.Join(path, o.Name()),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"service %q: %w", o.Name(), err,
			)
		}
		services = append(services, s)
	}
	return services, nil
}

func readServiceDir(filesystem fs.FS, path string) (*Service, error) {
	dir, err := fs.ReadDir(filesystem, path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	id := filepath.Base(path)
	if err := ValidateID(id); err != nil {
		return nil, fmt.Errorf("validating %q: %w", path, err)
	}
	id = strings.ToLower(id)

	var configFile bool
	s := &Service{
		ID: id,
	}

	for _, o := range dir {
		n := o.Name()
		if o.IsDir() {
			switch n {
			case TemplatesEnabledDir:
				templates, err := readTemplatesDir(
					filesystem, filepath.Join(path, n),
				)
				if err != nil {
					return nil, err
				}
				s.TemplatesEnabled = append(s.TemplatesEnabled, templates...)
			case TemplatesDisabledDir:
				templates, err := readTemplatesDir(
					filesystem, filepath.Join(path, n),
				)
				if err != nil {
					return nil, err
				}
				s.TemplatesDisabled = append(s.TemplatesDisabled, templates...)
			}
			continue
		}
		if n == ServiceConfigFile1 ||
			n == ServiceConfigFile2 {
			if configFile {
				return nil, fmt.Errorf(
					"conflicting files: %q - %q",
					filepath.Join(path, ServiceConfigFile1),
					filepath.Join(path, ServiceConfigFile2),
				)
			}
			c, err := readServiceConfigFile(
				filesystem,
				filepath.Join(path, n),
			)
			if err != nil {
				return nil, fmt.Errorf("reading %q: %w", n, err)
			}
			s.Name = c.Name
			s.ForwardURL = c.ForwardURL
			s.ForwardReduced = c.ForwardReduced
			configFile = true
		}
	}

	if !configFile {
		return nil, ErrMissingConfigFile
	}

	if d := duplicate(
		s.TemplatesEnabled,
		s.TemplatesDisabled,
		func(a, b *Template) bool { return a.ID == b.ID },
	); d != nil {
		return nil, fmt.Errorf(
			"template %q is both enabled and disabled", d.ID,
		)
	}

	return s, nil
}

type serviceConfig struct {
	Name           string `yaml:"name"`
	ForwardURL     string `yaml:"forward_url"`
	ForwardReduced bool   `yaml:"forward_reduced"`
}

func readServiceConfigFile(
	filesystem fs.FS,
	path string,
) (*serviceConfig, error) {
	f, err := filesystem.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	var c serviceConfig
	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err := d.Decode(&c); err != nil {
		return nil, fmt.Errorf("decoding yaml: %w", err)
	}
	if c.ForwardURL == "" {
		return nil, ErrMissingForwardURL
	}
	if _, err := url.ParseRequestURI(c.ForwardURL); err != nil {
		return nil, err
	}
	return &c, nil
}

func readTemplatesDir(
	filesystem fs.FS,
	path string,
) (t []*Template, err error) {
	dir, err := fs.ReadDir(filesystem, path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	for _, o := range dir {
		if o.IsDir() {
			continue
		}
		n := o.Name()
		if !strings.HasSuffix(n, FileExtGQT) {
			// Ignore non-GQT files
			continue
		}
		p := filepath.Join(path, n)
		id := n[:len(n)-len(filepath.Ext(n))]
		if err := ValidateID(id); err != nil {
			return nil, fmt.Errorf("validating %q: %w", p, err)
		}

		id = strings.ToLower(id)

		src, err := filesystem.Open(p)
		if err != nil {
			return nil, fmt.Errorf("opening %q: %w", p, err)
		}
		b, err := io.ReadAll(src)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", p, err)
		}

		meta, template, err := metadata.Parse(b)
		if err != nil {
			return nil, fmt.Errorf("parsing %q metadata: %w", p, err)
		}

		doc, errp := gqt.Parse(template)
		if errp.IsErr() {
			return nil, fmt.Errorf("parsing %q: %w", p, errp)
		}
		t = append(t, &Template{
			ID:       id,
			Source:   template,
			Document: doc,
			Name:     meta.Name,
			Tags:     meta.Tags,
		})
	}

	return t, nil
}

func ValidateID(n string) error {
	if n == "" {
		return fmt.Errorf("empty")
	}
	for i := range n {
		if strings.IndexByte(IDValidCharDict, n[i]) < 0 {
			return ErrIllegalID
		}
	}
	return nil
}

const IDValidCharDict = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"0123456789" +
	"_-"

var ErrMissingConfigFile = fmt.Errorf("missing %s", ServiceConfigFile1)
var ErrMissingForwardURL = fmt.Errorf("missing forward_url")
var ErrIllegalID = fmt.Errorf("illegal identifier")

func duplicate[T any](a, b []T, isEqual func(a, b T) bool) (d T) {
	for i := range a {
		for i2 := range b {
			if isEqual(a[i], b[i2]) {
				return a[i]
			}
		}
	}
	return
}
