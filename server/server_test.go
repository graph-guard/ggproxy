package server_test

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"io/fs"
	"net"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/graph-guard/gguard-proxy/config"
	"github.com/graph-guard/gguard-proxy/config/metadata"
	"github.com/graph-guard/gguard-proxy/server"
	plog "github.com/phuslu/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"gopkg.in/yaml.v3"
)

//go:embed tests
var testsFS embed.FS

type Setup struct {
	Name   string
	Config *config.Config
	Tests  []Test
}

type Test struct {
	Name               string
	InputMeta          InputMeta
	InputBody          string
	ExpectResponseMeta ExpectResponseMeta
	ExpectResponseBody string
	*Destination
}

type Destination struct {
	// What is expected to arrive at the destination server
	ExpectForwardedMeta ExpectForwardedMeta
	ExpectForwardedBody string

	// What the destination server will send back
	ResponseMeta ResponseMeta
	ResponseBody string
}

type (
	InputMeta struct {
		Method   string            `yaml:"method"`
		Endpoint string            `yaml:"endpoint"`
		Headers  map[string]string `yaml:"headers"`
	}
	ExpectForwardedMeta struct {
		Headers map[string]string `yaml:"headers"`
	}
	ResponseMeta struct {
		Status  int               `yaml:"status"`
		Headers map[string]string `yaml:"headers"`
	}
	ExpectResponseMeta struct {
		Status  int               `yaml:"status"`
		Headers map[string]string `yaml:"headers"`
	}
)

func TestServer(t *testing.T) {
	setups := GetSetups(t, testsFS, "tests")
	for _, setup := range setups {
		t.Run(setup.Name, func(t *testing.T) {
			clientProxy, forwarded, respSetter := launchSetup(t, setup)

			for _, test := range setup.Tests {
				t.Run(test.Name, func(t *testing.T) {
					if test.Destination != nil {
						respSetter.Set(&Resp{
							Body:         test.Destination.ResponseBody,
							ResponseMeta: test.Destination.ResponseMeta,
						})
					} else {
						respSetter.Set(nil)
					}
					respMeta, respBody := doRequest(
						t, clientProxy,
						test.InputMeta.Method,
						"localhost:8000",
						test.InputMeta.Endpoint,
						func(r *fasthttp.Request) {
							r.Header.Set("Content-Type", "application/json")
							r.SetBody([]byte(test.InputBody))
						},
					)

					if test.Destination != nil {
						forwarded := <-forwarded
						compareHeaders(
							t,
							"forwarded",
							test.ExpectForwardedMeta.Headers,
							forwarded.ExpectForwardedMeta.Headers,
						)
						assert.Equal(
							t, test.ExpectForwardedBody,
							forwarded.Body,
							"unexpected body was forwarded to destination",
						)
					}

					// Compare results
					if e := test.ExpectResponseMeta.Status; e != respMeta.Status {
						t.Errorf(
							"unexpected response status: %d; expected: %d",
							respMeta.Status, e,
						)
					}
					compareHeaders(
						t, "response", test.ExpectResponseMeta.Headers, respMeta.Headers,
					)
					assert.Equal(
						t, test.ExpectResponseBody,
						respBody,
						"unexpected response body",
					)
				})
			}
		})
	}
}

func GetSetups(t *testing.T, filesystem fs.FS, path string) []Setup {
	var setups []Setup

	d, err := fs.ReadDir(filesystem, path)
	require.NoError(t, err)
	for _, setupDir := range d {
		if !setupDir.IsDir() {
			continue
		}
		n := setupDir.Name()
		if !strings.HasPrefix(n, "setup_") {
			t.Logf("ignoring %q", filepath.Join(n))
			continue
		}

		c, err := config.ReadConfig(testsFS, filepath.Join(path, n))
		require.NoError(t, err)

		tests := GetTests(t, filesystem, filepath.Join(path, n))

		setups = append(setups, Setup{
			Name:   n,
			Config: c,
			Tests:  tests,
		})
	}

	return setups
}

func GetTests(t *testing.T, filesystem fs.FS, root string) []Test {
	var tests []Test
	d, err := fs.ReadDir(filesystem, root)
	require.NoError(t, err)
	for _, testDir := range d {
		if !testDir.IsDir() {
			continue
		}
		n := testDir.Name()
		if !strings.HasPrefix(n, "test_") {
			continue
		}

		var test Test
		{ // Read Input file
			fInput, err := filesystem.Open(filepath.Join(root, n, "input.txt"))
			require.NoError(t, err)
			bInput, err := io.ReadAll(fInput)
			require.NoError(t, err)
			header, body, err := metadata.Split(bInput)
			require.NoError(t, err)

			// Parse metadata
			var m InputMeta
			d := yaml.NewDecoder(bytes.NewReader(header))
			d.KnownFields(true)
			err = d.Decode(&m)
			require.NoError(t, err)

			test.InputBody = string(body)
			test.InputMeta = m
		}

		{ // Read expect_response.txt file
			f, err := filesystem.Open(
				filepath.Join(root, n, "expect_response.txt"),
			)
			require.NoError(t, err)
			b, err := io.ReadAll(f)
			require.NoError(t, err)
			header, body, err := metadata.Split(b)
			require.NoError(t, err)

			// Parse metadata
			var m ExpectResponseMeta
			if header != nil {
				d := yaml.NewDecoder(bytes.NewReader(header))
				d.KnownFields(true)
				err = d.Decode(&m)
				require.NoError(t, err)
			}

			test.ExpectResponseBody = string(body)
			test.ExpectResponseMeta = m
		}

		fExpectForwarded, err := filesystem.Open(
			filepath.Join(root, n, "expect_forwarded.txt"),
		)
		if errors.Is(err, fs.ErrNotExist) {
			fExpectForwarded = nil
		}
		fResponse, err := filesystem.Open(
			filepath.Join(root, n, "response.txt"),
		)
		if errors.Is(err, fs.ErrNotExist) {
			fResponse = nil
		}

		if fExpectForwarded == nil && fResponse != nil ||
			fExpectForwarded != nil && fResponse == nil {
			t.Fatalf(`expect_forwarded.txt and "response.txt" must ` +
				`either both exist or not exist`)
		} else if fExpectForwarded == nil && fResponse == nil {
			// Don't expect the request to arrive at the destination server
			tests = append(tests, test)
			continue
		}

		test.Destination = new(Destination)

		{ // Read expect_forwarded.txt file
			b, err := io.ReadAll(fExpectForwarded)
			require.NoError(t, err)
			header, body, err := metadata.Split(b)
			require.NoError(t, err)

			// Parse metadata
			var m ExpectForwardedMeta
			if header != nil {
				d := yaml.NewDecoder(bytes.NewReader(header))
				d.KnownFields(true)
				err = d.Decode(&m)
				require.NoError(t, err)
			}

			test.Destination.ExpectForwardedBody = string(body)
			test.Destination.ExpectForwardedMeta = m
		}

		{ // Read response.txt file
			b, err := io.ReadAll(fResponse)
			require.NoError(t, err)
			header, body, err := metadata.Split(b)
			require.NoError(t, err)

			// Parse metadata
			var m ResponseMeta
			if header != nil {
				d := yaml.NewDecoder(bytes.NewReader(header))
				d.KnownFields(true)
				err = d.Decode(&m)
				require.NoError(t, err)
			}

			test.Destination.ResponseBody = string(body)
			test.Destination.ResponseMeta = m
		}

		tests = append(tests, test)
	}
	return tests
}

type Resp struct {
	ResponseMeta
	Body string
}
type ReceivedReq struct {
	Body string
	ExpectForwardedMeta
}

func launchSetup(t *testing.T, s Setup) (
	clientProxy *fasthttp.Client,
	forwarded <-chan ReceivedReq,
	resp *Syncronized[*Resp],
) {
	resp = new(Syncronized[*Resp])

	lnDest := fasthttputil.NewInmemoryListener()
	t.Cleanup(func() { lnDest.Close() })

	lnProxy := fasthttputil.NewInmemoryListener()
	t.Cleanup(func() { lnProxy.Close() })

	forwardedRW := make(chan ReceivedReq, 1)
	forwarded = forwardedRW

	go func() {
		s := &fasthttp.Server{
			Handler: func(ctx *fasthttp.RequestCtx) {
				// Send the received request context for the check
				var meta ExpectForwardedMeta
				meta.Headers = make(map[string]string, ctx.Request.Header.Len())
				ctx.Request.Header.VisitAll(func(key, value []byte) {
					meta.Headers[string(key)] = string(value)
				})
				forwardedRW <- ReceivedReq{
					ExpectForwardedMeta: meta,
					Body:                string(ctx.Request.Body()),
				}

				// Send response
				r := resp.Get()
				if r == nil {
					ctx.Error(
						fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
						fasthttp.StatusInternalServerError,
					)
					return
				}
				ctx.Response.SetStatusCode(r.ResponseMeta.Status)
				for k, v := range r.Headers {
					ctx.Response.Header.Set(k, v)
				}
				ctx.Response.SetBodyString(r.Body)
			},
		}
		if err := s.Serve(lnDest); err != nil {
			panic(err)
		}
	}()

	// Launch proxy server
	log := plog.Logger{
		Level:      plog.DebugLevel,
		TimeField:  "time",
		TimeFormat: "23:59:59",
		Writer:     &plog.IOWriter{Writer: &TestPrintWriter{T: t}},
	}
	server := server.New(
		s.Config,
		time.Second*10,
		time.Second*10,
		1024*64,
		1024*64,
		log,
		&fasthttp.Client{
			Dial: func(addr string) (net.Conn, error) {
				return lnDest.Dial()
			},
		},
	)

	go func() {
		server.Serve(lnProxy)
	}()

	clientProxy = &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return lnProxy.Dial()
		},
	}

	return
}

func doRequest(
	t *testing.T,
	client *fasthttp.Client,
	method, host, path string,
	prepareReq func(*fasthttp.Request),
) (meta ResponseMeta, body string) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	if prepareReq != nil {
		prepareReq(req)
	}
	req.Header.SetMethod(method)
	req.SetHost(host)
	req.URI().SetPath(path)

	err := client.Do(req, resp)
	require.NoError(t, err)

	meta.Status = resp.StatusCode()
	meta.Headers = make(map[string]string, resp.Header.Len())
	resp.Header.VisitAll(func(key, value []byte) {
		meta.Headers[string(key)] = string(value)
	})
	body = string(resp.Body())
	return
}

type Syncronized[T any] struct {
	lock sync.Mutex
	t    T
}

func (s *Syncronized[T]) Get() T {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t
}

func (s *Syncronized[T]) Set(t T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.t = t
}

func compareHeaders(t *testing.T, title string, expected, actual map[string]string) {
	t.Helper()
	e := make(map[string]string, len(expected))
	for k, v := range expected {
		e[k] = v
	}
	a := make(map[string]string, len(actual))
	for k, v := range actual {
		a[k] = v
	}
	for k, ev := range expected {
		delete(e, k)
		delete(a, k)
		if av, ok := actual[k]; ok {
			expr, err := regexp.Compile(ev)
			if err != nil {
				t.Fatalf(
					"compiling regexp for %s header %q (%q): %v",
					title, k, ev, err,
				)
			}
			if !expr.MatchString(av) {
				t.Errorf(
					"%s header %q expected regexp: %q; received: %q",
					title, k, ev, av,
				)
			}
		}
	}
	for k, v := range e {
		t.Errorf("missing %s header %q (%q)", title, k, v)
	}
	for k, v := range a {
		t.Errorf("unexpected %s header %q (%q)", title, k, v)
	}
}

type TestPrintWriter struct{ T *testing.T }

func (w *TestPrintWriter) Write(d []byte) (int, error) {
	w.T.Log(string(d))
	return len(d), nil
}
