package testsetup_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

func TestStarwars(t *testing.T)     { checkSetup(t, testsetup.Starwars()) }
func TestTest1(t *testing.T)        { checkSetup(t, testsetup.Test1()) }
func TestInputsSchema(t *testing.T) { checkSetup(t, testsetup.InputsSchema()) }

func checkSetup(t *testing.T, s testsetup.Setup) {
	require.NotZero(t, s.Config)
	require.NotZero(t, s.Name)
	require.NotZero(t, s.Tests)
}
