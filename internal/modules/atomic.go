package modules

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func atomicWriteFile(path string, data []byte, mode os.FileMode) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempPath := file.Name()
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()
	if err = file.Chmod(mode); err != nil {
		return fmt.Errorf("set temporary file mode: %w", err)
	}
	if err = writeAll(file, data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err = replaceFile(tempPath, path); err != nil {
		return fmt.Errorf("replace destination: %w", err)
	}
	if err = os.Chmod(path, mode); err != nil {
		return fmt.Errorf("set destination file mode: %w", err)
	}
	return nil
}

func writeAll(writer io.Writer, data []byte) error {
	written, err := writer.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}
