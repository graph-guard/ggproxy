package graph

import (
	"time"

	"github.com/graph-guard/ggproxy/api/graph/model"
	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/gqlreduce"
)

//go:generate go run github.com/99designs/gqlgen

type Resolver struct {
	Start    time.Time
	Version  string
	Conf     *config.Config
	Reducer  *gqlreduce.Reducer
	Services map[string]*model.Service
}
