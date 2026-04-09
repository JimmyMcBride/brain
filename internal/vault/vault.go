package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"brain/internal/config"
)

var paraDirs = []string{
	"Projects",
	"Areas",
	"Resources",
	"Archives",
}

type Service struct {
	Root     string
	DataPath string
}

func New(cfg *config.Config) *Service {
	return &Service{
		Root:     filepath.Clean(cfg.VaultPath),
		DataPath: filepath.Clean(cfg.DataPath),
	}
}

func (s *Service) Initialize() error {
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return fmt.Errorf("create vault root: %w", err)
	}
	for _, dir := range paraDirs {
		if err := os.MkdirAll(filepath.Join(s.Root, dir), 0o755); err != nil {
			return fmt.Errorf("create PARA dir %s: %w", dir, err)
		}
	}
	return nil
}

func (s *Service) Validate() error {
	info, err := os.Stat(s.Root)
	if err != nil {
		return fmt.Errorf("vault path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("vault path is not a directory: %s", s.Root)
	}
	for _, dir := range paraDirs {
		info, statErr := os.Stat(filepath.Join(s.Root, dir))
		if statErr != nil {
			return fmt.Errorf("missing PARA directory %s: %w", dir, statErr)
		}
		if !info.IsDir() {
			return fmt.Errorf("PARA path is not a directory: %s", dir)
		}
	}
	return nil
}

func (s *Service) Abs(rel string) string {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(s.Root, filepath.Clean(rel))
}

func (s *Service) Rel(path string) (string, error) {
	abs := s.Abs(path)
	rel, err := filepath.Rel(s.Root, abs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes vault: %s", path)
	}
	return filepath.ToSlash(rel), nil
}

func (s *Service) ResolveMarkdown(path string) (string, string, error) {
	if path == "" {
		return "", "", errors.New("empty note path")
	}
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}
	abs := s.Abs(path)
	rel, err := s.Rel(abs)
	if err != nil {
		return "", "", err
	}
	return abs, rel, nil
}

func (s *Service) WalkMarkdownFiles() ([]string, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	var files []string
	err := filepath.WalkDir(s.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") && path != s.Root {
				return filepath.SkipDir
			}
			if filepath.Clean(path) == filepath.Clean(s.DataPath) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(name), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk vault: %w", err)
	}
	return files, nil
}

func PARASections() []string {
	out := make([]string, len(paraDirs))
	copy(out, paraDirs)
	return out
}
