package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultAndNormalize(t *testing.T) {
	cfg := Default()
	if cfg.Port == 0 || cfg.DataDir == "" || cfg.MaxConcurrentTasks < 1 {
		t.Fatalf("default config invalid: %+v", cfg)
	}

	got := normalizeExtensions([]string{"PDF", ".jpeg", "pdf", "  .JPG"})

	has := func(slice []string, s string) bool {
		for _, v := range slice {
			if v == s {
				return true
			}
		}
		return false
	}
	if !has(got, ".pdf") || !has(got, ".jpeg") || !has(got, ".jpg") {
		t.Fatalf("expected normalized set to contain .pdf,.jpeg,.jpg got %v", got)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	_, err := Load("not_exists.yml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
}

func TestLoadReadsAndValidates(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "cfg.yml")
	content := []byte("port: 9090\ndata_dir: testdata\nallowed_extensions: [pdf, .jpeg]\nmax_concurrent_tasks: 2\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Port != 9090 || cfg.DataDir != "testdata" || cfg.MaxConcurrentTasks != 2 {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}

	if len(cfg.AllowedExtensions) == 0 || cfg.AllowedExtensions[0][0] != '.' {
		t.Fatalf("extensions not normalized: %v", cfg.AllowedExtensions)
	}
}

func TestLoadRejectsInvalidConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "cfg.yml")
	content := []byte("max_concurrent_tasks: 0\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatalf("expected error for invalid concurrency")
	}
}
