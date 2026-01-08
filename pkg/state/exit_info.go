package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

type ExitInfo struct {
	Service   string    `json:"service"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	ExitedAt  time.Time `json:"exited_at"`

	ExitCode *int   `json:"exit_code,omitempty"`
	Signal   string `json:"signal,omitempty"`
	Error    string `json:"error,omitempty"`

	StderrTail []string `json:"stderr_tail,omitempty"`
	StdoutTail []string `json:"stdout_tail,omitempty"`
}

func WriteExitInfo(path string, info ExitInfo) error {
	if path == "" {
		return errors.New("missing path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrap(err, "mkdir exit info dir")
	}
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal exit info")
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return errors.Wrap(err, "write exit info")
	}
	return nil
}

func ReadExitInfo(path string) (*ExitInfo, error) {
	if path == "" {
		return nil, errors.New("missing path")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read exit info")
	}
	var info ExitInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return nil, errors.Wrap(err, "unmarshal exit info")
	}
	return &info, nil
}
