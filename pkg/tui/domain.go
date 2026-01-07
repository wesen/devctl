package tui

import (
	"time"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

type EventLogEntry struct {
	At     time.Time `json:"at"`
	Source string    `json:"source,omitempty"`
	Level  LogLevel  `json:"level,omitempty"`
	Text   string    `json:"text"`
}
