package update

import (
	"archive/tar"
	"compress/gzip"
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
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"brain/internal/buildinfo"
	"brain/internal/config"
)

const (
	defaultAPIBaseURL = "https://api.github.com"
	releaseOwner      = "JimmyMcBride"
	releaseRepo       = "brain"
)

var tagPattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z.-]+)?$`)

var errNoReleases = errors.New("no github releases published yet")

type Options struct {
	APIBaseURL     string
	HTTPClient     *http.Client
	ExecutablePath string
	HomeDir        string
	GOOS           string
	GOARCH         string
	LookPath       func(string) (string, error)
}

type Manager struct {
	cfg    *config.Config
	paths  config.Paths
	client *http.Client
	base   string
	exe    string
	home   string
	goos   string
	goarch string
	look   func(string) (string, error)
}

type Request struct {
	CheckOnly         bool
	IncludePrerelease bool
}

type Result struct {
	CurrentVersion       string `json:"current_version"`
	LatestVersion        string `json:"latest_version"`
	ReleaseTag           string `json:"release_tag"`
	ReleaseURL           string `json:"release_url"`
	Updated              bool   `json:"updated"`
	InstalledPath        string `json:"installed_path"`
	FallbackUsed         bool   `json:"fallback_used"`
	Status               string `json:"status"`
	Message              string `json:"message,omitempty"`
	CurrentPath          string `json:"current_path,omitempty"`
	LookPathTarget       string `json:"look_path_target,omitempty"`
	PathContainsLocalBin bool   `json:"path_contains_local_bin,omitempty"`
}

type release struct {
	TagName     string  `json:"tag_name"`
	HTMLURL     string  `json:"html_url"`
	Prerelease  bool    `json:"prerelease"`
	PublishedAt string  `json:"published_at"`
	Assets      []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type semVersion struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

type apiError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("github api %s: %s", e.Status, e.Body)
}

func New(cfg *config.Config, paths config.Paths, opts Options) *Manager {
	base := strings.TrimRight(opts.APIBaseURL, "/")
	if base == "" {
		base = defaultAPIBaseURL
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	exe := opts.ExecutablePath
	if exe == "" {
		exe, _ = os.Executable()
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	home := opts.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	goos := opts.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := opts.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	look := opts.LookPath
	if look == nil {
		look = exec.LookPath
	}
	return &Manager{
		cfg:    cfg,
		paths:  paths,
		client: client,
		base:   base,
		exe:    exe,
		home:   home,
		goos:   goos,
		goarch: goarch,
		look:   look,
	}
}

func (m *Manager) Update(ctx context.Context, req Request) (Result, error) {
	current := buildinfo.Current()
	result := Result{
		CurrentVersion: current.Version,
		CurrentPath:    m.exe,
	}

	platformOK := slices.Contains([]string{"linux", "darwin"}, m.goos) && slices.Contains([]string{"amd64", "arm64"}, m.goarch)
	if !platformOK {
		result.Status = "unsupported_platform"
		result.Message = fmt.Sprintf("%s/%s is not supported by the release updater", m.goos, m.goarch)
		return result, nil
	}

	rel, err := m.fetchRelease(ctx, req.IncludePrerelease)
	if err != nil {
		if errors.Is(err, errNoReleases) {
			result.Status = "no_releases"
			result.Message = "no GitHub releases published yet"
			return result, nil
		}
		return result, err
	}
	result.LatestVersion = rel.TagName
	result.ReleaseTag = rel.TagName
	result.ReleaseURL = rel.HTMLURL

	if !isNewerVersion(current.Version, rel.TagName) {
		result.Status = "up_to_date"
		result.Message = fmt.Sprintf("already up to date (%s)", chooseVersion(current.Version))
		return result, nil
	}

	if req.CheckOnly {
		result.Status = "update_available"
		result.Message = fmt.Sprintf("%s -> %s", chooseVersion(current.Version), rel.TagName)
		return result, nil
	}

	archiveAsset, checksumsAsset, err := selectAssets(rel, m.goos, m.goarch)
	if err != nil {
		return result, err
	}
	installPath, fallbackUsed, err := m.chooseInstallPath()
	if err != nil {
		return result, err
	}

	tmpDir, err := os.MkdirTemp("", "brain-update-*")
	if err != nil {
		return result, fmt.Errorf("create update temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archiveAsset.Name)
	if err := m.downloadFile(ctx, archiveAsset.BrowserDownloadURL, archivePath); err != nil {
		return result, err
	}
	checksumPath := filepath.Join(tmpDir, checksumsAsset.Name)
	if err := m.downloadFile(ctx, checksumsAsset.BrowserDownloadURL, checksumPath); err != nil {
		return result, err
	}
	if err := verifyChecksumFile(archivePath, checksumPath, archiveAsset.Name); err != nil {
		return result, err
	}

	extractedPath := filepath.Join(tmpDir, "brain")
	if err := extractBinary(archivePath, extractedPath); err != nil {
		return result, err
	}
	if err := m.installBinary(extractedPath, installPath); err != nil {
		return result, err
	}

	result.Updated = true
	result.InstalledPath = installPath
	result.FallbackUsed = fallbackUsed
	if fallbackUsed {
		result.Status = "installed_to_fallback"
	} else {
		result.Status = "updated"
	}
	result.PathContainsLocalBin, result.LookPathTarget = m.pathDetails()
	if fallbackUsed && result.LookPathTarget != "" && !sameFilePath(result.LookPathTarget, installPath) {
		result.Message = fmt.Sprintf("installed %s, but `brain` may still resolve to %s", installPath, result.LookPathTarget)
	} else {
		result.Message = fmt.Sprintf("%s -> %s", chooseVersion(current.Version), rel.TagName)
	}
	return result, nil
}

func chooseVersion(v string) string {
	if strings.TrimSpace(v) == "" {
		return "dev"
	}
	return v
}

func (m *Manager) fetchRelease(ctx context.Context, includePrerelease bool) (*release, error) {
	if !includePrerelease {
		var rel release
		if err := m.getJSON(ctx, fmt.Sprintf("%s/repos/%s/%s/releases/latest", m.base, releaseOwner, releaseRepo), &rel); err != nil {
			var apiErr *apiError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
				return nil, errNoReleases
			}
			return nil, fmt.Errorf("fetch latest release: %w", err)
		}
		return &rel, nil
	}

	var releases []release
	if err := m.getJSON(ctx, fmt.Sprintf("%s/repos/%s/%s/releases", m.base, releaseOwner, releaseRepo), &releases); err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	var best *release
	var bestVersion semVersion
	for i := range releases {
		v, ok := parseSemver(releases[i].TagName)
		if !ok {
			continue
		}
		if best == nil || compareSemver(v, bestVersion) > 0 {
			best = &releases[i]
			bestVersion = v
		}
	}
	if best == nil {
		return nil, errNoReleases
	}
	return best, nil
}

func (m *Manager) getJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "brain-updater")
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &apiError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func (m *Manager) downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "brain-updater")
	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("download %s: %s %s", url, resp.Status, strings.TrimSpace(string(body)))
	}
	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create download %s: %w", dest, err)
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("write download %s: %w", dest, err)
	}
	return nil
}

func (m *Manager) chooseInstallPath() (string, bool, error) {
	if isWritableTarget(m.exe) {
		return m.exe, false, nil
	}
	localBin := filepath.Join(m.home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		return "", false, fmt.Errorf("create fallback bin dir: %w", err)
	}
	return filepath.Join(localBin, "brain"), true, nil
}

func (m *Manager) installBinary(extractedPath, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}
	if err := os.MkdirAll(m.paths.UpdateBackupDir, 0o755); err != nil {
		return fmt.Errorf("create update backup dir: %w", err)
	}

	if _, err := os.Stat(target); err == nil {
		backupName := fmt.Sprintf("brain_%s_%s", time.Now().UTC().Format("20060102T150405Z"), sanitizeVersion(chooseVersion(buildinfo.Version)))
		backupPath := filepath.Join(m.paths.UpdateBackupDir, backupName)
		if err := copyFile(target, backupPath, 0o755); err != nil {
			return fmt.Errorf("backup current binary: %w", err)
		}
	}

	tempFile, err := os.CreateTemp(filepath.Dir(target), ".brain-update-*")
	if err != nil {
		return fmt.Errorf("create install temp file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	if err := copyFile(extractedPath, tempPath, 0o755); err != nil {
		return fmt.Errorf("stage updated binary: %w", err)
	}
	if err := os.Chmod(tempPath, 0o755); err != nil {
		return fmt.Errorf("chmod updated binary: %w", err)
	}
	if err := os.Rename(tempPath, target); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

func (m *Manager) pathDetails() (bool, string) {
	localBin := filepath.Join(m.home, ".local", "bin")
	contains := pathContains(localBin, os.Getenv("PATH"))
	found, err := m.look("brain")
	if err != nil {
		return contains, ""
	}
	if resolved, err := filepath.EvalSymlinks(found); err == nil {
		found = resolved
	}
	return contains, found
}

func selectAssets(rel *release, goos, goarch string) (asset, asset, error) {
	archiveName := fmt.Sprintf("brain_%s_%s_%s.tar.gz", rel.TagName, goos, goarch)
	checksumName := fmt.Sprintf("brain_%s_checksums.txt", rel.TagName)
	var archiveAsset asset
	var checksumsAsset asset
	for _, candidate := range rel.Assets {
		switch candidate.Name {
		case archiveName:
			archiveAsset = candidate
		case checksumName:
			checksumsAsset = candidate
		}
	}
	if archiveAsset.Name == "" {
		return asset{}, asset{}, fmt.Errorf("no release asset for %s/%s", goos, goarch)
	}
	if checksumsAsset.Name == "" {
		return asset{}, asset{}, errors.New("release checksums asset missing")
	}
	return archiveAsset, checksumsAsset, nil
}

func verifyChecksumFile(archivePath, checksumsPath, assetName string) error {
	want, err := checksumForAsset(checksumsPath, assetName)
	if err != nil {
		return err
	}
	got, err := fileChecksum(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("checksum mismatch for %s", assetName)
	}
	return nil
}

func checksumForAsset(checksumsPath, assetName string) (string, error) {
	raw, err := os.ReadFile(checksumsPath)
	if err != nil {
		return "", fmt.Errorf("read checksums: %w", err)
	}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[len(fields)-1] == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum entry missing for %s", assetName)
}

func fileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func extractBinary(archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != "brain" {
			continue
		}
		out, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("create extracted binary: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return fmt.Errorf("extract binary: %w", err)
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("close extracted binary: %w", err)
		}
		if err := os.Chmod(dest, 0o755); err != nil {
			return fmt.Errorf("chmod extracted binary: %w", err)
		}
		return nil
	}
	return errors.New("brain binary not found in archive")
}

func copyFile(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func isWritableTarget(target string) bool {
	if target == "" {
		return false
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	file, err := os.CreateTemp(dir, ".brain-writecheck-*")
	if err != nil {
		return false
	}
	path := file.Name()
	file.Close()
	os.Remove(path)
	return true
}

func pathContains(dir, pathEnv string) bool {
	cleanDir := filepath.Clean(dir)
	for _, entry := range filepath.SplitList(pathEnv) {
		if filepath.Clean(entry) == cleanDir {
			return true
		}
	}
	return false
}

func sanitizeVersion(version string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-")
	return replacer.Replace(version)
}

func sameFilePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func parseSemver(tag string) (semVersion, bool) {
	matches := tagPattern.FindStringSubmatch(strings.TrimSpace(tag))
	if matches == nil {
		return semVersion{}, false
	}
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	return semVersion{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: strings.TrimPrefix(matches[4], "-"),
	}, true
}

func compareSemver(a, b semVersion) int {
	if a.Major != b.Major {
		return compareInt(a.Major, b.Major)
	}
	if a.Minor != b.Minor {
		return compareInt(a.Minor, b.Minor)
	}
	if a.Patch != b.Patch {
		return compareInt(a.Patch, b.Patch)
	}
	if a.Prerelease == "" && b.Prerelease == "" {
		return 0
	}
	if a.Prerelease == "" {
		return 1
	}
	if b.Prerelease == "" {
		return -1
	}
	if a.Prerelease == b.Prerelease {
		return 0
	}
	return comparePrerelease(a.Prerelease, b.Prerelease)
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func comparePrerelease(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		if i >= len(aParts) {
			return -1
		}
		if i >= len(bParts) {
			return 1
		}
		if aParts[i] == bParts[i] {
			continue
		}
		aNum, aErr := strconv.Atoi(aParts[i])
		bNum, bErr := strconv.Atoi(bParts[i])
		switch {
		case aErr == nil && bErr == nil:
			return compareInt(aNum, bNum)
		case aErr == nil:
			return -1
		case bErr == nil:
			return 1
		default:
			if aParts[i] < bParts[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func isNewerVersion(current, latest string) bool {
	currentVersion, currentOK := parseSemver(current)
	latestVersion, latestOK := parseSemver(latest)
	if !latestOK {
		return false
	}
	if !currentOK {
		return true
	}
	return compareSemver(latestVersion, currentVersion) > 0
}
