package backups

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"ebox/app/models"
)

func TestParseSchedule(t *testing.T) {
	valid := map[string]string{
		"0 3 * * *":    "0 3 * * *",
		"*/15 * * * *": "*/15 * * * *",
		"@daily":       "@daily",
		" 30 2 * * 1 ": "30 2 * * 1",
		"":             "0 3 * * *",
	}

	for input, expected := range valid {
		parsed, err := ParseSchedule(input)
		require.NoError(t, err, input)
		require.Equal(t, expected, parsed, input)
	}

	for _, input := range []string{"every minute", "0 3 * *", "* * * * * * extra", "99 99 * * *"} {
		_, err := ParseSchedule(input)
		require.Error(t, err, input)
	}
}

func TestNextRun(t *testing.T) {
	reference := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)

	next := NextRun("0 3 * * *", reference)
	require.Equal(t, time.Date(2026, 3, 2, 3, 0, 0, 0, time.UTC), next.UTC())

	hourly := NextRun("@hourly", reference)
	require.Equal(t, time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC), hourly.UTC())

	require.True(t, NextRun("nonsense", reference).IsZero())
}

func TestDue(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)

	// A schedule that never ran is not due yet: it waits for its next slot.
	require.False(t, Due("@hourly", nil, now))

	// The next slot after 11:00 is 12:00, which has passed.
	last := time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC)
	require.True(t, Due("@hourly", &last, now))

	// Already ran this hour.
	recent := time.Date(2026, 3, 1, 12, 15, 0, 0, time.UTC)
	require.False(t, Due("@hourly", &recent, now))

	require.False(t, Due("nonsense", &last, now))
}

func TestDescribeKnownDescriptors(t *testing.T) {
	require.Equal(t, "Every hour", Describe("@hourly"))
	require.Equal(t, "Every day at midnight", Describe("@daily"))
	require.Equal(t, "", Describe("0 3 * * *"))
}

// buildTar creates a tar archive from name/content pairs.
func buildTar(t *testing.T, entries map[string]string) []byte {
	t.Helper()

	buffer := &bytes.Buffer{}
	writer := tar.NewWriter(buffer)

	for name, content := range entries {
		if strings.HasSuffix(name, "/") {
			require.NoError(t, writer.WriteHeader(&tar.Header{
				Name:     strings.TrimSuffix(name, "/"),
				Mode:     0o755,
				Typeflag: tar.TypeDir,
			}))

			continue
		}

		require.NoError(t, writer.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}))
		_, err := writer.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	return buffer.Bytes()
}

func readTar(t *testing.T, payload []byte) map[string]string {
	t.Helper()

	entries := map[string]string{}
	reader := tar.NewReader(bytes.NewReader(payload))

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Typeflag != tar.TypeReg {
			entries[header.Name] = ""

			continue
		}

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		entries[header.Name] = string(content)
	}

	return entries
}

func TestRewriteTarStripsDockerPrefix(t *testing.T) {
	// Docker's copy API wraps the requested directory in a tar entry.
	source := buildTar(t, map[string]string{
		"volume/":         "",
		"volume/app.db":   "database",
		"volume/nested/x": "nested",
	})

	output := &bytes.Buffer{}
	require.NoError(t, rewriteTar(bytes.NewReader(source), output, volumeMountPath, ""))

	entries := readTar(t, output.Bytes())
	require.Equal(t, "database", entries["app.db"])
	require.Equal(t, "nested", entries["nested/x"])
	require.NotContains(t, entries, "volume/")
}

func TestRewriteTarAddsPrefixForRestore(t *testing.T) {
	source := buildTar(t, map[string]string{
		"app.db":   "database",
		"nested/x": "nested",
	})

	output := &bytes.Buffer{}
	require.NoError(t, rewriteTar(bytes.NewReader(source), output, "", volumeMountPath))

	entries := readTar(t, output.Bytes())
	require.Equal(t, "database", entries["volume/app.db"])
	require.Equal(t, "nested", entries["volume/nested/x"])
}

func TestRewriteTarRoundTrip(t *testing.T) {
	original := map[string]string{
		"volume/one.txt": "first",
		"volume/two.txt": "second",
	}

	stripped := &bytes.Buffer{}
	require.NoError(t, rewriteTar(bytes.NewReader(buildTar(t, original)), stripped, volumeMountPath, ""))

	restored := &bytes.Buffer{}
	require.NoError(t, rewriteTar(bytes.NewReader(stripped.Bytes()), restored, "", volumeMountPath))

	entries := readTar(t, restored.Bytes())
	require.Equal(t, "first", entries["volume/one.txt"])
	require.Equal(t, "second", entries["volume/two.txt"])
}

func credentialsWith(user, password, database string) models.DatabaseCredentials {
	return models.DatabaseCredentials{Username: user, Password: password, Database: database}
}

func TestDumpCommandQuotesCredentials(t *testing.T) {
	command, err := dumpCommand("postgres", credentialsWith("o'brien", "p@ss word", "app_db"))
	require.NoError(t, err)
	require.Len(t, command, 3)

	script := command[2]
	require.Contains(t, script, "PGPASSWORD='p@ss word'")
	require.Contains(t, script, `-U 'o'\''brien'`)
	require.Contains(t, script, "-d 'app_db'")
	require.Contains(t, script, "pg_dump")

	mysql, err := dumpCommand("mysql", credentialsWith("ebox", "secret", "app"))
	require.NoError(t, err)
	require.Contains(t, mysql[2], "mysqldump")
	require.Contains(t, mysql[2], "-p'secret'")

	redis, err := dumpCommand("redis", credentialsWith("default", "secret", "0"))
	require.NoError(t, err)
	require.Contains(t, redis[2], "--rdb /dev/stdout")

	_, err = dumpCommand("mongo", credentialsWith("root", "secret", "app"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "volume")
}

func TestRestoreCommand(t *testing.T) {
	command, err := restoreCommand("postgres", credentialsWith("ebox", "secret", "app_db"))
	require.NoError(t, err)
	require.Contains(t, command[2], "psql")
	require.Contains(t, command[2], "ON_ERROR_STOP=1")

	_, err = restoreCommand("redis", credentialsWith("default", "secret", "0"))
	require.Error(t, err)
}

func TestGzipRoundTripOfDump(t *testing.T) {
	// The dump path always gzips, so the restore path must read the same framing.
	payload := "CREATE TABLE t (id int);\nINSERT INTO t VALUES (1);\n"

	compressed, err := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
	require.NoError(t, err)
	require.NoError(t, compressed.Close())

	buffer := &bytes.Buffer{}
	writer, err := gzip.NewWriterLevel(buffer, gzip.BestSpeed)
	require.NoError(t, err)
	_, err = writer.Write([]byte(payload))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader, err := gzip.NewReader(buffer)
	require.NoError(t, err)

	restored, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, payload, string(restored))
	require.True(t, strings.Contains(string(restored), "INSERT INTO t"))
}
