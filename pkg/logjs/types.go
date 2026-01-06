package logjs

type Event struct {
	Timestamp  *string        `json:"timestamp,omitempty"`
	Level      string         `json:"level"`
	Message    string         `json:"message"`
	Fields     map[string]any `json:"fields"`
	Tags       []string       `json:"tags"`
	Source     string         `json:"source"`
	Raw        string         `json:"raw"`
	LineNumber int64          `json:"lineNumber"`
}

type Stats struct {
	LinesProcessed int64
	EventsEmitted  int64
	LinesDropped   int64
	HookErrors     int64
	HookTimeouts   int64
}

type Options struct {
	HookTimeout string
}
