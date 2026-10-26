//go:build windows

package github

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestMappingLockContentionWindows(t *testing.T) {
	tests := []error{os.ErrExist, syscall.ERROR_ACCESS_DENIED, windowsErrorSharingViolation}
	for _, test := range tests {
		if !isMappingLockContention(&os.PathError{Op: "open", Path: "lock", Err: test}) {
			t.Fatalf("expected %v to be classified as contention", test)
		}
	}
	if isMappingLockContention(errors.New("unexpected")) {
		t.Fatal("expected an unrelated error not to be classified as contention")
	}
}
