package graph

import (
	"time"

	"github.com/graph-guard/ggproxy/api/graph/model"
	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/gqlparse"
	plog "github.com/phuslu/log"
)

//go:generate go run github.com/99designs/gqlgen

type Resolver struct {
	Start    time.Time
	Version  string
	Conf     *config.Config
	Parser   *gqlparse.Parser
	Services map[string]*model.Service
	Log      plog.Logger
}

// nsToF64 safely converts nanoseconds to float64
// with a max value of 9.00719925474e+15.
func nsToF64(nanoseconds int64) float64 {
	const max = 9.00719925474e+15
	if nanoseconds > max {
		nanoseconds = max
	}
	return float64(nanoseconds)
}
