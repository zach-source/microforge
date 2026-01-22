package util

import (
	"os"
	"path/filepath"
)

func EnsureDir(path string) error { return os.MkdirAll(path, 0o755) }

func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil { return err }
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil { return err }
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil { tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	if err := os.Chmod(tmpName, perm); err != nil { return err }
	return os.Rename(tmpName, path)
}
