package config

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const DefaultConfigFilename = ".devctl.yaml"

type File struct {
	Plugins    []Plugin `yaml:"plugins"`
	Strictness string   `yaml:"strictness,omitempty"` // "warn" | "error"
}

type Plugin struct {
	ID       string            `yaml:"id"`
	Path     string            `yaml:"path"`
	Args     []string          `yaml:"args,omitempty"`
	Priority int               `yaml:"priority,omitempty"`
	WorkDir  string            `yaml:"workdir,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
}

func DefaultPath(repoRoot string) string {
	return filepath.Join(repoRoot, DefaultConfigFilename)
}

func LoadFromFile(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read config")
	}
	var cfg File
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, errors.Wrap(err, "parse config yaml")
	}
	return &cfg, nil
}

func LoadOptional(path string) (*File, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{}, nil
		}
		return nil, errors.Wrap(err, "stat config")
	}
	return LoadFromFile(path)
}
