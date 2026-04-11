package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Service struct {
	Root string
}

func New(root string) *Service {
	return &Service{Root: filepath.Clean(root)}
}

func (s *Service) Initialize() error {
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return fmt.Errorf("create project root: %w", err)
	}
	for _, dir := range []string{
		".brain",
		".brain/context",
		".brain/brainstorms",
		".brain/planning",
		".brain/resources/captures",
		".brain/resources/changes",
		".brain/resources/references",
		".brain/sessions",
		".brain/state",
		"docs",
	} {
		if err := os.MkdirAll(filepath.Join(s.Root, dir), 0o755); err != nil {
			return fmt.Errorf("create project dir %s: %w", dir, err)
		}
	}
	return nil
}

func (s *Service) Validate() error {
	info, err := os.Stat(s.Root)
	if err != nil {
		return fmt.Errorf("project path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", s.Root)
	}
	for _, dir := range []string{".brain", ".brain/state"} {
		info, statErr := os.Stat(filepath.Join(s.Root, dir))
		if statErr != nil {
			return fmt.Errorf("missing brain directory %s: %w", dir, statErr)
		}
		if !info.IsDir() {
			return fmt.Errorf("brain path is not a directory: %s", dir)
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
		return "", fmt.Errorf("path escapes workspace: %s", path)
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
	var roots []string
	for _, rel := range []string{"AGENTS.md", "docs", ".brain"} {
		roots = append(roots, filepath.Join(s.Root, filepath.FromSlash(rel)))
	}

	var files []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if strings.EqualFold(filepath.Ext(root), ".md") {
				files = append(files, root)
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, relErr := filepath.Rel(s.Root, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			name := d.Name()
			if d.IsDir() {
				switch rel {
				case ".brain/state", ".brain/sessions":
					return filepath.SkipDir
				}
				if strings.HasPrefix(name, ".git") {
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
			return nil, fmt.Errorf("walk project notes: %w", err)
		}
	}
	return files, nil
}
