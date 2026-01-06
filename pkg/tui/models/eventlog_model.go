package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/devctl/pkg/tui"
)

type EventLogModel struct {
	max     int
	entries []tui.EventLogEntry
}

func NewEventLogModel() EventLogModel {
	return EventLogModel{max: 200, entries: nil}
}

func (m EventLogModel) Append(e tui.EventLogEntry) EventLogModel {
	m.entries = append(m.entries, e)
	if m.max > 0 && len(m.entries) > m.max {
		m.entries = append([]tui.EventLogEntry{}, m.entries[len(m.entries)-m.max:]...)
	}
	return m
}

func (m EventLogModel) View() string {
	if len(m.entries) == 0 {
		return "No events yet.\n"
	}
	var b strings.Builder
	b.WriteString("Events:\n")
	for _, e := range m.entries {
		ts := e.At
		if ts.IsZero() {
			ts = time.Now()
		}
		b.WriteString(fmt.Sprintf("- %s %s\n", ts.Format("15:04:05"), e.Text))
	}
	return b.String()
}
