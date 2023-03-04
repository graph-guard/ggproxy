package model

import (
	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/engine/playmon"
	"github.com/graph-guard/ggproxy/statistics"
)

type Service struct {
	Engine    *playmon.Engine
	Templates map[*config.Template]*Template
	Stats     *statistics.ServiceSync

	ID                string      `json:"id"`
	TemplatesEnabled  []*Template `json:"templatesEnabled"`
	TemplatesDisabled []*Template `json:"templatesDisabled"`
	ProxyURL          string      `json:"proxyURL"`
	ForwardURL        string      `json:"forwardURL"`
	ForwardReduced    bool        `json:"forwardReduced"`
	Enabled           bool        `json:"enabled"`
}

type Template struct {
	Service *Service
	Stats   *statistics.TemplateSync

	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Source  string   `json:"source"`
	Enabled bool     `json:"enabled"`
}
