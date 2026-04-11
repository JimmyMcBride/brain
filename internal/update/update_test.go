package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/buildinfo"
	"brain/internal/config"
)

func newUpdateTestSetup(t *testing.T, root string) (*config.Config, config.Paths) {
	t.Helper()
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data-home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config-home"))
	cfg, paths, err := config.LoadOrCreate(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg, paths
}

func TestIsNewerVersion(t *testing.T) {
	if !isNewerVersion("v0.1.0", "v0.2.0") {
		t.Fatal("expected newer release")
	}
	if isNewerVersion("v0.2.0", "v0.1.0") {
		t.Fatal("did not expect older release to compare newer")
	}
	if !isNewerVersion("dev", "v0.1.0") {
		t.Fatal("expected dev build to be older than release")
	}
	if !isNewerVersion("v0.2.0-rc.1", "v0.2.0") {
		t.Fatal("expected stable release to be newer than prerelease")
	}
}

func TestVerifyChecksumFileAndExtractBinary(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "brain_v1.2.3_linux_amd64.tar.gz")
	binary := []byte("new-brain-binary")
	if err := os.WriteFile(archivePath, mustArchive(t, binary), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(mustRead(t, archivePath))
	checksums := hex.EncodeToString(sum[:]) + "  " + filepath.Base(archivePath) + "\n"
	checksumPath := filepath.Join(root, "checksums.txt")
	if err := os.WriteFile(checksumPath, []byte(checksums), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksumFile(archivePath, checksumPath, filepath.Base(archivePath)); err != nil {
		t.Fatalf("expected checksum verification to pass: %v", err)
	}
	dest := filepath.Join(root, "brain")
	if err := extractBinary(archivePath, dest, "linux"); err != nil {
		t.Fatalf("expected extraction to pass: %v", err)
	}
	if got := string(mustRead(t, dest)); got != string(binary) {
		t.Fatalf("unexpected extracted binary: %s", got)
	}
}

func TestExtractBinaryWindowsZip(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "brain_v1.2.3_windows_amd64.zip")
	binary := []byte("new-brain-binary")
	if err := os.WriteFile(archivePath, mustZipArchive(t, binary), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(root, "brain.exe")
	if err := extractBinary(archivePath, dest, "windows"); err != nil {
		t.Fatalf("expected zip extraction to pass: %v", err)
	}
	if got := string(mustRead(t, dest)); got != string(binary) {
		t.Fatalf("unexpected extracted binary: %s", got)
	}
}

func TestManagerUpdateCheckOnly(t *testing.T) {
	restore := setBuildInfo("v0.1.0")
	defer restore()

	server, _ := newReleaseServer(t, []release{
		makeRelease("v0.2.0", false, "linux", "amd64", []byte("brain-v0.2.0")),
	})
	defer server.Close()

	root := t.TempDir()
	cfg, paths := newUpdateTestSetup(t, root)
	if err := os.MkdirAll(paths.UpdateBackupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manager := New(cfg, paths, Options{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: filepath.Join(root, "brain"),
		HomeDir:        filepath.Join(root, "home"),
		GOOS:           "linux",
		GOARCH:         "amd64",
	})
	result, err := manager.Update(context.Background(), Request{CheckOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "update_available" {
		t.Fatalf("unexpected status: %+v", result)
	}
	if result.Updated {
		t.Fatalf("did not expect installation during check-only: %+v", result)
	}
}

func TestManagerUpdateInPlace(t *testing.T) {
	restore := setBuildInfo("v0.1.0")
	defer restore()

	server, _ := newReleaseServer(t, []release{
		makeRelease("v0.2.0", false, "linux", "amd64", []byte("brain-v0.2.0")),
	})
	defer server.Close()

	root := t.TempDir()
	cfg, paths := newUpdateTestSetup(t, root)
	if err := os.MkdirAll(paths.UpdateBackupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(root, "brain")
	if err := os.WriteFile(exePath, []byte("brain-v0.1.0"), 0o755); err != nil {
		t.Fatal(err)
	}

	manager := New(cfg, paths, Options{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: exePath,
		HomeDir:        filepath.Join(root, "home"),
		GOOS:           "linux",
		GOARCH:         "amd64",
		LookPath:       func(string) (string, error) { return exePath, nil },
	})

	result, err := manager.Update(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "updated" || !result.Updated {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := string(mustRead(t, exePath)); got != "brain-v0.2.0" {
		t.Fatalf("expected binary replacement, got %q", got)
	}
	backups, err := filepath.Glob(filepath.Join(paths.UpdateBackupDir, "brain_*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) == 0 {
		t.Fatal("expected backup to be created")
	}
}

func TestManagerUpdateFallbackInstall(t *testing.T) {
	restore := setBuildInfo("v0.1.0")
	defer restore()

	server, _ := newReleaseServer(t, []release{
		makeRelease("v0.2.0", false, "linux", "amd64", []byte("brain-v0.2.0")),
	})
	defer server.Close()

	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, paths := newUpdateTestSetup(t, root)
	if err := os.MkdirAll(paths.UpdateBackupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lockedDir := filepath.Join(root, "locked")
	if err := os.MkdirAll(lockedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(lockedDir, "brain")
	if err := os.WriteFile(exePath, []byte("brain-v0.1.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(lockedDir, 0o755)
	t.Setenv("PATH", filepath.Join(root, "bin"))

	manager := New(cfg, paths, Options{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: exePath,
		HomeDir:        home,
		GOOS:           "linux",
		GOARCH:         "amd64",
		LookPath:       func(string) (string, error) { return exePath, nil },
	})

	result, err := manager.Update(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "installed_to_fallback" || !result.FallbackUsed {
		t.Fatalf("unexpected result: %+v", result)
	}
	localPath := filepath.Join(home, ".local", "bin", "brain")
	if result.InstalledPath != localPath {
		t.Fatalf("unexpected install path: %+v", result)
	}
	if got := string(mustRead(t, localPath)); got != "brain-v0.2.0" {
		t.Fatalf("expected fallback install content, got %q", got)
	}
}

func TestManagerUpdatePrerelease(t *testing.T) {
	restore := setBuildInfo("v0.2.0")
	defer restore()

	server, _ := newReleaseServer(t, []release{
		makeRelease("v0.2.0", false, "linux", "amd64", []byte("brain-v0.2.0")),
		makeRelease("v0.3.0-rc.1", true, "linux", "amd64", []byte("brain-v0.3.0-rc.1")),
	})
	defer server.Close()

	root := t.TempDir()
	cfg, paths := newUpdateTestSetup(t, root)
	if err := os.MkdirAll(paths.UpdateBackupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(root, "brain")
	if err := os.WriteFile(exePath, []byte("brain-v0.2.0"), 0o755); err != nil {
		t.Fatal(err)
	}

	manager := New(cfg, paths, Options{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: exePath,
		HomeDir:        filepath.Join(root, "home"),
		GOOS:           "linux",
		GOARCH:         "amd64",
		LookPath:       func(string) (string, error) { return exePath, nil },
	})

	result, err := manager.Update(context.Background(), Request{CheckOnly: true, IncludePrerelease: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.LatestVersion != "v0.3.0-rc.1" {
		t.Fatalf("expected prerelease selection, got %+v", result)
	}
}

func TestManagerChecksumMismatch(t *testing.T) {
	restore := setBuildInfo("v0.1.0")
	defer restore()

	rel := makeRelease("v0.2.0", false, "linux", "amd64", []byte("brain-v0.2.0"))
	server, assets := newReleaseServer(t, []release{rel})
	defer server.Close()
	assets[0].checksums = "deadbeef  " + assets[0].archiveName + "\n"

	root := t.TempDir()
	cfg, paths := newUpdateTestSetup(t, root)
	if err := os.MkdirAll(paths.UpdateBackupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(root, "brain")
	if err := os.WriteFile(exePath, []byte("brain-v0.1.0"), 0o755); err != nil {
		t.Fatal(err)
	}

	manager := New(cfg, paths, Options{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: exePath,
		HomeDir:        filepath.Join(root, "home"),
		GOOS:           "linux",
		GOARCH:         "amd64",
	})
	if _, err := manager.Update(context.Background(), Request{}); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestManagerUnsupportedPlatform(t *testing.T) {
	restore := setBuildInfo("v0.1.0")
	defer restore()

	root := t.TempDir()
	cfg, paths := newUpdateTestSetup(t, root)
	manager := New(cfg, paths, Options{
		ExecutablePath: filepath.Join(root, "brain"),
		HomeDir:        filepath.Join(root, "home"),
		GOOS:           "plan9",
		GOARCH:         "amd64",
	})
	result, err := manager.Update(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unsupported_platform" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestManagerUpdateWindowsFallbackInstall(t *testing.T) {
	restore := setBuildInfo("v0.1.0")
	defer restore()

	server, _ := newReleaseServerForPlatform(t, "windows", "amd64", []release{
		makeRelease("v0.2.0", false, "windows", "amd64", []byte("brain-v0.2.0")),
	})
	defer server.Close()

	root := t.TempDir()
	home := filepath.Join(root, "home")
	localAppData := filepath.Join(root, "LocalAppData")
	t.Setenv("LOCALAPPDATA", localAppData)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, paths := newUpdateTestSetup(t, root)
	if err := os.MkdirAll(paths.UpdateBackupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lockedDir := filepath.Join(root, "locked")
	if err := os.MkdirAll(lockedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(lockedDir, "brain.exe")
	if err := os.WriteFile(exePath, []byte("brain-v0.1.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(lockedDir, 0o755)

	manager := New(cfg, paths, Options{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: exePath,
		HomeDir:        home,
		GOOS:           "windows",
		GOARCH:         "amd64",
		LookPath:       func(string) (string, error) { return exePath, nil },
	})

	result, err := manager.Update(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "installed_to_fallback" || !result.FallbackUsed {
		t.Fatalf("unexpected result: %+v", result)
	}
	wantPath := filepath.Join(localAppData, "Programs", "brain", "brain.exe")
	if result.InstalledPath != wantPath {
		t.Fatalf("unexpected install path: %+v", result)
	}
	if got := string(mustRead(t, wantPath)); got != "brain-v0.2.0" {
		t.Fatalf("expected fallback install content, got %q", got)
	}
}

func TestManagerNoReleases(t *testing.T) {
	restore := setBuildInfo("dev")
	defer restore()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+releaseOwner+"/"+releaseRepo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	root := t.TempDir()
	cfg, paths := newUpdateTestSetup(t, root)
	manager := New(cfg, paths, Options{
		APIBaseURL:     server.URL,
		HTTPClient:     server.Client(),
		ExecutablePath: filepath.Join(root, "brain"),
		HomeDir:        filepath.Join(root, "home"),
		GOOS:           "linux",
		GOARCH:         "amd64",
	})

	result, err := manager.Update(context.Background(), Request{CheckOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "no_releases" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

type servedAsset struct {
	tag         string
	archiveName string
	archive     []byte
	checksums   string
}

func newReleaseServer(t *testing.T, releases []release) (*httptest.Server, []*servedAsset) {
	return newReleaseServerForPlatform(t, "linux", "amd64", releases)
}

func newReleaseServerForPlatform(t *testing.T, goos, goarch string, releases []release) (*httptest.Server, []*servedAsset) {
	t.Helper()
	assets := make([]*servedAsset, 0, len(releases))
	for _, rel := range releases {
		archiveAsset, checksumsAsset, err := selectAssets(&rel, goos, goarch)
		if err != nil {
			t.Fatal(err)
		}
		archive := []byte("brain-" + rel.TagName)
		for _, candidate := range rel.Assets {
			if candidate.Name == archiveAsset.Name && candidate.BrowserDownloadURL != "" {
				archive = []byte(candidate.BrowserDownloadURL)
				break
			}
		}
		archiveBytes := mustReleaseArchive(t, goos, archive)
		sum := sha256.Sum256(archiveBytes)
		assets = append(assets, &servedAsset{
			tag:         rel.TagName,
			archiveName: archiveAsset.Name,
			archive:     archiveBytes,
			checksums:   hex.EncodeToString(sum[:]) + "  " + archiveAsset.Name + "\n",
		})
		_ = checksumsAsset
	}

	mux := http.NewServeMux()
	releaseCopies := make([]release, len(releases))
	copy(releaseCopies, releases)
	server := httptest.NewServer(mux)
	for i := range releaseCopies {
		for j := range releaseCopies[i].Assets {
			name := releaseCopies[i].Assets[j].Name
			releaseCopies[i].Assets[j].BrowserDownloadURL = server.URL + "/download/" + name
		}
		releaseCopies[i].HTMLURL = server.URL + "/releases/tag/" + releaseCopies[i].TagName
	}

	mux.HandleFunc("/repos/"+releaseOwner+"/"+releaseRepo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		for _, rel := range releaseCopies {
			if !rel.Prerelease {
				_ = json.NewEncoder(w).Encode(rel)
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/repos/"+releaseOwner+"/"+releaseRepo+"/releases", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(releaseCopies)
	})
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/download/")
		for _, asset := range assets {
			if name == asset.archiveName {
				_, _ = w.Write(asset.archive)
				return
			}
			if name == "brain_"+asset.tag+"_checksums.txt" {
				_, _ = w.Write([]byte(asset.checksums))
				return
			}
		}
		http.NotFound(w, r)
	})

	return server, assets
}

func makeRelease(tag string, prerelease bool, goos, goarch string, archiveBinary []byte) release {
	archiveName := archiveAssetName(tag, goos, goarch)
	return release{
		TagName:    tag,
		Prerelease: prerelease,
		Assets: []asset{
			{Name: archiveName, BrowserDownloadURL: string(archiveBinary)},
			{Name: "brain_" + tag + "_checksums.txt"},
		},
	}
}

func mustReleaseArchive(t *testing.T, goos string, binary []byte) []byte {
	t.Helper()
	if goos == "windows" {
		return mustZipArchive(t, binary)
	}
	return mustArchive(t, binary)
}

func mustArchive(t *testing.T, binary []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	header := &tar.Header{
		Name: "brain",
		Mode: 0o755,
		Size: int64(len(binary)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func mustZipArchive(t *testing.T, binary []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	file, err := zw.CreateHeader(&zip.FileHeader{
		Name:   "brain.exe",
		Method: zip.Deflate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func setBuildInfo(version string) func() {
	oldVersion := buildinfo.Version
	oldCommit := buildinfo.Commit
	oldDate := buildinfo.Date
	buildinfo.Version = version
	buildinfo.Commit = "deadbeef"
	buildinfo.Date = "2026-04-10T00:00:00Z"
	return func() {
		buildinfo.Version = oldVersion
		buildinfo.Commit = oldCommit
		buildinfo.Date = oldDate
	}
}
