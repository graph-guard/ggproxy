package pathmatch_test

import (
	"embed"
	"testing"

	"github.com/graph-guard/ggproxy/pkg/config"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/pathmatch"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/pathscan"
	"github.com/graph-guard/gqt/v4"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := prepareTestSetup(t, tt.conf)
			for _, tt := range tt.tests {
				t.Run(tt.name, func(t *testing.T) {
					var actualMatches []string
					paths := make([]uint64, len(tt.paths))
					for i := range tt.paths {
						paths[i] = pathscan.Hash(tt.paths[i])
					}
					m.Match(paths, func(tm *config.Template) (stop bool) {
						actualMatches = append(actualMatches, tm.ID)
						return false
					})
					require.Equal(t, tt.expectIDs, actualMatches)
				})
			}
		})
	}
}

//go:embed test_setups
var embeddedTestSetups embed.FS

type test struct {
	name      string
	paths     []string
	expectIDs []string
}

var tests = []struct {
	name  string
	conf  *config.Service
	tests []test
}{
	{
		name: "no paths",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{{
				ID: "A", Source: []byte(`query { foo }`),
			}},
		},
		tests: []test{{
			paths:     nil,
			expectIDs: nil,
		}},
	},
	{
		name: "unknown path",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{{
				ID: "A", Source: []byte(`query { foo }`),
			}},
		},
		tests: []test{{
			paths:     []string{"Q.bar"},
			expectIDs: nil,
		}},
	},
	{
		name: "1_of_1",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{{
				ID: "A", Source: []byte(`query { foo }`),
			}},
		},
		tests: []test{{
			paths:     []string{"Q.foo"},
			expectIDs: []string{"A"},
		}},
	},
	{
		name: "first_of_2",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo }`)},
				{ID: "B", Source: []byte(`query { bar }`)},
			},
		},
		tests: []test{{
			paths:     []string{"Q.foo"},
			expectIDs: []string{"A"},
		}},
	},
	{
		name: "second_of_2",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo }`)},
				{ID: "B", Source: []byte(`query { bar }`)},
			},
		},
		tests: []test{{
			paths:     []string{"Q.bar"},
			expectIDs: []string{"B"},
		}},
	},
	{
		name: "not_both",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo }`)},
				{ID: "B", Source: []byte(`query { bar }`)},
			},
		},
		tests: []test{{
			paths:     []string{"Q.foo", "Q.bar"},
			expectIDs: nil,
		}},
	},
	{
		name: "1_of_3",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query { foo bazz }`)},
				{ID: "B", Source: []byte(`query { bar bazz }`)},
			},
		},
		tests: []test{{
			paths:     []string{"Q.bazz", "Q.bar"},
			expectIDs: []string{"B"},
		}},
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
		tests: []test{{
			paths:     []string{"Q.bazz"},
			expectIDs: []string{"A", "B", "C"},
		}},
	},
	{
		name: "starwars",
		conf: func() *config.Service {
			s, err := config.Read(
				embeddedTestSetups,
				"test_setups/starwars/",
				"test_setups/starwars/config.yml",
			)
			if err != nil {
				panic(err)
			}
			return s.ServicesEnabled[0]
		}(),
		/* query { hero(episode: EMPIRE || JEDI) {
			id, name, appearsIn
			friends {
				id, name, appearsIn
				friends { id, name, appearsIn }
			}
			friendsConnection(first: >= 0, after: len > 0) {
				totalCount
				friends { id, name, appearsIn }
			}
		} } */
		tests: []test{{
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
		}},
	},
	{
		name: "max1_1_template",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query {
					max 1 { a b c }
				}
				`)},
			},
		},
		tests: []test{{
			name:      "field_a",
			paths:     []string{"Q.a"},
			expectIDs: []string{"A"},
		}, {
			name:      "field_b",
			paths:     []string{"Q.b"},
			expectIDs: []string{"A"},
		}, {
			name:      "field_c",
			paths:     []string{"Q.c"},
			expectIDs: []string{"A"},
		}, {
			name:      "violate_c_a",
			paths:     []string{"Q.c", "Q.a"},
			expectIDs: nil,
		}, {
			name:      "violate_b_a",
			paths:     []string{"Q.b", "Q.a"},
			expectIDs: nil,
		}, {
			name:      "violate_a_c",
			paths:     []string{"Q.a", "Q.c"},
			expectIDs: nil,
		}, {
			name:      "violate_a_b_c",
			paths:     []string{"Q.a", "Q.b", "Q.c"},
			expectIDs: nil,
		}},
	},
	{
		name: "max_2_multitemplates",
		conf: &config.Service{
			TemplatesEnabled: []*config.Template{
				{ID: "A", Source: []byte(`query {
					max 1 {
						a {
							max 1 { a0 a1 }
						}
						b
					}
				}
				`)},
				{ID: "B", Source: []byte(`query {
					max 2 {
						b
						a {
							max 2 { a0 a1 a2 }
						}
						c
					}
				}
				`)},
			},
		},
		tests: []test{{
			name:      "field_b",
			paths:     []string{"Q.b"},
			expectIDs: []string{"A", "B"},
		}, {
			name:      "field_c",
			paths:     []string{"Q.c"},
			expectIDs: []string{"B"},
		}, {
			name:      "inexistent_path",
			paths:     []string{"Q.d"},
			expectIDs: nil,
		}, {
			name: "violate_A_max",
			paths: []string{
				"Q.a.a0",
				"Q.b",
			},
			expectIDs: []string{"B"},
		}, {
			name: "violate_A",
			paths: []string{
				"Q.a.a1",
				"Q.a.a0",
			},
			expectIDs: []string{"B"},
		}, {
			name: "violate_A_all_levels",
			paths: []string{
				"Q.b",
				"Q.a.a1",
				"Q.a.a0",
			},
			expectIDs: []string{"B"},
		}, {
			name: "a1",
			paths: []string{
				"Q.a.a1",
			},
			expectIDs: []string{"A", "B"},
		}, {
			name: "a2",
			paths: []string{
				"Q.a.a2",
			},
			expectIDs: []string{"B"},
		}},
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
