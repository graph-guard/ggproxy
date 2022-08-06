package server_test

import (
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/graph-guard/gguard-proxy/config"
	"github.com/graph-guard/gguard-proxy/server"
	plog "github.com/phuslu/log"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestServer(t *testing.T) {
	// Launch destination server
	lnDest := fasthttputil.NewInmemoryListener()
	defer lnDest.Close()

	type Resp struct {
		Status  int
		Headers map[string]string
		Body    string
	}
	type Req struct {
		Request  *fasthttp.RequestCtx
		Response chan *Resp
	}

	forwarded := make(chan Req, 1)

	go func() {
		s := &fasthttp.Server{
			Handler: func(ctx *fasthttp.RequestCtx) {
				resp := make(chan *Resp)
				forwarded <- Req{
					Request:  ctx,
					Response: resp,
				}
				r := <-resp
				ctx.Response.SetStatusCode(r.Status)
				for k, v := range r.Headers {
					ctx.Response.Header.Set(k, v)
				}
				ctx.Response.SetBody([]byte(r.Body))
			},
		}
		require.NoError(t, s.Serve(lnDest))
	}()

	// Launch proxy server
	conf, err := config.ReadConfig(validFS(), ".")
	require.NoError(t, err)

	log := plog.Logger{
		Level:      plog.DebugLevel,
		TimeField:  "time",
		TimeFormat: "23:59:59",
		Writer:     &plog.IOWriter{Writer: io.Discard},
	}
	s := server.New(
		conf,
		":8000",
		true,
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

	lnProxy := fasthttputil.NewInmemoryListener()
	defer lnProxy.Close()
	go func() {
		s.Serve(lnProxy)
	}()

	defer func() {
		require.NoError(t, s.Shutdown())
	}()

	clientProxy := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return lnProxy.Dial()
		},
	}

	for _, q := range []struct {
		query    string
		endpoint string
		method   string
		expect   int
	}{
		{
			method:   fasthttp.MethodPost,
			query:    `{"query": "mutation X { a { a0(a0_0: [ 0 ]) } }", "operationName": "X"}`,
			endpoint: "/service_a",
			expect:   fasthttp.StatusOK,
		},
		{
			method:   fasthttp.MethodPost,
			query:    `{"query": "query X { b { b0(b0_0: [ \"not a mutation\" ]) } }", "operationName": "X"}`,
			endpoint: "/service_b",
			expect:   fasthttp.StatusOK,
		},
		{
			method:   fasthttp.MethodPost,
			query:    `{"query": "mutation X { a { a0(a0_0: [ -1 ]) } }", "operationName": "X"}`,
			endpoint: "/service_a",
			expect:   fasthttp.StatusForbidden,
		},
		{
			method:   fasthttp.MethodPost,
			query:    `{"query": "query X { b { b0(b0_0: [ \"not a mutation\" ]) } }", "operationName": "X"}`,
			endpoint: "/service_c",
			expect:   fasthttp.StatusNotFound,
		},
		{
			method:   fasthttp.MethodPut,
			query:    `{"query": "mutation X { a { a0(a0_0: [ 0 ]) } }", "operationName": "X"}`,
			endpoint: "/service_a",
			expect:   fasthttp.StatusMethodNotAllowed,
		},
		{
			method:   fasthttp.MethodPost,
			query:    `{"query": "query X { b { b0(b0_0: [ \"not a mutation\" ]) } }"}`,
			endpoint: "/service_b",
			expect:   fasthttp.StatusBadRequest,
		},
		{
			// Syntax error in the query, missing '}'
			method:   fasthttp.MethodPost,
			query:    `{"query": "mutation X { a { a0(a0_0: [ 0 ]) }", "operationName": "X"}`,
			endpoint: "/service_a",
			expect:   fasthttp.StatusBadRequest,
		},
	} {
		t.Run("", func(t *testing.T) {
			go func() {
				f := <-forwarded
				f.Response <- &Resp{Status: 200}
			}()
			doRequest(
				t, clientProxy, q.method, "localhost:8000", q.endpoint,
				func(r *fasthttp.Request) {
					r.Header.Set("Content-Type", "application/json")
					r.SetBody([]byte(q.query))
				},
				func(r *fasthttp.Response) {
					require.Equal(t, q.expect, r.StatusCode())
				},
			)
		})
	}
}

func validFS() fstest.MapFS {
	return fstest.MapFS{
		config.ServerConfigFile1: &fstest.MapFile{
			Data: lines(
				`host: localhost:8080`,
			),
		},

		/**** SERVICE A ****/
		filepath.Join(
			config.ServicesEnabledDir,
			"service_a",
			config.ServiceConfigFile1,
		): &fstest.MapFile{
			Data: lines(
				`name: "Service A"`,
				`forward_url: "http://localhost:8081/service_a"`,
				`forward_reduced: false`,
			),
		},
		filepath.Join(
			config.ServicesEnabledDir,
			"service_a",
			config.TemplatesEnabledDir,
			"template_a1.gqt",
		): &fstest.MapFile{
			Data: lines(
				"---",
				`name: "Template A1"`,
				"tags:",
				"  - tag_a",
				"  - mutation",
				"---",
				"mutation {",
				"	a {",
				"		a0(",
				"			a0_0: val = [ val <= 0 ] && val != [ val = -1 ]",
				"		)",
				"	}",
				"}",
			),
		},

		/**** SERVICE B ****/
		filepath.Join(
			config.ServicesEnabledDir,
			"service_b",
			config.ServiceConfigFile1,
		): &fstest.MapFile{
			Data: lines(
				`name: "Service B"`,
				`forward_url: "http://localhost:8081/service_b"`,
				`forward_reduced: false`,
			),
		},
		filepath.Join(
			config.ServicesEnabledDir,
			"service_b",
			config.TemplatesEnabledDir,
			"template_b1.gqt",
		): &fstest.MapFile{
			Data: lines(
				"---",
				`name: "Template B1"`,
				"tags:",
				"  - tag_a",
				"  - query",
				"---",
				"query {",
				"	b {",
				"		b0(",
				"			b0_0: val = [ bytelen > 0 ] && val != [ val = \"kek\" ]",
				"		)",
				"	}",
				"}",
			),
		},
	}
}

func lines(lines ...string) []byte {
	var b strings.Builder
	for i := range lines {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func doRequest(
	t *testing.T,
	client *fasthttp.Client,
	method, host, path string,
	prepareReq func(*fasthttp.Request),
	checkReq func(*fasthttp.Response),
) {
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

	if checkReq != nil {
		checkReq(resp)
	}
}
