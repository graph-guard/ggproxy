package model

import "github.com/graph-guard/gguard-proxy/engines/rmap"

type Service struct {
	Matcher       *rmap.RulesMap
	TemplatesByID map[string]*Template

	ID                string             `json:"id"`
	TemplatesEnabled  []*Template        `json:"templatesEnabled"`
	TemplatesDisabled []*Template        `json:"templatesDisabled"`
	IngressURL        string             `json:"ingressURL"`
	ForwardURL        string             `json:"forwardURL"`
	ForwardReduced    bool               `json:"forwardReduced"`
	Enabled           bool               `json:"enabled"`
	Statistics        *ServiceStatistics `json:"statistics"`
}

type Template struct {
	Service *Service

	ID         string              `json:"id"`
	Tags       []string            `json:"tags"`
	Source     string              `json:"source"`
	Statistics *TemplateStatistics `json:"statistics"`
	Enabled    bool                `json:"enabled"`
}
