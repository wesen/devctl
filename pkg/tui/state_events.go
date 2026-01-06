package tui

import (
	"time"

	"github.com/go-go-golems/devctl/pkg/state"
)

type ServiceExitObserved struct {
	Name   string    `json:"name"`
	PID    int       `json:"pid"`
	When   time.Time `json:"when"`
	Reason string    `json:"reason,omitempty"`
}

type StateSnapshot struct {
	RepoRoot string          `json:"repo_root"`
	At       time.Time       `json:"at"`
	Exists   bool            `json:"exists"`
	State    *state.State    `json:"state,omitempty"`
	Alive    map[string]bool `json:"alive,omitempty"`
	Error    string          `json:"error,omitempty"`
}
