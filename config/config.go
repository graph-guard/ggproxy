package config

import (
	"crypto/md5"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	neturl "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/graph-guard/ggproxy/config/metadata"
	gqt "github.com/graph-guard/gqt/v4"
	gqlparser "github.com/vektah/gqlparser/v2"
	gqlast "github.com/vektah/gqlparser/v2/ast"
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

func New(path string) (c *Config, err error) {
	// Set default config values
	c = &Config{
		Proxy:    ProxyServerConfig{},
		API:      &APIServerConfig{},
		Services: make(map[string]*Service),
	}
	if err = c.readServerConfig(path); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) readServerConfig(path string) (err error) {
	dirPath := filepath.Dir(path)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening server config file: %w", err)
	}

	sc := &serverConfig{}
	d := yaml.NewDecoder(file)
	d.KnownFields(true)
	if err := d.Decode(sc); err != nil {
		return &ErrorIllegal{
			FilePath: path,
			Feature:  "syntax",
			Message:  err.Error(),
		}
	}

	if err := validateServerConfig(sc, path); err != nil {
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
	if err := c.readAllServices(servicesAllPath); err != nil {
		return err
	}

	// reading enabled services
	if err := c.readEnabledServices(servicesEnabledPath); err != nil {
		return err
	}

	return
}

func validateServerConfig(sc *serverConfig, path string) (err error) {
	if sc.Proxy.Host == "" {
		return &ErrorMissing{
			FilePath: path,
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
				FilePath: path,
				Feature:  "proxy.tls.key-file",
			}
		case (c.KeyFile != "" && c.CertFile == "") ||
			(c.KeyFile == "" && c.CertFile == ""):
			return &ErrorMissing{
				FilePath: path,
				Feature:  "proxy.tls.cert-file",
			}
		}
	}

	if sc.API != nil {
		c := sc.API
		if c.Host == "" {
			return &ErrorMissing{
				FilePath: path,
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
					FilePath: path,
					Feature:  "api.tls.key-file",
				}
			case (c.KeyFile != "" && c.CertFile == "") ||
				(c.KeyFile == "" && c.CertFile == ""):
				return &ErrorMissing{
					FilePath: path,
					Feature:  "api.tls.cert-file",
				}
			}
		}
	}

	if sc.Proxy.MaxRequestBodySizeBytes != nil {
		if *sc.Proxy.MaxRequestBodySizeBytes < MinReqBodySize {
			return &ErrorIllegal{
				FilePath: path,
				Feature:  "proxy.max-request-body-size",
				Message:  msgMaxReqBodySizeTooSmall,
			}
		}
	}

	if sc.ServicesAll == "" {
		return &ErrorMissing{
			FilePath: path,
			Feature:  "all-services",
		}
	}
	if sc.ServicesEnabled == "" {
		return &ErrorMissing{
			FilePath: path,
			Feature:  "enabled-services",
		}
	}

	return
}

func (c *Config) readAllServices(path string) (err error) {
	d, err := os.ReadDir(path)
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

		file, err := openFile(filepath.Join(path, sf.Name()))
		if err != nil {
			return err
		}
		h, err := calculateHash(file)
		if err != nil {
			return err
		}

		s, err := readServiceConfig(file)
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

func (c *Config) readEnabledServices(path string) (err error) {
	d, err := os.ReadDir(path)
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
		file, err := openFile(filePath)
		if err != nil {
			return err
		}
		h, err := calculateHash(file)
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

func readServiceConfig(file *os.File) (s *Service, err error) {
	filePath := file.Name()
	dirPath := filepath.Dir(file.Name())

	sc := &serviceConfig{}
	{
		d := yaml.NewDecoder(file)
		d.KnownFields(true)
		if err := d.Decode(sc); err != nil {
			return nil, &ErrorIllegal{
				FilePath: filePath,
				Feature:  "syntax",
				Message:  err.Error(),
			}
		}
	}

	if err := validateServiceConfig(sc, filePath); err != nil {
		return nil, err
	}

	// TODO: Add support for multiple graphqls files
	var schemaPath string
	if sc.Schema != "" {
		schemaPath = filepath.Join(dirPath, sc.Schema)
	}
	schema, gqtParser, err := s.readSchema(schemaPath)
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
			FilePath: filePath,
			Feature:  "id",
			Message:  err,
		}
	}
	id = strings.ToLower(id)

	s = &Service{
		ID:             id,
		Schema:         schema,
		Templates:      map[string]*Template{},
		FilePath:       filePath,
		Path:           sc.Path,
		ForwardURL:     sc.ForwardURL,
		ForwardReduced: sc.ForwardReduced,
	}

	// reading all templates
	err = s.readAllTemplates(templatesAllPath, gqtParser)
	if err != nil {
		return nil, err
	}

	// reading enabled templates
	err = s.readEnabledTemplates(templatesEnabledPath)
	if err != nil {
		return nil, err
	}

	return
}

func validateServiceConfig(sc *serviceConfig, path string) (err error) {
	if sc.Path == "" {
		return &ErrorMissing{
			FilePath: path,
			Feature:  "path",
		}
	}
	if err := validatePath(sc.Path); err != nil {
		return &ErrorIllegal{
			FilePath: path,
			Feature:  "path",
			Message:  err.Error(),
		}
	}
	if sc.ForwardURL == "" {
		return &ErrorMissing{
			FilePath: path,
			Feature:  "forward-url",
		}
	}
	if err := validateURL(sc.ForwardURL); err != nil {
		return &ErrorIllegal{
			FilePath: path,
			Feature:  "forward-url",
			Message:  err.Error(),
		}
	}
	if sc.TemplatesAll == "" {
		return &ErrorMissing{
			FilePath: path,
			Feature:  "all-templates",
		}
	}
	if sc.TemplatesEnabled == "" {
		return &ErrorMissing{
			FilePath: path,
			Feature:  "enabled-templates",
		}
	}

	return
}

func (s *Service) readSchema(path string) (*gqlast.Schema, *gqt.Parser, error) {
	if path == "" {
		p, err := gqt.NewParser(nil)
		return nil, p, err
	}

	f, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, &ErrorMissing{
			FilePath: path,
			Feature:  "schema",
		}
	}

	schema, err := gqlparser.LoadSchema(&gqlast.Source{
		Name:  path,
		Input: string(f),
	})
	if err != nil {
		return nil, nil, &ErrorIllegal{
			FilePath: path,
			Feature:  "schema",
			Message:  fmt.Sprintf("invalid schema: %v", err.Error()),
		}
	}

	gqtParser, err := gqt.NewParser([]gqt.Source{
		{Name: path, Content: string(f)},
	})
	return schema, gqtParser, err
}

func (s *Service) readAllTemplates(path string, p *gqt.Parser) (err error) {
	dir, err := os.ReadDir(path)
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

		file, err := openFile(filepath.Join(path, tf.Name()))
		if err != nil {
			return err
		}
		h, err := calculateHash(file)
		if err != nil {
			return err
		}

		t, err := readTemplate(file, p)
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

func (s *Service) readEnabledTemplates(path string) (err error) {
	dir, err := os.ReadDir(path)
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

		file, err := openFile(filepath.Join(path, tf.Name()))
		if err != nil {
			return err
		}

		h, err := calculateHash(file)
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

func readTemplate(file *os.File, p *gqt.Parser) (t *Template, err error) {
	filePath := file.Name()

	id := strings.ToLower(
		strings.TrimSuffix(
			filepath.Base(filePath), filepath.Ext(filePath),
		),
	)
	if err := ValidateID(id); err != "" {
		return nil, &ErrorIllegal{
			FilePath: filePath,
			Feature:  "id",
			Message:  err,
		}
	}

	b, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading template %q: %w", filePath, err)
	}

	meta, template, err := metadata.Parse(b)
	if err != nil {
		return nil, &ErrorIllegal{
			FilePath: filePath,
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
			FilePath: filePath,
			Feature:  "template",
			Message:  msg.String(),
		}
	}

	t = &Template{
		ID:          id,
		Source:      template,
		GQTTemplate: doc,
		Name:        meta.Name,
		Tags:        meta.Tags,
		FilePath:    filePath,
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

type ErrorDuplicate struct {
	Original  string
	Duplicate string
}

func (e ErrorDuplicate) Error() string {
	return fmt.Sprintf("%s is a duplicate of %s", e.Duplicate, e.Original)
}

type ErrorAlien struct {
	Items []string
}

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

func openFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("getting information about file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return nil, fmt.Errorf("reading link: %w", err)
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	return f, nil
}

// calculateHash returns a base32 encoded MD5 hash of file.
func calculateHash(file *os.File) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("copying file to md5 hash: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seeking start in file: %w", err)
	}
	sum := h.Sum(nil)
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum)
	return s, nil
}
