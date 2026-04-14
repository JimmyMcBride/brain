package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

type skillBundle struct {
	FS   fs.FS
	Hash string
}

var (
	bundleMu         sync.RWMutex
	registeredBundle fs.FS
)

func RegisterBundle(bundle fs.FS) {
	bundleMu.Lock()
	registeredBundle = bundle
	bundleMu.Unlock()
}

func loadBundle() (skillBundle, error) {
	bundleFS, err := currentBundleFS()
	if err != nil {
		return skillBundle{}, err
	}
	hash, err := bundleHash(bundleFS)
	if err != nil {
		return skillBundle{}, err
	}
	return skillBundle{FS: bundleFS, Hash: hash}, nil
}

func currentBundleFS() (fs.FS, error) {
	bundleMu.RLock()
	bundle := registeredBundle
	bundleMu.RUnlock()
	if bundle != nil {
		return bundle, nil
	}
	return sourceTreeBundle()
}

func sourceTreeBundle() (fs.FS, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("resolve bundled brain skill source")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	source := filepath.Join(root, "skills", "brain")
	if err := validateSkillSource(source); err != nil {
		return nil, err
	}
	return os.DirFS(source), nil
}

func validateSkillSource(source string) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("skill source %s: %w", source, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skill source is not a directory: %s", source)
	}
	if _, err := os.Stat(filepath.Join(source, "SKILL.md")); err != nil {
		return fmt.Errorf("skill source missing SKILL.md: %w", err)
	}
	return nil
}

func bundleHash(bundle fs.FS) (string, error) {
	files, err := bundleFiles(bundle)
	if err != nil {
		return "", err
	}
	sum := sha256.New()
	for _, path := range files {
		data, err := fs.ReadFile(bundle, path)
		if err != nil {
			return "", fmt.Errorf("read bundled skill file %s: %w", path, err)
		}
		sum.Write([]byte(path))
		sum.Write([]byte{0})
		sum.Write(data)
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func bundleFiles(bundle fs.FS) ([]string, error) {
	var files []string
	if err := fs.WalkDir(bundle, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk bundled brain skill: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func copyBundle(bundle skillBundle, target string) error {
	return fs.WalkDir(bundle.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dest := filepath.Join(target, path)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, err := fs.ReadFile(bundle.FS, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o644)
	})
}
