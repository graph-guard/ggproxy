package model

import (
	"github.com/graph-guard/ggproxy/engines/rmap"
	"github.com/graph-guard/ggproxy/statistics"
)

type Service struct {
	Matcher       *rmap.RulesMap
	TemplatesByID map[string]*Template
	Stats         *statistics.ServiceSync

	ID                string      `json:"id"`
	TemplatesEnabled  []*Template `json:"templatesEnabled"`
	TemplatesDisabled []*Template `json:"templatesDisabled"`
	IngressURL        string      `json:"ingressURL"`
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
