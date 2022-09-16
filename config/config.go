package config

import (
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
	"github.com/graph-guard/gqt"
	yaml "gopkg.in/yaml.v3"
)

var ConfigFileExtension = regexp.MustCompile(`\.(yml|yaml)$`)
var TemplateFileExtension = regexp.MustCompile(`\.gqt$`)

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
	ServicesAll     []*Service
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
	TemplatesAll     []*Template
	TemplatesEnabled []*Template
	ForwardReduced   bool
}

type Template struct {
	ID       string
	Source   []byte
	Document gqt.Doc
	Name     string
	Tags     []string
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
}

func ReadServerConfig(path string) (*Config, error) {
	dirPath := filepath.Dir(path)
	sc, err := readServerConfigFile(path)
	if err != nil {
		return nil, err
	}

	// Set default config values
	conf := &Config{
		Proxy: ProxyServerConfig{
			MaxReqBodySizeBytes: DefaultMaxReqBodySize,
		},
		API: &APIServerConfig{},
	}

	conf.Proxy.Host = sc.Proxy.Host
	if sc.Proxy.TLS != nil {
		conf.Proxy.TLS.CertFile = sc.Proxy.TLS.CertFile
		conf.Proxy.TLS.KeyFile = sc.Proxy.TLS.KeyFile
	}
	if sc.Proxy.MaxRequestBodySizeBytes == nil {
		conf.Proxy.MaxReqBodySizeBytes = DefaultMaxReqBodySize
	} else {
		if *sc.Proxy.MaxRequestBodySizeBytes < MinReqBodySize {
			return nil, &ErrorIllegal{
				FilePath: path,
				Feature:  "proxy.max-request-body-size",
				Message:  msgMaxReqBodySizeTooSmall,
			}
		}
	}
	if sc.API == nil {
		// Disable API server
		conf.API = nil
	} else {
		conf.API.Host = sc.API.Host
		if sc.API.TLS != nil {
			conf.API.TLS.CertFile = sc.API.TLS.CertFile
			conf.API.TLS.KeyFile = sc.API.TLS.KeyFile
		}
	}

	var servicesAllPath, servicesEnabledPath string
	servicesAllPath = sc.ServicesAll
	if servicesAllPath == "" {
		return nil, fmt.Errorf(
			"path to all services (all-services) is not defined in %s",
			path,
		)
	}
	servicesEnabledPath = sc.ServicesEnabled
	if servicesEnabledPath == "" {
		return nil, fmt.Errorf(
			"path to enabled services (enabled-services) is not defined in %s",
			path,
		)
	}
	if !strings.HasPrefix(servicesAllPath, "/") {
		servicesAllPath = filepath.Join(dirPath, servicesAllPath)
	}
	if !strings.HasPrefix(servicesEnabledPath, "/") {
		servicesEnabledPath = filepath.Join(dirPath, servicesEnabledPath)
	}

	// reading all services
	s, err := readServices(servicesAllPath)
	if err != nil {
		return nil, err
	}
	conf.ServicesAll = s

	// reading enabled services
	s, err = readServices(servicesEnabledPath)
	if err != nil {
		return nil, err
	}
	conf.ServicesEnabled = s

	var aliens []string
	for _, s := range conf.ServicesEnabled {
		if !contains(conf.ServicesAll, s, func(a, b *Service) bool { return a.ID == b.ID }) {
			aliens = append(aliens, filepath.Join(s.Path, s.ID))
		}
	}

	if len(aliens) > 0 {
		return nil, &ErrorAlien{
			Items: aliens,
		}
	}

	return conf, nil
}

func readServerConfigFile(path string) (sc *serverConfig, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening server config file: %w", err)
	}
	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err := d.Decode(&sc); err != nil {
		return nil, &ErrorIllegal{
			FilePath: path,
			Feature:  "syntax",
			Message:  err.Error(),
		}
	}

	if sc.Proxy.Host == "" {
		return nil, &ErrorMissing{
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
			return nil, &ErrorMissing{
				FilePath: path,
				Feature:  "proxy.tls.key-file",
			}
		case (c.KeyFile != "" && c.CertFile == "") ||
			(c.KeyFile == "" && c.CertFile == ""):
			return nil, &ErrorMissing{
				FilePath: path,
				Feature:  "proxy.tls.cert-file",
			}
		}
	}

	if sc.API != nil {
		c := sc.API
		if c.Host == "" {
			return nil, &ErrorMissing{
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
				return nil, &ErrorMissing{
					FilePath: path,
					Feature:  "api.tls.key-file",
				}
			case (c.KeyFile != "" && c.CertFile == "") ||
				(c.KeyFile == "" && c.CertFile == ""):
				return nil, &ErrorMissing{
					FilePath: path,
					Feature:  "api.tls.cert-file",
				}
			}
		}
	}

	return
}

func readServices(path string) ([]*Service, error) {
	d, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading services directory: %w", err)
	}
	var services []*Service
	for _, o := range d {
		if o.IsDir() {
			continue
		}
		if !ConfigFileExtension.MatchString(o.Name()) {
			// Ignore non-yaml files
			continue
		}
		s, err := readServiceConfig(filepath.Join(path, o.Name()))
		if err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

func readServiceConfig(path string) (*Service, error) {
	dirPath := filepath.Dir(path)
	sc, err := readServiceConfigFile(path)
	if err != nil {
		return nil, err
	}

	var templatesAllPath, templatesEnabledPath string
	templatesAllPath = sc.TemplatesAll
	if templatesAllPath == "" {
		return nil, fmt.Errorf(
			"path to all templates (all-templates) is not defined in %s",
			path,
		)
	}
	templatesEnabledPath = sc.TemplatesEnabled
	if templatesEnabledPath == "" {
		return nil, fmt.Errorf(
			"path to enabled templates (enabled-templates) is not defined in %s",
			path,
		)
	}
	if !strings.HasPrefix(templatesAllPath, "/") {
		templatesAllPath = filepath.Join(dirPath, templatesAllPath)
	}
	if !strings.HasPrefix(templatesEnabledPath, "/") {
		templatesEnabledPath = filepath.Join(dirPath, templatesEnabledPath)
	}

	fileName := filepath.Base(path)
	id := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if err := ValidateID(id); err != "" {
		return nil, &ErrorIllegal{
			FilePath: path,
			Feature:  "id",
			Message:  err,
		}
	}
	id = strings.ToLower(id)

	s := &Service{
		ID: id,
	}
	s.Path = sc.Path
	s.ForwardURL = sc.ForwardURL
	s.ForwardReduced = sc.ForwardReduced

	// reading all templates
	t, err := readTemplates(
		templatesAllPath,
	)
	if err != nil {
		return nil, err
	}
	s.TemplatesAll = t

	// reading enabled templates
	t, err = readTemplates(
		templatesEnabledPath,
	)
	if err != nil {
		return nil, err
	}
	s.TemplatesEnabled = t

	var aliens []string
	for _, t := range s.TemplatesEnabled {
		if !contains(s.TemplatesAll, t, func(a, b *Template) bool { return a.ID == b.ID }) {
			aliens = append(aliens, filepath.Join(s.Path, s.ID))
		}
	}

	if len(aliens) > 0 {
		return nil, &ErrorAlien{
			Items: aliens,
		}
	}

	return s, nil
}

func readServiceConfigFile(path string) (sc *serviceConfig, err error) {
	f, err := openFile(path)
	if err != nil {
		return nil, err
	}
	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err := d.Decode(&sc); err != nil {
		return nil, &ErrorIllegal{
			FilePath: path,
			Feature:  "syntax",
			Message:  err.Error(),
		}
	}
	if sc.Path == "" {
		return nil, &ErrorMissing{
			FilePath: path,
			Feature:  "path",
		}
	}
	if err := validatePath(sc.Path); err != nil {
		return nil, &ErrorIllegal{
			FilePath: path,
			Feature:  "path",
			Message:  err.Error(),
		}
	}
	if sc.ForwardURL == "" {
		return nil, &ErrorMissing{
			FilePath: path,
			Feature:  "forward-url",
		}
	}
	if err := validateURL(sc.ForwardURL); err != nil {
		return nil, &ErrorIllegal{
			FilePath: path,
			Feature:  "forward-url",
			Message:  err.Error(),
		}
	}
	return
}

func readTemplates(
	path string,
) (t []*Template, err error) {
	dir, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading templates directory: %w", err)
	}

	for _, o := range dir {
		if o.IsDir() {
			continue
		}
		n := o.Name()
		if !TemplateFileExtension.MatchString(n) {
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

		src, err := openFile(p)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(src)
		if err != nil {
			return nil, fmt.Errorf("reading template %q: %w", p, err)
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

type ErrorAlien struct {
	Items []string
}

func (e ErrorAlien) Error() string {
	var b strings.Builder
	b.WriteString("configs are not defined in the pool (all-configs): ")
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

var ErrPathNotAbsolute = errors.New("path is not starting with /")
var ErrURLProtocolProblem = errors.New("protocol is not supported or undefined")
var ErrURLNoHost = errors.New("host is not defined")

var ValidProtocolSchemes = []string{"http", "https"}

func validateURL(url string) error {
	u, err := neturl.Parse(url)
	if err != nil {
		return err
	}

	if !contains(ValidProtocolSchemes, u.Scheme, func(a, b string) bool { return a == b }) {
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

func contains[T any](arr []T, x T, equal func(a, b T) bool) bool {
	for _, el := range arr {
		if equal(el, x) {
			return true
		}
	}

	return false
}

func openFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("getting information about file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		path, err = os.Readlink(path)
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
