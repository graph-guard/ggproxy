package config

import (
	"bytes"
	"crypto/md5"
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
	Hash             []byte
	ID               string
	Path             string
	ForwardURL       string
	TemplatesAll     []*Template
	TemplatesEnabled []*Template
	ForwardReduced   bool
}

type Template struct {
	Hash     []byte
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
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening server config file: %w", err)
	}
	sc, err := readServerConfigFile(f)
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
	s, err := readServices(servicesAllPath, nil)
	if err != nil {
		return nil, err
	}
	conf.ServicesAll = s

	// reading enabled services
	s, err = readServices(servicesEnabledPath, conf.ServicesAll)
	if err != nil {
		return nil, err
	}
	conf.ServicesEnabled = s

	return conf, nil
}

func readServerConfigFile(file *os.File) (sc *serverConfig, err error) {
	filePath, _ := filepath.Abs(file.Name())
	d := yaml.NewDecoder(file)
	d.KnownFields(true)
	if err := d.Decode(&sc); err != nil {
		return nil, &ErrorIllegal{
			FilePath: filePath,
			Feature:  "syntax",
			Message:  err.Error(),
		}
	}

	if sc.Proxy.Host == "" {
		return nil, &ErrorMissing{
			FilePath: filePath,
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
				FilePath: filePath,
				Feature:  "proxy.tls.key-file",
			}
		case (c.KeyFile != "" && c.CertFile == "") ||
			(c.KeyFile == "" && c.CertFile == ""):
			return nil, &ErrorMissing{
				FilePath: filePath,
				Feature:  "proxy.tls.cert-file",
			}
		}
	}

	if sc.API != nil {
		c := sc.API
		if c.Host == "" {
			return nil, &ErrorMissing{
				FilePath: filePath,
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
					FilePath: filePath,
					Feature:  "api.tls.key-file",
				}
			case (c.KeyFile != "" && c.CertFile == "") ||
				(c.KeyFile == "" && c.CertFile == ""):
				return nil, &ErrorMissing{
					FilePath: filePath,
					Feature:  "api.tls.cert-file",
				}
			}
		}
	}

	return
}

func readServices(path string, existingServices []*Service) ([]*Service, error) {
	d, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading services directory: %w", err)
	}
	var services []*Service
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
			return nil, err
		}
		h, err := calculateHash(file)
		if err != nil {
			return nil, err
		}

		if existingServices != nil {
			if idx := contains(
				existingServices, h, func(a *Service, h []byte) bool { return bytes.Equal(a.Hash, h) },
			); idx != -1 {
				services = append(services, existingServices[idx])
			} else {
				return nil, &ErrorAlien{
					Items: []string{sf.Name()},
				}
			}
		} else {
			s, err := readServiceConfig(file)
			if err != nil {
				return nil, err
			}
			s.Hash = h
			services = append(services, s)
		}
	}
	return services, nil
}

func readServiceConfig(file *os.File) (*Service, error) {
	filePath, _ := filepath.Abs(file.Name())
	dirPath := filepath.Dir(file.Name())

	sc, err := readServiceConfigFile(file)
	if err != nil {
		return nil, err
	}

	var templatesAllPath, templatesEnabledPath string
	templatesAllPath = sc.TemplatesAll
	if templatesAllPath == "" {
		return nil, fmt.Errorf(
			"path to all templates (all-templates) is not defined in %s",
			filePath,
		)
	}
	templatesEnabledPath = sc.TemplatesEnabled
	if templatesEnabledPath == "" {
		return nil, fmt.Errorf(
			"path to enabled templates (enabled-templates) is not defined in %s",
			filePath,
		)
	}
	if !strings.HasPrefix(templatesAllPath, "/") {
		templatesAllPath = filepath.Join(dirPath, templatesAllPath)
	}
	if !strings.HasPrefix(templatesEnabledPath, "/") {
		templatesEnabledPath = filepath.Join(dirPath, templatesEnabledPath)
	}

	id := strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name()))
	if err := ValidateID(id); err != "" {
		return nil, &ErrorIllegal{
			FilePath: filePath,
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
		templatesAllPath, nil,
	)
	if err != nil {
		return nil, err
	}
	s.TemplatesAll = t

	// reading enabled templates
	t, err = readTemplates(
		templatesEnabledPath, s.TemplatesAll,
	)
	if err != nil {
		return nil, err
	}
	s.TemplatesEnabled = t

	return s, nil
}

func readServiceConfigFile(file *os.File) (sc *serviceConfig, err error) {
	filePath, _ := filepath.Abs(file.Name())

	d := yaml.NewDecoder(file)
	d.KnownFields(true)
	if err := d.Decode(&sc); err != nil {
		return nil, &ErrorIllegal{
			FilePath: filePath,
			Feature:  "syntax",
			Message:  err.Error(),
		}
	}
	if sc.Path == "" {
		return nil, &ErrorMissing{
			FilePath: filePath,
			Feature:  "path",
		}
	}
	if err := validatePath(sc.Path); err != nil {
		return nil, &ErrorIllegal{
			FilePath: filePath,
			Feature:  "path",
			Message:  err.Error(),
		}
	}
	if sc.ForwardURL == "" {
		return nil, &ErrorMissing{
			FilePath: filePath,
			Feature:  "forward-url",
		}
	}
	if err := validateURL(sc.ForwardURL); err != nil {
		return nil, &ErrorIllegal{
			FilePath: filePath,
			Feature:  "forward-url",
			Message:  err.Error(),
		}
	}

	return
}

func readTemplates(
	path string, existingTemplates []*Template,
) (templates []*Template, err error) {
	dir, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading templates directory: %w", err)
	}

	for _, t := range dir {
		if t.IsDir() {
			continue
		}
		n := t.Name()
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

		file, err := openFile(p)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(file)
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

		h, err := calculateHash(file)
		if err != nil {
			return nil, err
		}

		if existingTemplates != nil {
			idx := contains(
				existingTemplates, h,
				func(a *Template, h []byte) bool { return bytes.Equal(a.Hash, h) },
			)
			if idx != -1 {
				templates = append(templates, existingTemplates[idx])
			} else {
				return nil, &ErrorAlien{
					Items: []string{t.Name()},
				}
			}
		} else {
			templates = append(templates, &Template{
				Hash:     h,
				ID:       id,
				Source:   template,
				Document: doc,
				Name:     meta.Name,
				Tags:     meta.Tags,
			})
		}
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

type ErrorAlien struct {
	Items []string
}

func (e ErrorAlien) Error() string {
	var b strings.Builder
	b.WriteString("configs are not defined in the pool: ")
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

func contains[T any, V any](arr []T, x V, equal func(a T, b V) bool) int {
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

func calculateHash(file *os.File) (sum []byte, err error) {
	h := md5.New()

	_, err = io.Copy(h, file)
	if err != nil {
		return nil, err
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	sum = h.Sum(nil)

	return
}
