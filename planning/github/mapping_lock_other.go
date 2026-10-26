//go:build !windows

package github

import "os"

func isMappingLockContention(err error) bool {
	return os.IsExist(err)
}
