package modules

import (
	"errors"
	"io"
	"testing"
)

func TestWriteAllRejectsShortWrite(t *testing.T) {
	err := writeAll(shortWriter{}, []byte("complete"))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected io.ErrShortWrite, got %v", err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) {
	return len(data) - 1, nil
}
