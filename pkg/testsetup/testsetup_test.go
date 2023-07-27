package testsetup_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

func TestStarwars(t *testing.T) {
	s, ok := testsetup.ByName(testsetup.SetupNameStarwars)
	require.True(t, ok)
	checkSetup(t, s)
}

func TestInputsSchema(t *testing.T) {
	s, ok := testsetup.ByName(testsetup.SetupNameInputsSchema)
	require.True(t, ok)
	checkSetup(t, s)
}

func checkSetup(t *testing.T, s testsetup.Setup) {
	require.NotZero(t, s.Config)
	require.NotZero(t, s.Name)
}
