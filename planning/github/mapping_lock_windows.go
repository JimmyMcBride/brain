//go:build windows

package github

import (
	"errors"
	"os"
	"syscall"
)

const windowsErrorSharingViolation syscall.Errno = 32

func isMappingLockContention(err error) bool {
	return os.IsExist(err) ||
		errors.Is(err, syscall.ERROR_ACCESS_DENIED) ||
		errors.Is(err, windowsErrorSharingViolation)
}
