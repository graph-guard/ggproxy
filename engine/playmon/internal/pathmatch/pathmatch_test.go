package pathmatch_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/pathmatch"
	"github.com/graph-guard/gqt/v4"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := prepareTestSetup(t, tt.conf)
			var actualMatches []string
			paths := make([][]byte, len(tt.paths))
			for i := range tt.paths {
				paths[i] = []byte(tt.paths[i])
			}
			m.Match(paths, func(tm *config.Template) (stop bool) {
				actualMatches = append(actualMatches, tm.ID)
				return false
			})
			require.Equal(t, tt.expectIDs, actualMatches)
		})
	}
}

var tests = []struct {
	name      string
	conf      *config.Service
	paths     []string
	expectIDs []string
}{
	{
		name: "no paths",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{{
				ID: "A", Source: []byte(`query { foo }`),
			}},
		},
		paths:     nil,
		expectIDs: nil,
	},
	{
		name: "unknown path",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{{
				ID: "A", Source: []byte(`query { foo }`),
			}},
		},
		paths:     []string{"Q.bar"},
		expectIDs: nil,
	},
	{
		name: "1_of_1",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{{
				ID: "A", Source: []byte(`query { foo }`),
			}},
		},
		paths:     []string{"Q.foo"},
		expectIDs: []string{"A"},
	},
	{
		name: "first_of_2",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo }`)},
				{ID: "B", Source: []byte(`query { bar }`)},
			},
		},
		paths:     []string{"Q.foo"},
		expectIDs: []string{"A"},
	},
	{
		name: "second_of_2",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo }`)},
				{ID: "B", Source: []byte(`query { bar }`)},
			},
		},
		paths:     []string{"Q.bar"},
		expectIDs: []string{"B"},
	},
	{
		name: "not_both",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo }`)},
				{ID: "B", Source: []byte(`query { bar }`)},
			},
		},
		paths:     []string{"Q.foo", "Q.bar"},
		expectIDs: nil,
	},
	{
		name: "1_of_3",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo bazz }`)},
				{ID: "B", Source: []byte(`query { bar bazz }`)},
			},
		},
		paths:     []string{"Q.bazz", "Q.bar"},
		expectIDs: []string{"B"},
	},
	{
		name: "3_of_3",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo bazz }`)},
				{ID: "B", Source: []byte(`query { bar bazz }`)},
				{ID: "C", Source: []byte(`query { bazz }`)},
			},
		},
		paths:     []string{"Q.bazz"},
		expectIDs: []string{"A", "B", "C"},
	},
	// {
	// 	name: "2_of_20",
	// 	conf: &config.Service{
	// 		TemplatesEnabled: []*config.Template{
	// 			{ID: "Q1", Source: []byte(`query { foo bazz }`)},
	// 			{ID: "Q2", Source: []byte(`query { bar bazz }`)},
	// 			{ID: "Q3", Source: []byte(`query { bazz { fraz } maz }`)},
	// 			{ID: "Q4", Source: []byte(`query { bazz { fraz } }`)},
	// 		},
	// 	},
	// 	paths:     []string{"Q.bazz"},
	// 	expectIDs: []string{"A", "B", "C"},
	// },
	{
		name: "starwars",
		conf: func() *config.Service {
			s, err := config.New("./test_setups/starwars/config.yml")
			if err != nil {
				panic(err)
			}
			return s.ServicesEnabled[0]
		}(),
		paths: []string{
			"Q.hero|episode,.id",
			"Q.hero|episode,.name",
			"Q.hero|episode,.appearsIn",
			"Q.hero|episode,.friends.id",
			"Q.hero|episode,.friends.name",
			"Q.hero|episode,.friends.appearsIn",
			"Q.hero|episode,.friends.friends.id",
			"Q.hero|episode,.friends.friends.name",
			"Q.hero|episode,.friends.friends.appearsIn",
			"Q.hero|episode,.friendsConnection|after,first,.totalCount",
			"Q.hero|episode,.friendsConnection|after,first,.friends.id",
			"Q.hero|episode,.friendsConnection|after,first,.friends.name",
			"Q.hero|episode,.friendsConnection|after,first,.friends.appearsIn",
		},
		expectIDs: []string{"c"},
		/* query {
		    hero(episode: EMPIRE || JEDI) {
		        id
		        name
		        appearsIn
		        friends {
		            id
		            name
		            appearsIn
		            friends {
		                id
		                name
		                appearsIn
		            }
		        }
		        friendsConnection(first: >= 0, after: len > 0) {
		            totalCount
		            friends {
		                id
		                name
		                appearsIn
		            }
		        }
		    }
		} */
	},
}

func prepareTestSetup(t testing.TB, c *config.Service) *pathmatch.Matcher {
	p, err := gqt.NewParser(nil)
	if err != nil {
		t.Fatalf("initializing gqt parser: %v", err)
	}
	for _, tmpl := range c.TemplatesEnabled {
		opr, _, errs := p.Parse(tmpl.Source)
		if errs != nil {
			t.Fatalf("parsing template: %v", errs)
		}
		tmpl.GQTTemplate, tmpl.Enabled = opr, true
	}
	m := pathmatch.New(c)
	return m
}
