package models

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/devctl/pkg/tui"
)

type DashboardModel struct {
	last *tui.StateSnapshot
}

func NewDashboardModel() DashboardModel { return DashboardModel{} }

func (m DashboardModel) WithSnapshot(s tui.StateSnapshot) DashboardModel {
	m.last = &s
	return m
}

func (m DashboardModel) View() string {
	if m.last == nil {
		return "Loading state...\n"
	}

	s := m.last
	if !s.Exists {
		return "System: Stopped (no state)\n"
	}
	if s.Error != "" {
		return fmt.Sprintf("System: Error (state)\n\n%s\n", s.Error)
	}
	if s.State == nil {
		return "System: Unknown (state missing)\n"
	}

	var b strings.Builder
	b.WriteString("System: Running\n\n")
	b.WriteString(fmt.Sprintf("RepoRoot: %s\n", s.RepoRoot))
	b.WriteString(fmt.Sprintf("Started: %s\n\n", s.State.CreatedAt.Format("2006-01-02 15:04:05")))

	b.WriteString("Services:\n")
	for _, svc := range s.State.Services {
		a := false
		if s.Alive != nil {
			a = s.Alive[svc.Name]
		}
		status := "dead"
		if a {
			status = "alive"
		}
		b.WriteString(fmt.Sprintf("- %s pid=%d %s\n", svc.Name, svc.PID, status))
	}

	return b.String()
}
