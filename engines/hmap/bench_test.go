package hmap_test

import (
	"testing"

	"github.com/graph-guard/gguard-proxy/engines/hmap"
	"github.com/stretchr/testify/require"

	"github.com/graph-guard/gqt"
)

var GI int

func TestBenchmarks(t *testing.T) {
	for _, td := range benchmarks {
		t.Run(td.name, func(t *testing.T) {
			require := require.New(t)
			m := hmap.New(1024)
			for i, s := range td.templates {
				d, err := gqt.Parse([]byte(s))
				require.False(err.IsErr(), "parsing template %d: %s", i, err)
				m.AddTemplate(d)
			}
			index, err := m.Match([]byte(td.query), []byte(td.varsJSON))
			require.NoError(err)
			require.Equal(td.expect, index)
		})
	}
}

func BenchmarkMatch(b *testing.B) {
	for _, td := range benchmarks {
		b.Run(td.name, func(b *testing.B) {
			m := hmap.New(1024)
			for i, s := range td.templates {
				d, err := gqt.Parse([]byte(s))
				if err.IsErr() {
					b.Fatalf("parsing template %d: %s", i, err)
				}
				m.AddTemplate(d)
			}
			query := []byte(td.query)
			varsJSON := []byte(td.varsJSON)
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				var err error
				if GI, err = m.Match(query, varsJSON); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

var benchmarks = []struct {
	name      string
	expect    int
	query     string
	varsJSON  string
	templates []string
}{
	{
		name:      "tiny",
		expect:    0,
		query:     "{x}",
		templates: []string{`query { x }`},
	},
	{
		name:   "big_noinput",
		expect: 0,
		query: `query X {
			node {
				subnode {
					subsubnodeA
					subsubnodeB
					subsubnodeC { sub4Node }
				}
			}
			nodeA {
				nodeA_subnode {
					nodeA_subnode_subnode
					nodeA_subnode_subnode_B
					nodeA_subnode_subnode_thirdsubnode
				}
			}
			x
			y
			anotherNode {
				yetAnotherNode {
					evenMoreNodes {
						weNeedToGoDeeper {
							evenDeeper {
								moreMoreMore {
									thisIsntEnough {
										almostThere {
											almost {
												thisShouldDo
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}`,
		templates: []string{
			`query {
				node {
					subnode {
						subsubnodeA
						subsubnodeB
						subsubnodeC { sub4Node }
					}
				}
				nodeA {
					nodeA_subnode {
						nodeA_subnode_subnode
						nodeA_subnode_subnode_B
						nodeA_subnode_subnode_thirdsubnode
					}
				}
				x
				y
				anotherNode {
					yetAnotherNode {
						evenMoreNodes {
							weNeedToGoDeeper {
								evenDeeper {
									moreMoreMore {
										thisIsntEnough {
											almostThere {
												almost {
													thisShouldDo
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}`,
		},
	},
	{
		name:   "complex",
		expect: 0,
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
	},
	{ /* Worst case scenario when everything matches
		except the last constraint */
		name:   "worst case",
		expect: 1,
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
								val != { n: val = "it" }
							]
						]
					]
				)
			}`,
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
	},
}
