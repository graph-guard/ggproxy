package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"bytes"
	"context"
	"time"

	"github.com/graph-guard/ggproxy/api/graph/generated"
	"github.com/graph-guard/ggproxy/api/graph/model"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/tokenwriter"
)

// Uptime is the resolver for the uptime field.
func (r *queryResolver) Uptime(ctx context.Context) (int, error) {
	d := time.Since(r.Resolver.Start)
	return int(d.Seconds()), nil
}

// Version is the resolver for the version field.
func (r *queryResolver) Version(ctx context.Context) (string, error) {
	return r.Resolver.Version, nil
}

// Service is the resolver for the service field.
func (r *queryResolver) Service(ctx context.Context, id string) (*model.Service, error) {
	s, ok := r.Resolver.Services[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

// Services is the resolver for the services field.
func (r *queryResolver) Services(ctx context.Context) ([]*model.Service, error) {
	l := make([]*model.Service, len(r.Resolver.Services))
	i := 0
	for _, s := range r.Resolver.Services {
		l[i] = s
		i++
	}
	return l, nil
}

// MatchAll is the resolver for the matchAll field.
func (r *serviceResolver) MatchAll(ctx context.Context, obj *model.Service, query string, operationName *string, variablesJSON *string) (*model.MatchResult, error) {
	// Declare here instead of using named return variables
	// to avoid code generation overriding them.
	m := new(model.MatchResult)
	var err error

	oprName := []byte(nil)
	if operationName != nil {
		oprName = []byte(*operationName)
	}

	varsJSON := []byte(nil)
	if variablesJSON != nil {
		varsJSON = []byte(*variablesJSON)
	}

	startParsing := time.Now()
	r.Resolver.Parser.Parse(
		[]byte(query), oprName, varsJSON,
		func(
			varVals [][]gqlparse.Token,
			operation []gqlparse.Token,
			selectionSet []gqlparse.Token,
		) {
			m.TimeParsingNs = nsToF64(time.Since(startParsing).Nanoseconds())
			startMatching := time.Now()
			obj.Matcher.MatchAll(
				varVals,
				operation[0].ID,
				selectionSet,
				func(id string) {
					t := obj.TemplatesByID[id]
					m.Templates = append(m.Templates, t)
				},
			)
			m.TimeMatchingNs = nsToF64(time.Since(startMatching).Nanoseconds())
			var forwarded bytes.Buffer
			if err = tokenwriter.Write(&forwarded, operation); err != nil {
				r.Log.Error().
					Err(err).
					Msg("writing parsed")
				return
			}
			forwardStr := forwarded.String()
			m.Forwarded = &forwardStr
		},
		func(errParser error) {
			m.TimeParsingNs = nsToF64(time.Since(startParsing).Nanoseconds())
			err = errParser
		},
	)
	return m, err
}

// Match is the resolver for the match field.
func (r *serviceResolver) Match(ctx context.Context, obj *model.Service, query string, operationName *string, variablesJSON *string) (*model.MatchResult, error) {
	// Declare here instead of using named return variables
	// to avoid code generation overriding them.
	m := new(model.MatchResult)
	var err error

	oprName := []byte(nil)
	if operationName != nil {
		oprName = []byte(*operationName)
	}

	varsJSON := []byte(nil)
	if variablesJSON != nil {
		varsJSON = []byte(*variablesJSON)
	}

	startParsing := time.Now()
	r.Resolver.Parser.Parse(
		[]byte(query), oprName, varsJSON,
		func(
			varVals [][]gqlparse.Token,
			operation []gqlparse.Token,
			selectionSet []gqlparse.Token,
		) {
			m.TimeParsingNs = nsToF64(time.Since(startParsing).Nanoseconds())
			startMatching := time.Now()
			if id := obj.Matcher.Match(varVals, operation[0].ID, selectionSet); id != "" {
				m.Templates = []*model.Template{obj.TemplatesByID[id]}
			}
			m.TimeMatchingNs = nsToF64(time.Since(startMatching).Nanoseconds())
			var forwarded bytes.Buffer
			if err = tokenwriter.Write(&forwarded, operation); err != nil {
				r.Log.Error().
					Err(err).
					Msg("writing parsed")
				return
			}
			forwardStr := forwarded.String()
			m.Forwarded = &forwardStr
		},
		func(errParser error) {
			m.TimeParsingNs = nsToF64(time.Since(startParsing).Nanoseconds())
			err = errParser
		},
	)
	return m, err
}

// Statistics is the resolver for the statistics field.
func (r *serviceResolver) Statistics(ctx context.Context, obj *model.Service) (*model.ServiceStatistics, error) {
	return &model.ServiceStatistics{
		BlockedRequests:       int(obj.Stats.GetBlockedRequests()),
		ForwardedRequests:     int(obj.Stats.GetForwardedRequests()),
		ReceivedBytes:         int(obj.Stats.GetReceivedBytes()),
		SentBytes:             int(obj.Stats.GetSentBytes()),
		HighestProcessingTime: int(obj.Stats.GetHighestProcessingTime()),
		AverageProcessingTime: int(obj.Stats.GetAverageProcessingTime()),
		HighestResponseTime:   int(obj.Stats.GetHighestResponseTime()),
		AverageResponseTime:   int(obj.Stats.GetAverageResponseTime()),
	}, nil
}

// Statistics is the resolver for the statistics field.
func (r *templateResolver) Statistics(ctx context.Context, obj *model.Template) (*model.TemplateStatistics, error) {
	return &model.TemplateStatistics{
		Matches:               int(obj.Stats.GetMatches()),
		HighestProcessingTime: int(obj.Stats.GetHighestProcessingTime()),
		AverageProcessingTime: int(obj.Stats.GetAverageProcessingTime()),
		HighestResponseTime:   int(obj.Stats.GetHighestResponseTime()),
		AverageResponseTime:   int(obj.Stats.GetAverageResponseTime()),
	}, nil
}

// Service is the resolver for the service field.
func (r *templateResolver) Service(ctx context.Context, obj *model.Template) (*model.Service, error) {
	return obj.Service, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Service returns generated.ServiceResolver implementation.
func (r *Resolver) Service() generated.ServiceResolver { return &serviceResolver{r} }

// Template returns generated.TemplateResolver implementation.
func (r *Resolver) Template() generated.TemplateResolver { return &templateResolver{r} }

type queryResolver struct{ *Resolver }
type serviceResolver struct{ *Resolver }
type templateResolver struct{ *Resolver }
