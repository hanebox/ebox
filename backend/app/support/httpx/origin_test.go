package httpx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidOrigin(t *testing.T) {
	cases := []struct {
		value string
		valid bool
	}{
		{"http://localhost:5173", true},
		{"https://panel.example.com", true},
		{"https://panel.example.com/", true},
		{"http://10.0.0.5:8000", true},
		{"https://", false},
		{"ftp://panel.example.com", false},
		{"https://panel.example.com/path", false},
		{"https://panel.example.com?query=1", false},
		{"javascript:alert(1)", false},
		{"", false},
	}

	for _, testCase := range cases {
		require.Equal(t, testCase.valid, ValidOrigin(testCase.value), "validating %q", testCase.value)
	}
}

func TestNormalizeOrigin(t *testing.T) {
	require.Equal(t, "https://panel.example.com", NormalizeOrigin("  https://panel.example.com/  "))
	require.Equal(t, "http://localhost:5173", NormalizeOrigin("http://localhost:5173"))
	require.Equal(t, "", NormalizeOrigin("  "))
}
