package server_test

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/graph-guard/ggproxy/pkg/server"
	"github.com/graph-guard/ggproxy/pkg/testsetup"
	plog "github.com/phuslu/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestProxy(t *testing.T) {
	setups := GetSetups(t)
	for _, setup := range setups {
		t.Run(setup.Name, func(t *testing.T) {
			clientProxy, forwarded, respSetter, logs := launchSetup(t, setup)

			for _, test := range setup.Tests {
				t.Run(test.Name, func(t *testing.T) {
					if test.Destination != nil {
						body := test.Destination.Response.Body
						if j := test.Destination.Response.BodyJSON; j != nil {
							b, err := json.Marshal(j)
							require.NoError(t, err)
							body = string(b)
						}
						respSetter.Set(&SendResponse{
							Status:  test.Destination.Response.Status,
							Body:    body,
							Headers: copyMap(test.Destination.Response.Headers),
						})
					} else {
						respSetter.Set(nil)
					}

					respStatus, respHeaders, respBody := doRequest(
						t, clientProxy,
						test.Client.Input.Method,
						"localhost:8000",
						test.Client.Input.Endpoint,
						func(r *fasthttp.Request) {
							r.Header.Set("Content-Type", "application/json")
							body := test.Client.Input.Body
							if j := test.Client.Input.BodyJSON; j != nil {
								b, err := json.Marshal(j)
								require.NoError(t, err)
								body = string(b)
							}
							r.SetBodyString(body)
						},
					)

					if test.Destination != nil {
						var f ReceivedRequest
						ok := false
						select {
						case x := <-forwarded:
							ok = true
							f = x
						default:
							t.Errorf("the request wansn't forwarded as expected")
						}
						if ok {
							compareHeaders(
								t, "forwarded",
								test.Destination.ExpectForwarded.Headers,
								f.Headers,
							)
							j := test.Destination.ExpectForwarded.BodyJSON
							body := test.Destination.ExpectForwarded.Body
							if j != nil {
								b, err := json.Marshal(j)
								require.NoError(t, err)
								body = string(b)
							}
							assert.Equal(
								t, body, f.Body,
								"unexpected body was forwarded to destination",
							)
						}
					}

					// Compare results
					if e := test.Client.ExpectResponse.Status; e != respStatus {
						t.Errorf(
							"unexpected response status: %d; expected: %d",
							respStatus, e,
						)
					}
					compareHeaders(
						t, "response",
						test.Client.ExpectResponse.Headers, respHeaders,
					)
					{
						body := test.Client.ExpectResponse.Body
						if j := test.Client.ExpectResponse.BodyJSON; j != nil {
							b, err := json.Marshal(j)
							require.NoError(t, err)
							body = string(b)
						}
						assert.Equal(
							t, body, respBody,
							"unexpected response body",
						)
					}

					// Check logs
					logs.ReadLogs(func(m []map[string]any) {
						for i, x := range m {
							if i >= len(test.Logs) {
								t.Errorf("unexpected log: %v", m[i])
								continue
							}
							assert.Equal(t,
								test.Logs[i], x,
								"unexpected log at index %d", i,
							)
						}
					})
					logs.Reset()
				})
			}
		})
	}
}

func GetSetups(t *testing.T) []testsetup.Setup {
	return []testsetup.Setup{
		testsetup.Starwars(),
		testsetup.Test1(),
		testsetup.InputsSchema(),
	}
}

type SendResponse struct {
	Status  int
	Body    string
	Headers map[string]string
}
type ReceivedRequest struct {
	Body    string
	Headers map[string]string
}

func launchSetup(t *testing.T, s testsetup.Setup) (
	clientProxy *fasthttp.Client,
	forwarded <-chan ReceivedRequest,
	resp *Syncronized[*SendResponse],
	logRecorder *LogRecorder,
) {
	resp = new(Syncronized[*SendResponse])

	lnDest := fasthttputil.NewInmemoryListener()
	t.Cleanup(func() { lnDest.Close() })

	lnProxy := fasthttputil.NewInmemoryListener()
	t.Cleanup(func() { lnProxy.Close() })

	forwardedRW := make(chan ReceivedRequest, 1)
	forwarded = forwardedRW

	go func() {
		s := &fasthttp.Server{
			Handler: func(ctx *fasthttp.RequestCtx) {
				// Send the received request context for the check
				var rr ReceivedRequest
				rr.Headers = make(map[string]string, ctx.Request.Header.Len())
				ctx.Request.Header.VisitAll(func(key, value []byte) {
					rr.Headers[string(key)] = string(value)
				})
				rr.Body = string(ctx.Request.Body())
				forwardedRW <- rr

				// Send response
				sr := resp.Get()
				if sr == nil {
					ctx.Error(
						fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
						fasthttp.StatusInternalServerError,
					)
					return
				}
				ctx.Response.SetStatusCode(sr.Status)
				for k, v := range sr.Headers {
					ctx.Response.Header.Set(k, v)
				}
				ctx.Response.SetBodyString(sr.Body)
			},
		}
		if err := s.Serve(lnDest); err != nil {
			panic(err)
		}
	}()

	// Launch proxy server
	logRecorder = new(LogRecorder)
	log := plog.Logger{
		Level:      plog.DebugLevel,
		TimeField:  "time",
		TimeFormat: "23:59:59",
		Writer:     &plog.IOWriter{Writer: logRecorder},
	}
	server := server.NewProxy(
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
		nil,
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
) (status int, headers map[string]string, body string) {
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

	status = resp.StatusCode()
	headers = make(map[string]string, resp.Header.Len())
	resp.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
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

type LogRecorder struct {
	Lock     sync.Mutex
	Recorded []map[string]any
}

func (w *LogRecorder) Write(d []byte) (int, error) {
	var m map[string]any
	if err := json.Unmarshal(d, &m); err != nil {
		return 0, fmt.Errorf("unmarshalling JSON: %w", err)
	}
	delete(m, "time") // We don't need to check the log time
	w.Lock.Lock()
	defer w.Lock.Unlock()
	w.Recorded = append(w.Recorded, m)
	return len(d), nil
}

func (w *LogRecorder) Reset() {
	w.Lock.Lock()
	defer w.Lock.Unlock()
	w.Recorded = nil
}

func (w *LogRecorder) ReadLogs(fn func([]map[string]any)) {
	w.Lock.Lock()
	defer w.Lock.Unlock()
	fn(w.Recorded)
}

func copyMap[K comparable, V any](m map[K]V) (copy map[K]V) {
	copy = make(map[K]V, len(m))
	for k, v := range m {
		copy[k] = v
	}
	return copy
}

// func checkLogs(t *testing.T, expected, actual []map[string]any) {
// 	for i, x := range expected {
// 		if i >= len(expected) {
// 			t.Errorf("unexpected log: %v", actual[i])
// 			continue
// 		}
// 		assert.Equal(t,
// 			expected[i], x,
// 			"unexpected log at index %d", i,
// 		)
// 	}
// }
