package server_test

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/graph-guard/gguard-proxy/server"
	"github.com/graph-guard/gguard/engines/rmap"
	"github.com/graph-guard/gqt"
	plog "github.com/phuslu/log"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

//go:embed assets/benchassets
var assets embed.FS

func BenchmarkServer100(b *testing.B) {
	var templates []gqt.Doc
	var queries []struct {
		query         string
		operationName string
		variables     string
	}
	services := make([]server.ServiceConfig, 100)

	fs.WalkDir(assets, "assets/benchassets/templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			template, err := fs.ReadFile(assets, path)
			if err != nil {
				panic(err)
			}
			if rd, err := gqt.Parse([]byte(template)); !err.IsErr() {
				templates = append(templates, rd)
			} else {
				panic(err)
			}
		}
		return nil
	})

	fs.WalkDir(assets, "assets/benchassets/queries", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			query, err := fs.ReadFile(assets, path)
			if err != nil {
				panic(err)
			}
			data := make(map[string]string)
			err = yaml.Unmarshal(query, &data)
			if err != nil {
				panic(err)
			}
			queries = append(queries, struct {
				query         string
				operationName string
				variables     string
			}{
				query:         data["query"],
				operationName: data["operationName"],
				variables:     data["veriables"],
			})
		}
		return nil
	})

	endpoints := make([]string, 100)
	for i := 0; i < len(endpoints); i++ {
		rm, err := rmap.New(templates, 0)
		require.NoError(b, err)

		var se string
		idx := 0
		for idx >= 0 {
			se = "service_" + RandomString(4)
			idx = slices.Index(endpoints, se)
		}
		endpoints[i] = se
		services[i] = server.ServiceConfig{
			Name:        fmt.Sprintf("Service %s", se),
			Source:      fmt.Sprintf("/%s", se),
			Destination: fmt.Sprintf("localhost:8080/%s", se),
			Engine:      rm,
		}
	}

	requests := make([][]byte, 100)
	for i := 0; i < len(requests); i++ {
		q := queries[rand.Intn(len(queries))]
		query := fmt.Sprintf(`{"query": "%s", "operationName": "%s"}`, q.query, q.operationName)
		endpoint := fmt.Sprintf("/%s", endpoints[rand.Intn(len(endpoints))])
		requests[i] = []byte(
			fmt.Sprintf(
				"POST %s HTTP/1.1\r\nHost: localhost:8000\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s",
				endpoint,
				len(query),
				query,
			),
		)
	}

	log := plog.Logger{
		Level:      plog.DebugLevel,
		TimeField:  "time",
		TimeFormat: "23:59:59",
		Writer:     &plog.IOWriter{ioutil.Discard},
	}
	s := server.New(
		services,
		"Proxy Benchmark",
		":8000",
		true,
		time.Second*10,
		log,
	)

	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		if err := s.Server.Serve(ln); err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}()

	c, err := ln.Dial()
	require.NoError(b, err)
	defer c.Close()

	br := bufio.NewReader(c)
	var resp fasthttp.Response

	b.Run("BenchmarkServer100", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			_, err = c.Write(requests[rand.Intn(len(requests))])
			if err != nil {
				b.Fatal(err)
			}
			err = resp.Read(br)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func RandomString(n int) string {
	var letters = []rune("abcd")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
