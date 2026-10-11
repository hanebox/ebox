package envvars

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDotEnv(t *testing.T) {
	pairs, err := ParseDotEnv(`# comment

APP_ENV=production
export DATABASE_URL=postgres://user:pass@db:5432/app
QUOTED="hello world"
SINGLE='single quoted'
ESCAPED="line1\nline2"
EQUALS=a=b=c
EMPTY=
`)
	require.NoError(t, err)

	expected := map[string]string{
		"APP_ENV":      "production",
		"DATABASE_URL": "postgres://user:pass@db:5432/app",
		"QUOTED":       "hello world",
		"SINGLE":       "single quoted",
		"ESCAPED":      "line1\nline2",
		"EQUALS":       "a=b=c",
		"EMPTY":        "",
	}

	require.Len(t, pairs, len(expected))
	for _, pair := range pairs {
		require.Equal(t, expected[pair.Key], pair.Value, "key %s", pair.Key)
	}
}

func TestParseDotEnvRejectsInvalidLines(t *testing.T) {
	_, err := ParseDotEnv("NOT A PAIR")
	require.Error(t, err)

	_, err = ParseDotEnv("1INVALID=value")
	require.Error(t, err)
}

func TestQuoteValueRoundTrip(t *testing.T) {
	values := []string{
		"simple",
		"with spaces",
		`with "quotes"`,
		"with\nnewline",
		"with#hash",
		"",
		`back\slash`,
	}

	for _, value := range values {
		content := "KEY=" + quoteValue(value) + "\n"
		pairs, err := ParseDotEnv(content)
		require.NoError(t, err)
		require.Len(t, pairs, 1)
		require.Equal(t, value, pairs[0].Value, "value %q", value)
	}
}

func TestWriteEnvFileUsesStrictPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apps", "abc", "env")

	require.NoError(t, Default().WriteEnvFile(path, map[string]string{
		"B_VAR": "second",
		"A_VAR": "first",
	}))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "A_VAR=first\nB_VAR=second\n", string(content))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
