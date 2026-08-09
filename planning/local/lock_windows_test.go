//go:build windows

package local

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestMutationLockContentionWindows(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "existing", err: os.ErrExist},
		{name: "access denied", err: syscall.ERROR_ACCESS_DENIED},
		{name: "sharing violation", err: windowsErrorSharingViolation},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := &os.PathError{Op: "open", Path: "lock", Err: test.err}
			if !isMutationLockContention(err) {
				t.Fatalf("expected %v to be classified as contention", test.err)
			}
		})
	}
	if isMutationLockContention(errors.New("unexpected")) {
		t.Fatal("expected an unrelated error not to be classified as contention")
	}
}
