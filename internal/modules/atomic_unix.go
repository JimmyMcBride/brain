//go:build !windows

package modules

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
