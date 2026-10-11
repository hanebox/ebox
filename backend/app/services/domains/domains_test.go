package domains

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeHost(t *testing.T) {
	cases := []struct {
		input string
		want  string
		valid bool
	}{
		{"app.example.com", "app.example.com", true},
		{"App.Example.COM", "app.example.com", true},
		{"https://app.example.com", "app.example.com", true},
		{"http://app.example.com/some/path", "app.example.com", true},
		{"app.example.com:8080", "app.example.com", true},
		{"app.example.com.", "app.example.com", true},
		{"localhost", "localhost", true},
		{"  app.example.com  ", "app.example.com", true},
		{"", "", false},
		{"not a host", "", false},
		{"-leading.example.com", "", false},
		{"trailing-.example.com", "", false},
		{"app_example.com", "", false},
	}

	for _, testCase := range cases {
		got, err := NormalizeHost(testCase.input)
		if !testCase.valid {
			require.Error(t, err, "input %q", testCase.input)
			continue
		}

		require.NoError(t, err, "input %q", testCase.input)
		require.Equal(t, testCase.want, got, "input %q", testCase.input)
	}
}
