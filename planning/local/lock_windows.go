//go:build windows

package local

import (
	"errors"
	"os"
	"syscall"
)

const windowsErrorSharingViolation syscall.Errno = 32

func isMutationLockContention(err error) bool {
	return os.IsExist(err) ||
		errors.Is(err, syscall.ERROR_ACCESS_DENIED) ||
		errors.Is(err, windowsErrorSharingViolation)
}
