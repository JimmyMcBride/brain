package conformance

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestReleaseWorkflowPublishesRequiredPlatformArchivesAndChecksums(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".github", "workflows", "release.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs struct {
			Build struct {
				Strategy struct {
					Matrix struct {
						Include []struct {
							OS   string `yaml:"os"`
							Arch string `yaml:"arch"`
						} `yaml:"include"`
					} `yaml:"matrix"`
				} `yaml:"strategy"`
			} `yaml:"build"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(raw, &workflow); err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(workflow.Jobs.Build.Strategy.Matrix.Include))
	for _, target := range workflow.Jobs.Build.Strategy.Matrix.Include {
		got = append(got, target.OS+"/"+target.Arch)
	}
	want := []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("release targets=%v want=%v", got, want)
	}

	contract := string(raw)
	for _, required := range []string{
		"workflow_call:",
		"TAG_INPUT=\"${{ inputs.tag }}\"",
		"SHA_INPUT=\"${{ inputs.sha }}\"",
		"ref: ${{ needs.resolve.outputs.sha }}",
		"brain_${TAG}_${GOOS}_${GOARCH}.${ARCHIVE_EXT}",
		"sha256sum brain_${{ needs.resolve.outputs.tag }}_*.* > \"brain_${{ needs.resolve.outputs.tag }}_checksums.txt\"",
		"dist/brain_${{ needs.resolve.outputs.tag }}_*.tar.gz",
		"dist/brain_${{ needs.resolve.outputs.tag }}_*.zip",
		"dist/brain_${{ needs.resolve.outputs.tag }}_checksums.txt",
	} {
		if !strings.Contains(contract, required) {
			t.Errorf("release workflow is missing contract fragment %q", required)
		}
	}
}
