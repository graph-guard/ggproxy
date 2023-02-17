package config

import (
	"bytes"
	"crypto/md5"
	"encoding/base32"
	"errors"
	"fmt"
	"io/fs"
	neturl "net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/graph-guard/ggproxy/config/metadata"
	gqt "github.com/graph-guard/gqt/v4"
	gqlparser "github.com/vektah/gqlparser/v2"
	gqlast "github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/exp/slices"
	yaml "gopkg.in/yaml.v3"
)

var (
	ConfigFileExtension   = regexp.MustCompile(`\.(yml|yaml)$`)
	TemplateFileExtension = regexp.MustCompile(`\.gqt$`)
)

// MinReqBodySize defines the minimum accepted value for
// `max-request-body-size` in bytes.
const MinReqBodySize = 256

// DefaultMaxReqBodySize defines the default maximum
// request body size in bytes.
const DefaultMaxReqBodySize = 4 * 1024 * 1024

var msgMaxReqBodySizeTooSmall = fmt.Sprintf(
	"maximum request body size should not be smaller than %s",
	humanize.Bytes(MinReqBodySize),
)

type Config struct {
	Proxy           ProxyServerConfig
	API             *APIServerConfig
	Services        map[string]*Service
	ServicesEnabled []*Service
}

type ProxyServerConfig struct {
	Host                string
	TLS                 TLS
	MaxReqBodySizeBytes int
}

type APIServerConfig struct {
	Host string
	TLS  TLS
}

type TLS struct {
	CertFile string
	KeyFile  string
}

type Service struct {
	ID               string
	Path             string
	ForwardURL       string
	Schema           *gqlast.Schema
	Templates        map[string]*Template
	TemplatesEnabled []*Template
	ForwardReduced   bool
	Enabled          bool
	FilePath         string
}

type Template struct {
	ID          string
	Source      []byte
	GQTTemplate *gqt.Operation
	Name        string
	Tags        []string
	Enabled     bool
	FilePath    string
}

type serverConfig struct {
	Proxy struct {
		Host string `yaml:"host"`
		TLS  *struct {
			CertFile string `yaml:"cert-file"`
			KeyFile  string `yaml:"key-file"`
		} `yaml:"tls"`
		MaxRequestBodySizeBytes *int `yaml:"max-request-body-size"`
	} `yaml:"proxy"`
	API *struct {
		Host string `yaml:"host"`
		TLS  *struct {
			CertFile string `yaml:"cert-file"`
			KeyFile  string `yaml:"key-file"`
		} `yaml:"tls"`
	} `yaml:"api"`
	ServicesAll     string `yaml:"all-services"`
	ServicesEnabled string `yaml:"enabled-services"`
}

type serviceConfig struct {
	Name             string `yaml:"name"`
	Path             string `yaml:"path"`
	ForwardURL       string `yaml:"forward-url"`
	ForwardReduced   bool   `yaml:"forward-reduced"`
	TemplatesAll     string `yaml:"all-templates"`
	TemplatesEnabled string `yaml:"enabled-templates"`
	Schema           string `yaml:"schema"`
}

// Read parses configuration files and composes the server config.
func Read(fsys fs.FS, basePath, path string) (c *Config, err error) {
	// Set default config values
	c = &Config{
		Proxy:    ProxyServerConfig{},
		API:      &APIServerConfig{},
		Services: make(map[string]*Service),
	}
	if err = c.readServerConfig(fsys, basePath, path); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) readServerConfig(fsys fs.FS, basePath, path string) (err error) {
	dirPath := filepath.Dir(path)
	file, err := openFile(fsys, basePath, path, "server config")
	if err != nil {
		return err
	}

	sc := &serverConfig{}
	d := yaml.NewDecoder(file)
	d.KnownFields(true)
	if err := d.Decode(sc); err != nil {
		return &ErrorIllegal{
			FilePath: filepath.Join(basePath, path),
			Feature:  "syntax",
			Message:  err.Error(),
		}
	}

	if err := validateServerConfig(sc, basePath, path); err != nil {
		return err
	}

	c.Proxy.Host = sc.Proxy.Host
	if sc.Proxy.TLS != nil {
		c.Proxy.TLS.CertFile = sc.Proxy.TLS.CertFile
		c.Proxy.TLS.KeyFile = sc.Proxy.TLS.KeyFile
	}
	if sc.Proxy.MaxRequestBodySizeBytes == nil {
		c.Proxy.MaxReqBodySizeBytes = DefaultMaxReqBodySize
	} else {
		c.Proxy.MaxReqBodySizeBytes = *sc.Proxy.MaxRequestBodySizeBytes
	}
	if sc.API == nil {
		// Disable API server
		c.API = nil
	} else {
		c.API.Host = sc.API.Host
		if sc.API.TLS != nil {
			c.API.TLS.CertFile = sc.API.TLS.CertFile
			c.API.TLS.KeyFile = sc.API.TLS.KeyFile
		}
	}

	var servicesAllPath, servicesEnabledPath string
	servicesAllPath = sc.ServicesAll
	servicesEnabledPath = sc.ServicesEnabled
	if !strings.HasPrefix(servicesAllPath, "/") {
		servicesAllPath = filepath.Join(dirPath, servicesAllPath)
	}
	if !strings.HasPrefix(servicesEnabledPath, "/") {
		servicesEnabledPath = filepath.Join(dirPath, servicesEnabledPath)
	}

	// reading all services
	if err := c.readAllServices(fsys, basePath, servicesAllPath); err != nil {
		return err
	}

	if len(c.Services) < 1 {
		return ErrNoServices
	}

	// reading enabled services
	if err := c.readEnabledServices(fsys, servicesEnabledPath); err != nil {
		return err
	}

	if len(c.ServicesEnabled) < 1 {
		return ErrNoServicesEnabled
	}

	{ // Make sure there are no collisions on service paths
		sk := sortedKeys(c.Services)
		paths := make(map[string]*Service, len(c.Services))
		for _, sk := range sk {
			s := c.Services[sk]
			if s2, ok := paths[s.Path]; ok {
				return &ErrorConflict{
					Feature:  "path",
					Value:    s.Path,
					Subject1: s.FilePath,
					Subject2: s2.FilePath,
				}
			}
			paths[s.Path] = s
		}
	}

	return
}

func validateServerConfig(sc *serverConfig, basePath, path string) (err error) {
	if sc.Proxy.Host == "" {
		return &ErrorMissing{
			FilePath: filepath.Join(basePath, path),
			Feature:  "proxy.host",
		}
	}

	if sc.Proxy.TLS != nil {
		c := sc.Proxy.TLS
		// If either of proxy.cert-file and proxy.key-file are
		// present then both must be defined, otherwise TLS must be nil.
		switch {
		case c.CertFile != "" && c.KeyFile == "":
			return &ErrorMissing{
				FilePath: filepath.Join(basePath, path),
				Feature:  "proxy.tls.key-file",
			}
		case (c.KeyFile != "" && c.CertFile == "") ||
			(c.KeyFile == "" && c.CertFile == ""):
			return &ErrorMissing{
				FilePath: filepath.Join(basePath, path),
				Feature:  "proxy.tls.cert-file",
			}
		}
	}

	if sc.API != nil {
		c := sc.API
		if c.Host == "" {
			return &ErrorMissing{
				FilePath: filepath.Join(basePath, path),
				Feature:  "api.host",
			}
		}

		if c.TLS != nil {
			c := c.TLS
			// If either of api.cert-file and api.key-file are present
			// then both must be defined, otherwise TLS must be nil.
			switch {
			case c.CertFile != "" && c.KeyFile == "":
				return &ErrorMissing{
					FilePath: filepath.Join(basePath, path),
					Feature:  "api.tls.key-file",
				}
			case (c.KeyFile != "" && c.CertFile == "") ||
				(c.KeyFile == "" && c.CertFile == ""):
				return &ErrorMissing{
					FilePath: filepath.Join(basePath, path),
					Feature:  "api.tls.cert-file",
				}
			}
		}
	}

	if sc.Proxy.MaxRequestBodySizeBytes != nil {
		if *sc.Proxy.MaxRequestBodySizeBytes < MinReqBodySize {
			return &ErrorIllegal{
				FilePath: filepath.Join(basePath, path),
				Feature:  "proxy.max-request-body-size",
				Message:  msgMaxReqBodySizeTooSmall,
			}
		}
	}

	if sc.ServicesAll == "" {
		return &ErrorMissing{
			FilePath: filepath.Join(basePath, path),
			Feature:  "all-services",
		}
	}
	if sc.ServicesEnabled == "" {
		return &ErrorMissing{
			FilePath: filepath.Join(basePath, path),
			Feature:  "enabled-services",
		}
	}

	return
}

func (c *Config) readAllServices(fsys fs.FS, basePath, path string) (err error) {
	d, err := fs.ReadDir(fsys, path)
	if err != nil {
		return fmt.Errorf("reading services directory: %w", err)
	}

	for _, sf := range d {
		if sf.IsDir() {
			continue
		}
		if !ConfigFileExtension.MatchString(sf.Name()) {
			// Ignore non-yaml files
			continue
		}

		filePath := filepath.Join(path, sf.Name())
		h, err := calculateHash(fsys, filePath)
		if err != nil {
			return err
		}

		s, err := readServiceConfig(fsys, basePath, filePath)
		if err != nil {
			return err
		}
		original, ok := c.Services[string(h)]
		if ok {
			return &ErrorDuplicate{
				Original:  original.FilePath,
				Duplicate: s.FilePath,
			}
		}
		c.Services[string(h)] = s
	}

	return
}

func (c *Config) readEnabledServices(fsys fs.FS, path string) (err error) {
	d, err := fs.ReadDir(fsys, path)
	if err != nil {
		return fmt.Errorf("reading enabled services directory: %w", err)
	}

	var alien []string
	for _, sf := range d {
		if sf.IsDir() {
			continue
		}
		if !ConfigFileExtension.MatchString(sf.Name()) {
			// Ignore non-yaml files
			continue
		}

		filePath := filepath.Join(path, sf.Name())
		h, err := calculateHash(fsys, filePath)
		if err != nil {
			return err
		}
		s, ok := c.Services[string(h)]
		if ok {
			s.Enabled = true
			c.ServicesEnabled = append(c.ServicesEnabled, s)
		} else {
			alien = append(alien, filePath)
		}
	}

	if len(alien) > 0 {
		return &ErrorAlien{Items: alien}
	}

	return
}

func readServiceConfig(fsys fs.FS, basePath, filePath string) (s *Service, err error) {
	dirPath := filepath.Dir(filePath)

	sc := &serviceConfig{}
	{
		c, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return nil, err
		}
		d := yaml.NewDecoder(bytes.NewReader(c))
		d.KnownFields(true)
		if err := d.Decode(sc); err != nil {
			return nil, &ErrorIllegal{
				FilePath: filepath.Join(basePath, filePath),
				Feature:  "syntax",
				Message:  err.Error(),
			}
		}
	}

	if err := validateServiceConfig(sc, basePath, filePath); err != nil {
		return nil, err
	}

	// TODO: Add support for multiple graphqls files
	var schemaPath string
	if sc.Schema != "" {
		schemaPath = filepath.Join(dirPath, sc.Schema)
	}
	schema, gqtParser, err := s.readSchema(fsys, basePath, schemaPath)
	if err != nil {
		return nil, err
	}

	var templatesAllPath, templatesEnabledPath string
	templatesAllPath = sc.TemplatesAll
	templatesEnabledPath = sc.TemplatesEnabled
	if !strings.HasPrefix(templatesAllPath, "/") {
		templatesAllPath = filepath.Join(dirPath, templatesAllPath)
	}
	if !strings.HasPrefix(templatesEnabledPath, "/") {
		templatesEnabledPath = filepath.Join(dirPath, templatesEnabledPath)
	}

	id := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	if err := ValidateID(id); err != "" {
		return nil, &ErrorIllegal{
			FilePath: filepath.Join(basePath, filePath),
			Feature:  "id",
			Message:  err,
		}
	}
	id = strings.ToLower(id)

	s = &Service{
		ID:             id,
		Schema:         schema,
		Templates:      map[string]*Template{},
		FilePath:       filepath.Join(basePath, filePath),
		Path:           sc.Path,
		ForwardURL:     sc.ForwardURL,
		ForwardReduced: sc.ForwardReduced,
	}

	// reading all templates
	if err := s.readAllTemplates(
		fsys, basePath, templatesAllPath, gqtParser,
	); err != nil {
		return nil, err
	}

	if len(s.Templates) < 1 {
		return nil, ErrNoTemplates
	}

	// reading enabled templates
	if err := s.readEnabledTemplates(fsys, templatesEnabledPath); err != nil {
		return nil, err
	}

	if len(s.TemplatesEnabled) < 1 {
		return nil, ErrNoTemplatesEnabled
	}

	return
}

func validateServiceConfig(sc *serviceConfig, basePath, path string) (err error) {
	if sc.Path == "" {
		return &ErrorMissing{
			FilePath: filepath.Join(basePath, path),
			Feature:  "path",
		}
	}
	if err := validatePath(sc.Path); err != nil {
		return &ErrorIllegal{
			FilePath: filepath.Join(basePath, path),
			Feature:  "path",
			Message:  err.Error(),
		}
	}
	if sc.ForwardURL == "" {
		return &ErrorMissing{
			FilePath: filepath.Join(basePath, path),
			Feature:  "forward-url",
		}
	}
	if err := validateURL(sc.ForwardURL); err != nil {
		return &ErrorIllegal{
			FilePath: filepath.Join(basePath, path),
			Feature:  "forward-url",
			Message:  err.Error(),
		}
	}
	if sc.TemplatesAll == "" {
		return &ErrorMissing{
			FilePath: filepath.Join(basePath, path),
			Feature:  "all-templates",
		}
	}
	if sc.TemplatesEnabled == "" {
		return &ErrorMissing{
			FilePath: filepath.Join(basePath, path),
			Feature:  "enabled-templates",
		}
	}

	return
}

func (s *Service) readSchema(fsys fs.FS, basePath, path string) (*gqlast.Schema, *gqt.Parser, error) {
	if path == "" {
		p, err := gqt.NewParser(nil)
		return nil, p, err
	}

	f, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, nil, &ErrorMissing{
			FilePath: filepath.Join(basePath, path),
			Feature:  "schema",
		}
	}

	schema, err := gqlparser.LoadSchema(&gqlast.Source{
		Name:  filepath.Join(basePath, path),
		Input: string(f),
	})
	if err != nil {
		return nil, nil, &ErrorIllegal{
			FilePath: filepath.Join(basePath, path),
			Feature:  "schema",
			Message:  fmt.Sprintf("invalid schema: %v", err.Error()),
		}
	}

	gqtParser, err := gqt.NewParser([]gqt.Source{
		{Name: filepath.Join(basePath, path), Content: string(f)},
	})
	return schema, gqtParser, err
}

func (s *Service) readAllTemplates(
	fsys fs.FS,
	basePath, path string,
	p *gqt.Parser,
) (err error) {
	dir, err := fs.ReadDir(fsys, path)
	if err != nil {
		return fmt.Errorf("reading templates directory: %w", err)
	}

	for _, tf := range dir {
		if tf.IsDir() {
			continue
		}
		if !TemplateFileExtension.MatchString(tf.Name()) {
			// Ignore non-GQT files
			continue
		}

		path := filepath.Join(path, tf.Name())
		h, err := calculateHash(fsys, path)
		if err != nil {
			return err
		}

		t, err := readTemplate(fsys, basePath, path, p)
		if err != nil {
			return err
		}

		original, ok := s.Templates[string(h)]
		if ok {
			return &ErrorDuplicate{
				Original:  original.FilePath,
				Duplicate: t.FilePath,
			}
		}
		s.Templates[string(h)] = t
	}

	return
}

func (s *Service) readEnabledTemplates(fsys fs.FS, path string) (err error) {
	dir, err := fs.ReadDir(fsys, path)
	if err != nil {
		return fmt.Errorf("reading enabled templates directory: %w", err)
	}

	var aliens []string
	for _, tf := range dir {
		if tf.IsDir() {
			continue
		}
		if !TemplateFileExtension.MatchString(tf.Name()) {
			// Ignore non-GQT files
			continue
		}

		p := filepath.Join(path, tf.Name())
		h, err := calculateHash(fsys, p)
		if err != nil {
			return err
		}
		t, ok := s.Templates[string(h)]
		if ok {
			t.Enabled = true
			s.TemplatesEnabled = append(s.TemplatesEnabled, t)
		} else {
			aliens = append(aliens, tf.Name())
		}
	}

	if len(aliens) > 0 {
		return &ErrorAlien{Items: aliens}
	}

	return
}

func readTemplate(
	fsys fs.FS,
	basePath, filePath string,
	p *gqt.Parser,
) (t *Template, err error) {
	id := strings.ToLower(
		strings.TrimSuffix(
			filepath.Base(filePath), filepath.Ext(filePath),
		),
	)
	if err := ValidateID(id); err != "" {
		return nil, &ErrorIllegal{
			FilePath: filepath.Join(basePath, filePath),
			Feature:  "id",
			Message:  err,
		}
	}

	b, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return nil, fmt.Errorf("reading template %q: %w", filePath, err)
	}

	meta, template, err := metadata.Parse(b)
	if err != nil {
		return nil, &ErrorIllegal{
			FilePath: filepath.Join(basePath, filePath),
			Feature:  "metadata",
			Message:  err.Error(),
		}
	}

	doc, _, errs := p.Parse(template)
	if errs != nil {
		var msg strings.Builder
		for i := range errs {
			msg.WriteString(errs[i].Error())
			if i+1 < len(errs) {
				msg.WriteString("; ")
			}
		}
		return nil, &ErrorIllegal{
			FilePath: filepath.Join(basePath, filePath),
			Feature:  "template",
			Message:  msg.String(),
		}
	}

	t = &Template{
		ID:          id,
		FilePath:    filepath.Join(basePath, filePath),
		Source:      template,
		GQTTemplate: doc,
		Name:        meta.Name,
		Tags:        meta.Tags,
	}

	return
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

type ErrorDuplicate struct{ Original, Duplicate string }

func (e ErrorDuplicate) Error() string {
	return fmt.Sprintf("%s is a duplicate of %s", e.Duplicate, e.Original)
}

var (
	ErrNoServices         = errors.New("no services defined")
	ErrNoServicesEnabled  = errors.New("no services enabled")
	ErrNoTemplates        = errors.New("no templates defined")
	ErrNoTemplatesEnabled = errors.New("no templates enabled")
)

type ErrorAlien struct{ Items []string }

func (e ErrorAlien) Error() string {
	var b strings.Builder
	b.WriteString("templates are not defined in the service templates pool (all-templates): ")
	for i := range e.Items {
		b.WriteString(e.Items[i])
		if i+1 < len(e.Items) {
			b.WriteString(", ")
		}
	}
	return b.String()
}

type ErrorConflict struct{ Feature, Value, Subject1, Subject2 string }

func (e ErrorConflict) Error() string {
	return "conflict on " + e.Feature + " (" + e.Value +
		") between " + e.Subject1 + " and " + e.Subject2
}

type ErrorMissing struct{ FilePath, Feature string }

func (e ErrorMissing) Error() string {
	if e.Feature == "" {
		return "missing " + e.FilePath
	}
	return "missing " + e.Feature + " in " + e.FilePath
}

type ErrorIllegal struct{ FilePath, Feature, Message string }

func (e ErrorIllegal) Error() string {
	return "illegal " + e.Feature + " in " + e.FilePath + ": " + e.Message
}

var (
	ErrPathNotAbsolute    = errors.New("path is not starting with /")
	ErrURLProtocolProblem = errors.New("protocol is not supported or undefined")
	ErrURLNoHost          = errors.New("host is not defined")
)

var ValidProtocolSchemes = []string{"http", "https"}

func validateURL(url string) error {
	u, err := neturl.Parse(url)
	if err != nil {
		return err
	}

	if contains(ValidProtocolSchemes, u.Scheme, func(a, b string) bool { return a == b }) == -1 {
		return ErrURLProtocolProblem
	}
	if u.Host == "" {
		return ErrURLNoHost
	}

	return nil
}

func validatePath(path string) error {
	if !filepath.IsAbs(path) {
		return ErrPathNotAbsolute
	}

	return nil
}

func contains[T any](arr []T, x T, equal func(a, b T) bool) int {
	for i, el := range arr {
		if equal(el, x) {
			return i
		}
	}

	return -1
}

func openFile(fsys fs.FS, basePath, path, feature string) (fs.File, error) {
	f, err := fsys.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, &ErrorMissing{
				FilePath: filepath.Join(basePath, path),
				Feature:  feature,
			}
		}
		return nil, fmt.Errorf("opening file: %w", err)
	}
	return f, nil
}

// calculateHash returns a base32 encoded MD5 hash of file.
func calculateHash(fsys fs.FS, path string) (string, error) {
	h := md5.New()
	c, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	if _, err := h.Write(c); err != nil {
		return "", fmt.Errorf("writing: %w", err)
	}
	sum := h.Sum(nil)
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum)
	return s, nil
}

func sortedKeys[T any](m map[string]T) []string {
	l := make([]string, 0, len(m))
	for k := range m {
		l = append(l, k)
	}
	slices.Sort(l)
	return l
}
