package server_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/graph-guard/gguard-proxy/server"
	"github.com/graph-guard/gguard/engines/rmap"
	"github.com/graph-guard/gqt"
	plog "github.com/phuslu/log"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestServer(t *testing.T) {
	services := []server.ServiceConfig{}
	for _, s := range []struct {
		rules       []string
		name        string
		source      string
		destination string
		queries     []string
	}{
		{
			rules: []string{`
				mutation {
					a {
						a0(
							a0_0: val = [ val <= 0 ] && val != [ val = -1 ]
						)
					}
				}`,
			},
			name:        "Service A",
			source:      "/service_a",
			destination: "localhost:8080/service_a",
		},
		{
			rules: []string{`
				query {
					b {
						b0(
							b0_0: val = [ bytelen > 0 ] && val != [ val = "kek" ]
						)
					}
				}
				`,
			},
			name:        "Service B",
			source:      "/service_b",
			destination: "localhost:8080/service_b",
		},
	} {
		rules := make([]gqt.Doc, len(s.rules))
		for i, r := range s.rules {
			rd, err := gqt.Parse([]byte(r))
			if err.IsErr() {
				fmt.Println(err)
			}
			rules[i] = rd
		}
		rm, err := rmap.New(rules, 0)
		require.NoError(t, err)

		services = append(services, server.ServiceConfig{
			Name:        s.name,
			Source:      s.source,
			Destination: s.destination,
			Engine:      rm,
		})
	}

	log := plog.Logger{
		Level:      plog.DebugLevel,
		TimeField:  "time",
		TimeFormat: "23:59:59",
		Writer:     &plog.IOWriter{ioutil.Discard},
		// Writer: &plog.IOWriter{os.Stdout},
	}
	s := server.New(
		services,
		"Test Proxy",
		":8000",
		true,
		time.Second*10,
		log,
	)

	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		if err := s.Server.Serve(ln); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	c, err := ln.Dial()
	require.NoError(t, err)
	defer c.Close()

	for _, q := range []struct {
		query    string
		endpoint string
		method   string
		expect   int
	}{
		{
			query:    `{"query": "mutation X { a { a0(a0_0: [ 0 ]) } }", "operationName": "X"}`,
			endpoint: "/service_a",
			expect:   fasthttp.StatusOK,
		},
		{
			query:    `{"query": "query X { b { b0(b0_0: [ \"not a mutation\" ]) } }", "operationName": "X"}`,
			endpoint: "/service_b",
			expect:   fasthttp.StatusOK,
		},
		{
			query:    `{"query": "mutation X { a { a0(a0_0: [ -1 ]) } }", "operationName": "X"}`,
			endpoint: "/service_a",
			expect:   fasthttp.StatusForbidden,
		},
		{
			query:    `{"query": "query X { b { b0(b0_0: [ \"not a mutation\" ]) } }", "operationName": "X"}`,
			endpoint: "/service_c",
			expect:   fasthttp.StatusNotFound,
		},
		{
			query:    `{"query": "mutation X { a { a0(a0_0: [ 0 ]) } }", "operationName": "X"}`,
			endpoint: "/service_a",
			method:   "PUT",
			expect:   fasthttp.StatusMethodNotAllowed,
		},
		{
			query:    `{"query": "query X { b { b0(b0_0: [ \"not a mutation\" ]) } }"}`,
			endpoint: "/service_b",
			expect:   fasthttp.StatusBadRequest,
		},
		{
			query:    `{"query": "mutation X { a { a0(a0_0: [ 0 ]) }", "operationName": "X"}`,
			endpoint: "/service_a",
			expect:   fasthttp.StatusBadRequest,
		},
	} {
		method := "POST"
		if q.method != "" {
			method = q.method
		}
		request := []byte(
			fmt.Sprintf(
				"%s %s HTTP/1.1\r\nHost: localhost:8000\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s",
				method,
				q.endpoint,
				len(q.query),
				q.query,
			),
		)
		_, err = c.Write(request)
		require.NoError(t, err)

		br := bufio.NewReader(c)
		var resp fasthttp.Response
		err = resp.Read(br)
		require.NoError(t, err)
		require.Equal(t, q.expect, resp.StatusCode())
	}
}
