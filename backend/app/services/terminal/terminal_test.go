package terminal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseUint(t *testing.T) {
	require.Equal(t, uint(120), parseUint("120"))
	require.Equal(t, uint(0), parseUint(""))
	require.Equal(t, uint(0), parseUint("12a"))
	require.Equal(t, uint(0), parseUint("-3"))
}

func TestClampSize(t *testing.T) {
	require.Equal(t, uint(80), clampSize(0, 80, 20, 500))
	require.Equal(t, uint(80), clampSize(5, 80, 20, 500))
	require.Equal(t, uint(500), clampSize(900, 80, 20, 500))
	require.Equal(t, uint(120), clampSize(120, 80, 20, 500))
	require.Equal(t, uint(24), clampSize(0, 24, 5, 300))
}
