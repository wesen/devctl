package tui

import (
	"time"
)

type EventLogEntry struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}
