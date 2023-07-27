package pathinfo_test

import (
	"testing"

	maxsets "github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/pathinfo"
	"github.com/graph-guard/gqt/v4"
	"github.com/stretchr/testify/require"
)

func TestInfo(t *testing.T) {
	opr, _, errs := gqt.Parse([]byte(`query {
		maz {
			muzz
		}
		max 1 {
			foo(x:>1,y:!="ok") {
				max 2 {
					bar {
						... on Baz {
							baz
						}
					}
					bar2
					bar3
				}
			}
			foo2
		}
	}`))
	require.Nil(t, errs)
	{
		depth, parent := maxsets.Info(nil)
		require.Equal(t, 0, depth)
		require.Nil(t, parent)
	}
	{
		depth, parent := maxsets.Info(opr)
		require.Equal(t, 0, depth)
		require.Nil(t, parent)
	}
	{
		maz := opr.Selections[0].(*gqt.SelectionField)
		require.Equal(t, "maz", maz.Name.Name)

		iDepth, iParent := maxsets.Info(maz)
		require.Equal(t, 0, iDepth)
		require.Nil(t, iParent)
	}
	{
		maz := opr.Selections[0].(*gqt.SelectionField)
		muzz := maz.Selections[0].(*gqt.SelectionField)
		require.Equal(t, "muzz", muzz.Name.Name)

		iDepth, iParent := maxsets.Info(muzz)
		require.Equal(t, 0, iDepth)
		require.Nil(t, iParent)
	}
	{
		firstMax := opr.Selections[1].(*gqt.SelectionMax)
		foo := firstMax.Options.Selections[0].(*gqt.SelectionField)
		require.Equal(t, "foo", foo.Name.Name)

		iDepth, iParent := maxsets.Info(foo)
		require.Equal(t, 1, iDepth)
		require.Equal(t, 1, iParent.Limit)
	}
	{
		firstMax := opr.Selections[1].(*gqt.SelectionMax)
		foo := firstMax.Options.Selections[0].(*gqt.SelectionField)
		secondMax := foo.Selections[0].(*gqt.SelectionMax)
		bar := secondMax.Options.Selections[0].(*gqt.SelectionField)
		require.Equal(t, "bar", bar.Name.Name)

		iDepth, iParent := maxsets.Info(bar)
		require.Equal(t, 2, iDepth)
		require.Equal(t, 2, iParent.Limit)
	}
	{
		firstMax := opr.Selections[1].(*gqt.SelectionMax)
		foo := firstMax.Options.Selections[0].(*gqt.SelectionField)
		secondMax := foo.Selections[0].(*gqt.SelectionMax)
		bar := secondMax.Options.Selections[0].(*gqt.SelectionField)
		onBaz := bar.Selections[0].(*gqt.SelectionInlineFrag)
		baz := onBaz.Selections[0].(*gqt.SelectionField)
		require.Equal(t, "baz", baz.Name.Name)

		iDepth, iParent := maxsets.Info(baz)
		require.Equal(t, 2, iDepth)
		require.Equal(t, 2, iParent.Limit)
	}
}
