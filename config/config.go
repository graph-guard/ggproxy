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

const ServerConfigFile1 = "config.yaml"
const ServerConfigFile2 = "config.yml"
const ServicesEnabledDir = "services_enabled"
const ServicesDisabledDir = "services_disabled"
const TemplatesEnabledDir = "templates_enabled"
const TemplatesDisabledDir = "templates_disabled"
const ServiceConfigFile1 = "config.yaml"
const ServiceConfigFile2 = "config.yml"
const FileExtGQT = ".gqt"

type Config struct {
	Host             string
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
	var serverConf bool

	conf := &Config{}

	for _, o := range d {
		n := o.Name()
		if o.IsDir() {
			switch n {
			case ServicesEnabledDir:
				servicesEnabledDir = true
			case ServicesDisabledDir:
				servicesDisabledDir = true
			}
			continue
		} else if n == ServerConfigFile1 ||
			n == ServerConfigFile2 {
			if serverConf {
				return nil, &ErrorConflict{Items: []string{
					ServerConfigFile1,
					ServerConfigFile2,
				}}
			}
			serverConf = true

			p := filepath.Join(dirPath, n)
			f, err := filesystem.Open(p)
			if err != nil {
				return nil, fmt.Errorf("reading server config: %w", err)
			}

			var c serverConfig
			d := yaml.NewDecoder(f)
			d.KnownFields(true)
			if err := d.Decode(&c); err != nil {
				return nil, &ErrorIllegal{
					FilePath: p,
					Message:  err.Error(),
				}
			}
			if c.Host == "" {
				return nil, &ErrorMissing{
					FilePath: p,
					Feature:  "host",
				}
			}
			conf.Host = c.Host
		}
	}

	if !serverConf {
		return nil, &ErrorMissing{
			FilePath: filepath.Join(dirPath, ServerConfigFile1),
		}
	}

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
		return nil, &ErrorConflict{Items: []string{
			filepath.Join(ServicesEnabledDir, d.ID),
			filepath.Join(ServicesDisabledDir, d.ID),
		}}
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
			return nil, err
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
	if err := ValidateID(id); err != "" {
		return nil, &ErrorIllegal{
			FilePath: path,
			Feature:  "id",
			Message:  err,
		}
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
				return nil, &ErrorConflict{Items: []string{
					filepath.Join(path, ServiceConfigFile1),
					filepath.Join(path, ServiceConfigFile2),
				}}
			}
			c, err := readServiceConfigFile(
				filesystem,
				filepath.Join(path, n),
			)
			if err != nil {
				return nil, err
			}
			s.Name = c.Name
			s.ForwardURL = c.ForwardURL
			s.ForwardReduced = c.ForwardReduced
			configFile = true
		}
	}

	if !configFile {
		return nil, &ErrorMissing{
			FilePath: filepath.Join(path, ServiceConfigFile1),
		}
	}

	if d := duplicate(
		s.TemplatesEnabled,
		s.TemplatesDisabled,
		func(a, b *Template) bool { return a.ID == b.ID },
	); d != nil {
		return nil, &ErrorConflict{
			Items: []string{
				filepath.Join(TemplatesEnabledDir, d.ID),
				filepath.Join(TemplatesDisabledDir, d.ID),
			},
		}
	}

	return s, nil
}

type serverConfig struct {
	Host string `yaml:"host"`
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
		return nil, &ErrorIllegal{
			FilePath: path,
			Message:  err.Error(),
		}
	}
	if c.ForwardURL == "" {
		return nil, &ErrorMissing{
			FilePath: path,
			Feature:  "forward_url",
		}
	}
	if _, err := url.ParseRequestURI(c.ForwardURL); err != nil {
		return nil, &ErrorIllegal{
			FilePath: path,
			Feature:  "forward_url",
			Message:  err.Error(),
		}
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
		if err := ValidateID(id); err != "" {
			return nil, &ErrorIllegal{
				FilePath: p,
				Feature:  "id",
				Message:  err,
			}
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
			return nil, &ErrorIllegal{
				FilePath: p,
				Feature:  "metadata",
				Message:  err.Error(),
			}
		}

		doc, errParser := gqt.Parse(template)
		if errParser.IsErr() {
			return nil, &ErrorIllegal{
				FilePath: p,
				Feature:  "template",
				Message:  errParser.Error(),
			}
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

func ValidateID(n string) (err string) {
	if n == "" {
		return "empty"
	}
	for i := range n {
		if strings.IndexByte(IDValidCharDict, n[i]) < 0 {
			return fmt.Sprintf("contains illegal character at index %d", i)
		}
	}
	return ""
}

const IDValidCharDict = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"0123456789" +
	"_-"

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

type ErrorConflict struct {
	Items []string
}

func (e ErrorConflict) Error() string {
	var b strings.Builder
	b.WriteString("conflict between: ")
	for i := range e.Items {
		b.WriteString(e.Items[i])
		if i+1 < len(e.Items) {
			b.WriteString(", ")
		}
	}
	return b.String()
}

type ErrorMissing struct {
	FilePath string
	Feature  string
}

func (e ErrorMissing) Error() string {
	var b strings.Builder
	if e.Feature == "" {
		b.Grow(len("missing ") + len(e.FilePath))
		b.WriteString("missing ")
		b.WriteString(e.FilePath)
		return b.String()
	}
	b.Grow(len("missing ") + len(e.Feature) + len(" in ") + len(e.FilePath))
	b.WriteString("missing ")
	b.WriteString(e.Feature)
	b.WriteString(" in ")
	b.WriteString(e.FilePath)
	return b.String()
}

type ErrorIllegal struct {
	FilePath string
	Feature  string
	Message  string
}

func (e ErrorIllegal) Error() string {
	var b strings.Builder
	b.Grow(len("illegal ") +
		len(e.Feature) +
		len(" in ") +
		len(e.FilePath) +
		len(": ") +
		len(e.Message))
	b.WriteString("illegal ")
	b.WriteString(e.Feature)
	b.WriteString(" in ")
	b.WriteString(e.FilePath)
	b.WriteString(": ")
	b.WriteString(e.Message)
	return b.String()
}
