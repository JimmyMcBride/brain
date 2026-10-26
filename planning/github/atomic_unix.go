//go:build !windows

package github

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
