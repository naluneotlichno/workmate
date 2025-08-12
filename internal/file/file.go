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

func EnsureDir(dirPath string) error {
	if dirPath == "" {
		return errors.New("empty dir path")
	}
	if err := os.MkdirAll(dirPath, appDirPerm); err != nil {
		return fmt.Errorf("ensure dir: %w", err)
	}
	return nil
}

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
	// Pretty-print JSON for human-friendly inspection on disk
	jsonEncoder.SetIndent("", "  ")
	if err := jsonEncoder.Encode(v); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("encode json: %w", err)
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
