package system

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewerComparesVersions(t *testing.T) {
	cases := []struct {
		candidate string
		current   string
		expected  bool
	}{
		{"0.5.0", "0.4.0", true},
		{"v0.5.0", "0.4.0", true},
		{"0.4.0", "0.5.0", false},
		{"0.5.0", "0.5.0", false},
		{"v1.0.0", "0.9.9", true},
		{"0.4.10", "0.4.9", true},
		{"0.10.0", "0.9.0", true},
		{"1.2.3", "dev", true},
		{"", "0.4.0", false},
		{"0.4.0-beta.1", "0.4.0", false},
	}

	for _, testCase := range cases {
		require.Equal(t, testCase.expected, newer(testCase.candidate, testCase.current),
			"%s vs %s", testCase.candidate, testCase.current)
	}
}

func TestAssetNameMatchesPlatform(t *testing.T) {
	require.Equal(t, "ebox-"+runtime.GOOS+"-"+runtime.GOARCH, AssetName())
}

func TestHumanSize(t *testing.T) {
	require.Equal(t, "512 B", humanSize(512))
	require.Equal(t, "1.0 kB", humanSize(1024))
	require.Equal(t, "1.5 MB", humanSize(1536*1024))
}

func TestInstallBinaryKeepsPreviousBuild(t *testing.T) {
	directory := t.TempDir()

	target := filepath.Join(directory, "ebox")
	require.NoError(t, os.WriteFile(target, []byte("old-binary"), 0o755))

	staged := filepath.Join(directory, "new-binary")
	require.NoError(t, os.WriteFile(staged, []byte("new-binary"), 0o644))

	require.NoError(t, installBinary(staged, target))

	current, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "new-binary", string(current))

	info, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), info.Mode().Perm())

	previous, err := os.ReadFile(target + previousSuffix)
	require.NoError(t, err)
	require.Equal(t, "old-binary", string(previous))

	// The staged file stays on disk: the caller owns the temp file lifetime.
	_, err = os.Stat(staged)
	require.NoError(t, err)
}

func TestRollbackRequiresPreviousBuild(t *testing.T) {
	updater := &Updater{}

	// The test binary has no .previous next to it, so rollback must refuse.
	err := updater.Rollback()
	require.Error(t, err)
	require.Contains(t, err.Error(), "previous")
}

func TestStatusReportsVersionAndCapability(t *testing.T) {
	status := Default().Status(context.Background(), false)

	require.NotEmpty(t, status.CurrentVersion)
	require.NotEmpty(t, status.AssetName)
	require.False(t, status.CheckedAt.IsZero())

	// On a machine without systemd self-update must be refused with a reason.
	if !status.Managed {
		require.False(t, status.SelfUpdateable)
		require.NotEmpty(t, status.Reason)
	}
}

func TestDownloadFileStreamsAndHashes(t *testing.T) {
	payload := []byte("ebox-binary-payload")

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(payload)
	}))
	defer server.Close()

	destination := &bytes.Buffer{}
	digest, err := downloadFile(context.Background(), server.URL, destination)
	require.NoError(t, err)

	sum := sha256.Sum256(payload)
	require.Equal(t, hex.EncodeToString(sum[:]), digest)
	require.Equal(t, payload, destination.Bytes())
}

func TestDownloadFileReportsHTTPFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := downloadFile(context.Background(), server.URL, &bytes.Buffer{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "404")
}

func TestChecksumParsingFromReleaseFile(t *testing.T) {
	asset := "ebox-linux-amd64"
	digest := "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"

	content := fmt.Sprintf("aaaa  other-asset\n%s  *%s\n", digest, asset)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(content))
	}))
	defer server.Close()

	updater := &Updater{}

	found, err := updater.checksum(context.Background(), server.URL, asset)
	require.NoError(t, err)
	require.Equal(t, digest, found)

	_, err = updater.checksum(context.Background(), server.URL, "ebox-linux-arm64")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no entry")
}

func TestApplyRefusesWithoutSystemd(t *testing.T) {
	// This test host is not managed by systemd, so applying must refuse before
	// downloading anything.
	release := &Release{Version: "9.9.9", AssetName: "ebox-test", AssetURL: "https://example.invalid/ebox"}

	err := Default().Apply(context.Background(), release, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "systemd")
}
