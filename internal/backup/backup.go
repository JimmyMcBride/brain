package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manager struct {
	Dir string
}

func New(dir string) *Manager {
	return &Manager{Dir: dir}
}

func (m *Manager) Create(source string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", fmt.Errorf("stat backup source: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("cannot backup directory: %s", source)
	}
	if err := os.MkdirAll(m.Dir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z07") + "__" + sanitize(source) + ".bak"
	target := filepath.Join(m.Dir, name)
	return target, copyFile(source, target)
}

func (m *Manager) Restore(backupPath, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create restore parent: %w", err)
	}
	return copyFile(backupPath, destination)
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	return out.Close()
}

func sanitize(path string) string {
	replacer := strings.NewReplacer("/", "__", "\\", "__", ":", "_")
	return replacer.Replace(strings.Trim(path, "/"))
}
