package hmap_test

import (
	"errors"
	"testing"

	"github.com/graph-guard/gguard-proxy/engines/hmap"
	"github.com/graph-guard/gqt"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	for ti, td := range []struct {
		index     int
		query     string
		varsJSON  string
		templates []string
		expect    int
	}{
		/************************************************
			NO MATCH (due to mismatched structure)
		************************************************/
		{index: 0, // No templates
			query:     "{x}",
			templates: []string{},
			expect:    -1,
		},
		{index: 1,
			query:     "{x}",
			templates: []string{`query {y}`},
			expect:    -1,
		},
		{index: 2,
			query: "{y}",
			templates: []string{
				`query {b}`,
				`query {c}`,
				`query {a}`,
				`query {x}`,
			},
			expect: -1,
		},
		{index: 3, // No submatches allowed
			query: `{
				a {
					aa {
						aaa
						aab
					}
					ab
					ac
				}
			}`,
			templates: []string{
				`query {
					a {
						aa {
							aaa
							aab
						}
						ab
						ac
					}
					b
					c
					d {
						da {
							daa
							dab
						}
					}
				}`,
			},
			expect: -1,
		},

		/************************************************
			NO MATCH (due to mismatched inputs)
		************************************************/
		{index: 4,
			query:     `{x(a:"actual")}`,
			templates: []string{`query {x(a: val = "expected")}`},
			expect:    -1,
		},
		{index: 5, // Negated equality
			query:     `{x(a:"text")}`,
			templates: []string{`query {x(a: val != "text")}`},
			expect:    -1,
		},
		{index: 6, // Value equals array
			query: `{x(a:["second", "first"])}`,
			templates: []string{
				`query {x(a: val = [ val = "first", val = "second" ])}`,
			},
			expect: -1,
		},
		{index: 7, // Negated array equality
			query: `{x(a:["first", "second"])}`,
			templates: []string{
				`query {x(a: val != [ val = "first", val = "second" ])}`,
			},
			expect: -1,
		},
		{index: 8,
			query: `{x(a:10)}`,
			templates: []string{
				`query {x(a: val > 10)}`,
			},
			expect: -1,
		},
		{index: 9,
			query: `{x(a:10)}`,
			templates: []string{
				`query {x(a: val < 10)}`,
			},
			expect: -1,
		},
		{index: 10,
			query: `{x(a:9)}`,
			templates: []string{
				`query {x(a: val >= 10)}`,
			},
			expect: -1,
		},
		{index: 11,
			query: `{x(a:11)}`,
			templates: []string{
				`query {x(a: val <= 10)}`,
			},
			expect: -1,
		},

		/************************************************
			MATCH
		************************************************/
		{index: 12,
			query:     "{x}",
			templates: []string{`query {x}`},
			expect:    0,
		},
		{index: 13,
			query: "{x}",
			templates: []string{
				`query {b}`,
				`query {c}`,
				`query {a}`,
				`query {x}`,
			},
			expect: 3,
		},
		{index: 14, // Exact structural match
			query: `{
				a {
					aa {
						aaa
						aab
					}
					ab
					ac
				}
				b
				c
				d {
					da {
						daa
						dab
					}
				}
			}`,
			templates: []string{
				`query {
					a {
						aa {
							aaa
							aab
						}
						ab
						ac
					}
					b
					c
					d {
						da {
							daa
							dab
						}
					}
				}`,
			},
			expect: 0,
		},
		{index: 15, // Value equals null
			query:     `{x(a:null)}`,
			templates: []string{`query {x(a: val = null)}`},
			expect:    0,
		},
		{index: 16, // Value equals bool
			query:     `{x(a:true)}`,
			templates: []string{`query {x(a: val = true)}`},
			expect:    0,
		},
		{index: 17, // Value equals string
			query:     `{x(a:"okay")}`,
			templates: []string{`query {x(a: val = "okay")}`},
			expect:    0,
		},
		{index: 18, // Value equals empty array
			query:     `{x(a:[])}`,
			templates: []string{`query {x(a: val = [])}`},
			expect:    0,
		},
		{index: 19, // Value equals array
			query: `{x(a:["first", "second"])}`,
			templates: []string{
				`query {x(a: val = [ val = "first", val = "second" ])}`,
			},
			expect: 0,
		},
		{index: 20,
			query: `{x(a:2 b:2 c:1 d:2)}`,
			templates: []string{
				`query {x(
					a: val > 1
					b: val >= 2
					c: val < 2
					d: val <= 2
				)}`,
			},
			expect: 0,
		},
		{index: 21, // Complex
			query: `query X {
				b {
					x {
						p(
							j: [ {f: -273} {f: -273} ]
							k: [ -13, -88 ]
						)
						n( i: "alive" ) { j }
					}
				}
				a {
					z(
						m: {
							i: [
								[
									{ g: 69 h: [ 0, -1 ] }
									{ g: 70 h: [ 0, 1 ] }
								]
								[]
								[ { g: 71 h: [ 2, 3 ] } ]
							]
							j: { h: "yo" }
						}
						n: -1
					) {
						y( n: [ "foo", "bar" ] p: { k: 0 j: "lol" } )
						x
					}
				}
				c(
					y: [
						[
							[ { n: "too" } { n: "deep" } ]
							[ { n: "steep" } ]
							[ { n: "yet" } { n: "another" } { n: "array" } ]
						]
						[ [] ]
						[ [] [ { n: "this" } { n: "is" } { n: "it" } ] ]
					]
				)
			}`,
			templates: []string{
				`query {
					b {
						x {
							p(
								j: val = [
									val = {f: val = -273}
									val = {f: val = -273}
								]
								k: val = [
									val = -13
									val = -88
								]
							)
							n( i: val = "alive" ) { j }
						}
					}
					a {
						z(
							m: val = {
								i: val = [
									val = [
										val = {
											g: val = 69
											h: val = [
												val = 0
												val = -1
											]
										}
										val = {
											g: val = 70
											h: val = [
												val = 0
												val = 1
											]
										}
									]
									val = []
									val = [
										val = {
											g: val = 71
											h: val = [
												val = 2
												val = 3
											]
										}
									]
								]
								j: val = {
									h: val = "yo"
								}
							}
							n: val = -1
						) {
							y(
								n: val = [
									val = "foo"
									val = "bar"
								]
								p: val = {
									k: val = 0
									j: val = "lol"
								}
							)
							x
						}
					}
					c(
						y: val = [
							val = [
								val = [
									val = { n: val = "too" }
									val = { n: val = "deep" }
								]
								val = [ val = { n: val = "steep" } ]
								val = [
									val = { n: val = "yet" }
									val = { n: val = "another" }
									val = { n: val = "array" }
								]
							]
							val = [ val = [] ]
							val = [
								val = []
								val = [
									val = { n: val = "this" }
									val = { n: val = "is" }
									val = { n: val = "it" }
								]
							]
						]
					)
				}`,
			},
			expect: 0,
		},
	} {
		t.Run("", func(t *testing.T) {
			require.Equal(t, ti, td.index)
			m := hmap.New(1024)
			for i, s := range td.templates {
				d, err := gqt.Parse([]byte(s))
				if err.IsErr() {
					t.Fatalf("parsing template %d: %s", i, err)
				}
				m.AddTemplate(d)
			}
			a, err := m.Match([]byte(td.query), []byte(td.varsJSON))
			require.NoError(t, err)
			require.Equal(t, td.expect, a)
		})
	}
}

func TestMatchExceedLimit(t *testing.T) {
	limit := 128
	m := hmap.New(limit)
	input := []byte(`query UnacceptableQuery {
		this_query_exceeds {
			the_configured_128_bytes_limit {
				and_should_make_match_return_an_error
			}
		}
	}`)
	require.GreaterOrEqual(t, len(input), limit)
	index, err := m.Match(input, nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, hmap.ErrInputExceedsLimit))
	require.Equal(t, -1, index)
}
