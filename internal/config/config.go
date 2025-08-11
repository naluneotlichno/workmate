package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultPort               = 8080
	defaultDataDir            = "data"
	defaultMaxConcurrentTasks = 3
)

// Config describes runtime configuration for the service.
type Config struct {
	Port               int      `yaml:"port"`
	DataDir            string   `yaml:"data_dir"`
	AllowedExtensions  []string `yaml:"allowed_extensions"`
	MaxConcurrentTasks int      `yaml:"max_concurrent_tasks"`
}

// Default returns sane defaults compliant with TZ.md
func Default() Config {
	return Config{
		Port:               defaultPort,
		DataDir:            defaultDataDir,
		AllowedExtensions:  []string{".pdf", ".jpeg"},
		MaxConcurrentTasks: defaultMaxConcurrentTasks,
	}
}

// Load reads YAML config from the provided path. If the file does not exist
// or is empty, defaults are returned with no error.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, errors.New("empty config path")
	}
	fileData, err := os.ReadFile(path) //nolint:gosec // config path is controlled by deployment
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if len(fileData) == 0 {
		return cfg, nil
	}
	if err := yaml.Unmarshal(fileData, &cfg); err != nil {
		return cfg, fmt.Errorf("parse yaml: %w", err)
	}
	// basic normalization
	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}
	if cfg.DataDir == "" {
		cfg.DataDir = defaultDataDir
	}
	// validate concurrency explicitly: values < 1 are not allowed
	if cfg.MaxConcurrentTasks < 1 {
		return cfg, fmt.Errorf("invalid max_concurrent_tasks: %d (must be >= 1)", cfg.MaxConcurrentTasks)
	}
	cfg.AllowedExtensions = normalizeExtensions(cfg.AllowedExtensions)
	return cfg, nil
}

func normalizeExtensions(in []string) []string {
	if len(in) == 0 {
		return []string{".pdf", ".jpeg"}
	}
	seen := make(map[string]struct{}, len(in))
	normalized := make([]string, 0, len(in))
	for _, ext := range in {
		e := strings.ToLower(strings.TrimSpace(ext))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		normalized = append(normalized, e)
	}
	return normalized
}
