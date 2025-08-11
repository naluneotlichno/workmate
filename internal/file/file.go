package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const appDirPerm os.FileMode = 0o750

// EnsureDir creates the directory if it does not exist.
func EnsureDir(dirPath string) error {
	if dirPath == "" {
		return errors.New("empty dir path")
	}
	if err := os.MkdirAll(dirPath, appDirPerm); err != nil { //nolint:gosec // app-owned data dir
		return fmt.Errorf("ensure dir: %w", err)
	}
	return nil
}

// WriteJSONAtomic marshals the value and atomically writes it to filename.
// The write is performed via a temporary file in the same directory
// followed by a rename to ensure atomicity on most filesystems.
func WriteJSONAtomic(filename string, v any) error {
	if filename == "" {
		return errors.New("empty filename")
	}

	dir := filepath.Dir(filename)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tempFile.Name()

	jsonEncoder := json.NewEncoder(tempFile)
	jsonEncoder.SetEscapeHTML(true)
	if err := jsonEncoder.Encode(v); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("encode json: %w", err)
	}

	// ensure data hits disk
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp: %w", err)
	}

	// remove existing file to avoid permission issues on Windows
	if _, err := os.Stat(filename); err == nil {
		// ignore error; if remove fails, rename may still succeed on POSIX
		_ = os.Remove(filename)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}

// CopyAtomic writes data provided by the reader to the destination file atomically.
func CopyAtomic(filename string, reader io.Reader) error {
	dir := filepath.Dir(filename)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tempFile.Name()
	if _, err := io.Copy(tempFile, reader); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("copy to temp: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp: %w", err)
	}
	if _, err := os.Stat(filename); err == nil {
		_ = os.Remove(filename)
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}
