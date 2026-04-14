package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"brain/internal/buildinfo"
)

const (
	manifestFileName      = ".brain-skill-manifest.json"
	manifestSchemaVersion = 1
)

type Manifest struct {
	SchemaVersion int    `json:"schema_version"`
	BrainVersion  string `json:"brain_version"`
	BrainCommit   string `json:"brain_commit"`
	BundleHash    string `json:"bundle_hash"`
	InstalledAt   string `json:"installed_at"`
	Agent         string `json:"agent"`
	Scope         string `json:"scope"`
}

func manifestPath(target string) string {
	return filepath.Join(target, manifestFileName)
}

func readManifest(target string) (*Manifest, error) {
	raw, err := os.ReadFile(manifestPath(target))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func writeManifest(target, agent, scope, bundleHash string) error {
	info := buildinfo.Current()
	manifest := Manifest{
		SchemaVersion: manifestSchemaVersion,
		BrainVersion:  info.Version,
		BrainCommit:   info.Commit,
		BundleHash:    bundleHash,
		InstalledAt:   time.Now().UTC().Format(time.RFC3339),
		Agent:         agent,
		Scope:         scope,
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(manifestPath(target), raw, 0o644)
}
