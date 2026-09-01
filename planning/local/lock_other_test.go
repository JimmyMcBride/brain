//go:build !windows

package local

import (
	"errors"
	"os"
	"testing"
)

func TestMutationLockContentionOther(t *testing.T) {
	if !isMutationLockContention(&os.PathError{Op: "open", Path: "lock", Err: os.ErrExist}) {
		t.Fatal("expected an existing lock to be classified as contention")
	}
	if isMutationLockContention(errors.New("permission denied")) {
		t.Fatal("expected an unrelated error not to be classified as contention")
	}
}
