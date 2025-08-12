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
	defaultDataDir            = "storage/data"
	defaultMaxConcurrentTasks = 3
)

type Config struct {
	Port               int      `yaml:"port"`
	DataDir            string   `yaml:"data_dir"`
	AllowedExtensions  []string `yaml:"allowed_extensions"`
	MaxConcurrentTasks int      `yaml:"max_concurrent_tasks"`
}

func Default() Config {
	return Config{
		Port:               defaultPort,
		DataDir:            defaultDataDir,
		AllowedExtensions:  []string{".pdf", ".jpeg", ".jpg"},
		MaxConcurrentTasks: defaultMaxConcurrentTasks,
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, errors.New("empty config path")
	}
	fileData, err := os.ReadFile(path)
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

	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}
	if cfg.DataDir == "" {
		cfg.DataDir = defaultDataDir
	}

	if cfg.MaxConcurrentTasks < 1 {
		return cfg, fmt.Errorf("invalid max_concurrent_tasks: %d (must be >= 1)", cfg.MaxConcurrentTasks)
	}
	cfg.AllowedExtensions = normalizeExtensions(cfg.AllowedExtensions)
	return cfg, nil
}

func normalizeExtensions(in []string) []string {
	if len(in) == 0 {
		return []string{".pdf", ".jpeg", ".jpg"}
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
