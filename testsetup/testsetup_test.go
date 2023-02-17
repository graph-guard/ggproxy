package testsetup_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/testsetup"
	"github.com/stretchr/testify/require"
)

func TestStarwars(t *testing.T) { checkSetup(t, testsetup.Starwars()) }
func TestTest1(t *testing.T)    { checkSetup(t, testsetup.Test1()) }
func TestTest2(t *testing.T)    { checkSetup(t, testsetup.Test2()) }

func checkSetup(t *testing.T, s testsetup.Setup) {
	require.NotZero(t, s.Config)
	require.NotZero(t, s.Name)
	require.NotZero(t, s.Tests)
}
