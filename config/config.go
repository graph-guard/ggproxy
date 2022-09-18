package config

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	neturl "net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/graph-guard/ggproxy/config/metadata"
	"github.com/graph-guard/ggproxy/utilities/container/hamap"
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
	Services        *hamap.Map[[]byte, *Service]
	ServicesEnabled []*Service
}

func (c *Config) Equal(d *Config) bool {
	eq := true
	c.Services.Visit(func(key []byte, value *Service) (stop bool) {
		v, ok := d.Services.Get(key)
		if !ok {
			eq = false
			return true
		}
		if !v.Equal(value) {
			eq = false
			return true
		}
		return
	})
	if !eq {
		return eq
	}

	less := func(a, b *Service) bool { return a.ID < b.ID }
	eq = eq &&
		reflect.DeepEqual(c.Proxy, d.Proxy) &&
		reflect.DeepEqual(c.API, d.API) &&
		cmp.Equal(c.ServicesEnabled, d.ServicesEnabled, cmpopts.SortSlices(less))

	return eq
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
	Templates        *hamap.Map[[]byte, *Template]
	TemplatesEnabled []*Template
	ForwardReduced   bool
	Enabled          bool
	FilePath         string
}

func (c *Service) Equal(d *Service) bool {
	less := func(a, b *Template) bool { return a.ID < b.ID }
	return c.ID == d.ID &&
		c.Path == d.Path &&
		c.ForwardURL == d.ForwardURL &&
		c.ForwardReduced == d.ForwardReduced &&
		c.Enabled == d.Enabled &&
		c.FilePath == d.FilePath &&
		reflect.DeepEqual(c.Templates, d.Templates) &&
		cmp.Equal(c.TemplatesEnabled, d.TemplatesEnabled, cmpopts.SortSlices(less))
}

type Template struct {
	ID       string
	Source   []byte
	Document gqt.Doc
	Name     string
	Tags     []string
	Enabled  bool
	FilePath string
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

func New(path string) (c *Config, err error) {
	// Set default config values
	c = &Config{
		Proxy:    ProxyServerConfig{},
		API:      &APIServerConfig{},
		Services: hamap.New[[]byte, *Service](0, nil),
	}
	err = c.readServerConfig(path)
	if err != nil {
		return nil, err
	}

	return
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

	err = validateServerConfig(sc, path)
	if err != nil {
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
	err = c.readAllServices(servicesAllPath)
	if err != nil {
		return err
	}

	// reading enabled services
	err = c.readEnabledServices(servicesEnabledPath)
	if err != nil {
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
		original, ok := c.Services.Get(h)
		if ok {
			return &ErrorDuplicate{
				Original:  original.FilePath,
				Duplicate: s.FilePath,
			}
		}
		c.Services.Set(h, s)
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
		s, ok := c.Services.Get(h)
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
	d := yaml.NewDecoder(file)
	d.KnownFields(true)
	if err := d.Decode(sc); err != nil {
		return nil, &ErrorIllegal{
			FilePath: filePath,
			Feature:  "syntax",
			Message:  err.Error(),
		}
	}

	err = validateServiceConfig(sc, filePath)
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
		Templates:      hamap.New[[]byte, *Template](0, nil),
		FilePath:       filePath,
		Path:           sc.Path,
		ForwardURL:     sc.ForwardURL,
		ForwardReduced: sc.ForwardReduced,
	}

	// reading all templates
	err = s.readAllTemplates(templatesAllPath)
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

func (s *Service) readAllTemplates(path string) (err error) {
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

		t, err := readTemplate(file)
		if err != nil {
			return err
		}

		original, ok := s.Templates.Get(h)
		if ok {
			return &ErrorDuplicate{
				Original:  original.FilePath,
				Duplicate: t.FilePath,
			}
		}
		s.Templates.Set(h, t)
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
		t, ok := s.Templates.Get(h)
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

func readTemplate(file *os.File) (t *Template, err error) {
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

	doc, errParser := gqt.Parse(template)
	if errParser.IsErr() {
		return nil, &ErrorIllegal{
			FilePath: filePath,
			Feature:  "template",
			Message:  errParser.Error(),
		}
	}

	t = &Template{
		ID:       id,
		Source:   template,
		Document: doc,
		Name:     meta.Name,
		Tags:     meta.Tags,
		FilePath: filePath,
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
