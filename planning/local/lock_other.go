//go:build !windows

package local

import "os"

func isMutationLockContention(err error) bool {
	return os.IsExist(err)
}
