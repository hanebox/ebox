package system

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ebox/app/facades"
	"ebox/app/support/version"
)

const (
	releaseTimeout   = 30 * time.Second
	downloadTimeout  = 15 * time.Minute
	maxBinarySize    = 300 << 20
	previousSuffix   = ".previous"
	checksumFileName = "checksums.txt"
)

// Release is a published build of the panel.
type Release struct {
	Version     string    `json:"version"`
	Name        string    `json:"name"`
	Notes       string    `json:"notes,omitempty"`
	URL         string    `json:"url,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	AssetName   string    `json:"asset_name,omitempty"`
	AssetURL    string    `json:"asset_url,omitempty"`
	AssetSize   int64     `json:"asset_size,omitempty"`
	ChecksumURL string    `json:"-"`
	Prerelease  bool      `json:"prerelease,omitempty"`
}

// UpdateStatus is what the panel shows on the settings page.
type UpdateStatus struct {
	CurrentVersion  string    `json:"current_version"`
	CurrentCommit   string    `json:"current_commit"`
	BuildDate       string    `json:"build_date"`
	Latest          *Release  `json:"latest,omitempty"`
	UpdateAvailable bool      `json:"update_available"`
	CheckError      string    `json:"check_error,omitempty"`
	CheckedAt       time.Time `json:"checked_at"`
	// Managed reports whether a systemd unit manages this process, which decides
	// how the panel restarts itself.
	Managed        bool   `json:"managed"`
	SelfUpdateable bool   `json:"self_updateable"`
	Reason         string `json:"reason,omitempty"`
	AssetName      string `json:"asset_name,omitempty"`
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// Updater checks for and applies panel updates.
type Updater struct {
	mu   sync.Mutex
	busy bool
	// sink receives progress lines while an update runs.
	sink func(string)
}

var defaultUpdater = &Updater{}

func Default() *Updater {
	return defaultUpdater
}

// repository is the GitHub repository releases are read from.
func (u *Updater) repository() string {
	return facades.Config().GetString("ebox.update.repository", "buildandbuild/ebox")
}

// AssetName is the release asset for this platform.
func AssetName() string {
	return fmt.Sprintf("ebox-%s-%s", runtime.GOOS, runtime.GOARCH)
}

// Status reports the current version and the newest release.
func (u *Updater) Status(ctx context.Context, check bool) UpdateStatus {
	status := UpdateStatus{
		CurrentVersion: version.Version,
		CurrentCommit:  version.Commit,
		BuildDate:      version.Date,
		CheckedAt:      time.Now().UTC(),
		AssetName:      AssetName(),
		Managed:        managed(),
	}

	status.SelfUpdateable, status.Reason = u.capability()
	if !check {
		return status
	}

	release, err := u.Latest(ctx)
	if err != nil {
		status.CheckError = err.Error()

		return status
	}

	status.Latest = release
	status.UpdateAvailable = newer(release.Version, version.Version)

	return status
}

// Latest returns the newest release with an asset for this platform.
func (u *Updater) Latest(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20", u.repository())

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "ebox/"+version.Version)

	client := &http.Client{Timeout: releaseTimeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("The release feed could not be reached.")
	}
	defer func() { _ = response.Body.Close() }()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, fmt.Errorf("No releases found for %s.", u.repository())
	default:
		return nil, fmt.Errorf("The release feed answered with status %d.", response.StatusCode)
	}

	var payload []githubRelease
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, errors.New("The release feed returned an unexpected payload.")
	}

	asset := AssetName()

	for _, item := range payload {
		if item.Prerelease {
			continue
		}

		release := &Release{
			Version:     strings.TrimPrefix(item.TagName, "v"),
			Name:        item.Name,
			Notes:       truncateNotes(item.Body),
			URL:         item.HTMLURL,
			Prerelease:  item.Prerelease,
			PublishedAt: parseTime(item.PublishedAt),
		}

		for _, candidate := range item.Assets {
			switch candidate.Name {
			case asset:
				release.AssetName = candidate.Name
				release.AssetURL = candidate.BrowserDownloadURL
				release.AssetSize = candidate.Size
			case checksumFileName:
				release.ChecksumURL = candidate.BrowserDownloadURL
			}
		}

		if release.AssetURL == "" {
			continue
		}

		return release, nil
	}

	return nil, fmt.Errorf("Release %s has no build for %s.", payload[0].TagName, AssetName())
}

// Apply downloads and installs a release, replacing the running binary.
func (u *Updater) Apply(ctx context.Context, release *Release, sink func(string)) error {
	u.mu.Lock()
	if u.busy {
		u.mu.Unlock()

		return errors.New("An update is already running.")
	}
	u.busy = true
	u.sink = sink
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		u.busy = false
		u.sink = nil
		u.mu.Unlock()
	}()

	if release == nil || release.AssetURL == "" {
		return errors.New("No release was selected.")
	}

	if !managed() {
		return errors.New("This panel is not managed by systemd, so it cannot restart itself. Update the binary manually.")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	u.log("Downloading " + release.AssetName + " (" + humanSize(release.AssetSize) + ")")

	archive, err := os.CreateTemp(facades.Config().GetString("ebox.data_dir"), "ebox-update-*")
	if err != nil {
		return err
	}
	archivePath := archive.Name()
	defer func() {
		_ = archive.Close()
		_ = os.Remove(archivePath)
	}()

	digest, err := downloadFile(ctx, release.AssetURL, archive)
	if err != nil {
		return err
	}

	if release.ChecksumURL != "" {
		u.log("Verifying checksum")
		expected, err := u.checksum(ctx, release.ChecksumURL, release.AssetName)
		if err != nil {
			return err
		}
		if !strings.EqualFold(expected, digest) {
			return errors.New("The downloaded file does not match the published checksum.")
		}
	} else {
		u.log("Warning: this release has no " + checksumFileName + ", skipping checksum verification")
	}

	target, err := os.Executable()
	if err != nil {
		return err
	}
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return err
	}

	u.log("Installing to " + target)

	if err := installBinary(archivePath, target); err != nil {
		return err
	}

	u.log("Update installed. Restarting the panel.")

	return nil
}

// Rollback restores the binary that was replaced by the last update.
func (u *Updater) Rollback() error {
	target, err := os.Executable()
	if err != nil {
		return err
	}
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return err
	}

	previous := target + previousSuffix
	if _, err := os.Stat(previous); err != nil {
		return errors.New("There is no previous version to roll back to.")
	}

	current := target + ".rollback"
	if err := copyFile(target, current, 0o755); err != nil {
		return err
	}

	if err := os.Rename(previous, target); err != nil {
		_ = os.Remove(current)

		return err
	}

	_ = os.Rename(current, previous)

	return nil
}

// installBinary swaps the binary in place, keeping the previous build so a
// rollback is always possible.
func installBinary(source, target string) error {
	directory := filepath.Dir(target)

	staged, err := os.CreateTemp(directory, ".ebox-staged-*")
	if err != nil {
		return errors.New("The panel binary directory is not writable: " + err.Error())
	}
	stagedPath := staged.Name()

	defer func() { _ = os.Remove(stagedPath) }()

	if err := copyInto(source, staged); err != nil {
		_ = staged.Close()

		return err
	}

	if err := staged.Sync(); err != nil {
		_ = staged.Close()

		return err
	}

	if err := staged.Close(); err != nil {
		return err
	}

	if err := os.Chmod(stagedPath, 0o755); err != nil {
		return err
	}

	previous := target + previousSuffix
	_ = os.Remove(previous)

	if err := copyFile(target, previous, 0o755); err != nil {
		return errors.New("The current binary could not be backed up: " + err.Error())
	}

	if err := os.Rename(stagedPath, target); err != nil {
		return errors.New("The new binary could not be moved into place: " + err.Error())
	}

	return nil
}

// copyInto streams source into an open file.
func copyInto(source string, destination *os.File) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()

	if _, err := io.Copy(destination, input); err != nil {
		return errors.New("The release asset could not be written: " + err.Error())
	}

	return nil
}

// downloadFile streams a URL into a file and returns its sha256 digest.
func downloadFile(ctx context.Context, url string, destination io.Writer) (string, error) {
	client := &http.Client{Timeout: downloadTimeout}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "ebox/"+version.Version)

	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("The download failed: " + err.Error())
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("The download failed with status %d.", response.StatusCode)
	}

	if response.ContentLength > maxBinarySize {
		return "", errors.New("The release asset is larger than the panel allows.")
	}

	hash := sha256.New()
	limited := io.LimitReader(response.Body, maxBinarySize)

	if _, err := io.Copy(io.MultiWriter(destination, hash), limited); err != nil {
		return "", errors.New("The download was interrupted: " + err.Error())
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// checksum reads the sha256 for an asset from a checksums file.
func (u *Updater) checksum(ctx context.Context, url, asset string) (string, error) {
	client := &http.Client{Timeout: releaseTimeout}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "ebox/"+version.Version)

	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("The checksum file could not be downloaded.")
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", errors.New("The checksum file could not be read.")
	}

	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		name := strings.TrimPrefix(fields[1], "*")
		if name == asset {
			return fields[0], nil
		}
	}

	return "", errors.New("The checksum file has no entry for " + asset + ".")
}

func (u *Updater) log(line string) {
	u.mu.Lock()
	sink := u.sink
	u.mu.Unlock()

	if sink != nil {
		sink(line)
	}
}

// Restart asks systemd to restart the panel unit. systemd-run is preferred: it
// runs outside the unit cgroup so the restart survives stopping this process.
func (u *Updater) Restart(unit string) error {
	if !managed() {
		return errors.New("This panel is not managed by systemd, so it cannot restart itself.")
	}

	if _, err := exec.LookPath("systemd-run"); err == nil {
		// systemd-run needs a clean environment; the unit cgroup is not inherited.
		if err := runCommand("systemd-run", "--no-block", "--collect", "--quiet", "systemctl", "restart", unit); err == nil {
			return nil
		}
	}

	if err := runCommand("systemctl", "--no-block", "restart", unit); err != nil {
		return errors.New("systemd refused the restart: " + err.Error())
	}

	return nil
}

func runCommand(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdout = nil
	command.Stderr = nil

	return command.Run()
}

// managed reports whether systemd owns this process.
func managed() bool {
	if os.Getenv("INVOCATION_ID") != "" || os.Getenv("JOURNAL_STREAM") != "" {
		return true
	}

	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return false
	}

	if _, err := os.Stat(facades.Config().GetString("ebox.systemd_unit", "/etc/systemd/system/ebox.service")); err == nil {
		return true
	}

	return false
}

// capability explains whether self-update can run at all.
func (u *Updater) capability() (bool, string) {
	if !managed() {
		return false, "The panel is not running under systemd, so it cannot restart itself after an update."
	}

	executable, err := os.Executable()
	if err != nil {
		return false, "The panel binary path could not be determined."
	}

	directory := filepath.Dir(executable)
	probe, err := os.CreateTemp(directory, ".ebox-writable-*")
	if err != nil {
		return false, "The panel binary directory is not writable (" + directory + ")."
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())

	return true, ""
}

// newer reports whether candidate is newer than current.
func newer(candidate, current string) bool {
	if candidate == "" {
		return false
	}

	clean := func(value string) string {
		return strings.TrimPrefix(strings.TrimSpace(value), "v")
	}

	candidate, current = clean(candidate), clean(current)

	if current == "dev" || current == "" {
		return candidate != ""
	}

	if candidate == current {
		return false
	}

	// A prerelease of the same version is not an upgrade.
	baseLeft, suffixLeft := splitVersion(candidate)
	baseRight, suffixRight := splitVersion(current)

	left := strings.Split(baseLeft, ".")
	right := strings.Split(baseRight, ".")

	for index := range 3 {
		leftPart := 0
		rightPart := 0

		if index < len(left) {
			leftPart = leadingNumber(left[index])
		}
		if index < len(right) {
			rightPart = leadingNumber(right[index])
		}

		if leftPart != rightPart {
			return leftPart > rightPart
		}
	}

	// Same numbers: only a newer prerelease suffix counts, and a release always
	// beats its own prereleases.
	if suffixLeft == suffixRight {
		return false
	}
	if suffixLeft != "" && suffixRight == "" {
		return false
	}
	if suffixLeft == "" && suffixRight != "" {
		return true
	}

	return suffixLeft > suffixRight
}

// splitVersion separates a version from its prerelease suffix.
func splitVersion(value string) (string, string) {
	if index := strings.IndexAny(value, "-+"); index >= 0 {
		return value[:index], value[index+1:]
	}

	return value, ""
}

func leadingNumber(value string) int {
	digits := 0

	for _, character := range value {
		if character < '0' || character > '9' {
			break
		}

		digits = digits*10 + int(character-'0')
	}

	return digits
}

func copyFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()

	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()

		return err
	}

	if err := output.Sync(); err != nil {
		_ = output.Close()

		return err
	}

	return output.Close()
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	return parsed.UTC()
}

func truncateNotes(notes string) string {
	notes = strings.TrimSpace(notes)
	if len(notes) <= 4000 {
		return notes
	}

	return notes[:4000] + "…"
}

func humanSize(bytes int64) string {
	const unit = 1024

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	value := float64(bytes)
	units := []string{"kB", "MB", "GB", "TB"}
	index := -1

	for value >= unit && index < len(units)-1 {
		value /= unit
		index++
	}

	return fmt.Sprintf("%.1f %s", value, units[index])
}
