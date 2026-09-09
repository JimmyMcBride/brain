//go:build !windows

package github

import (
	"errors"
	"os"
	"testing"
)

func TestMappingLockContentionOther(t *testing.T) {
	if !isMappingLockContention(&os.PathError{Op: "open", Path: "lock", Err: os.ErrExist}) {
		t.Fatal("expected an existing lock to be classified as contention")
	}
	if isMappingLockContention(errors.New("permission denied")) {
		t.Fatal("expected an unrelated error not to be classified as contention")
	}
}
