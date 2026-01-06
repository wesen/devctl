package engine

type ServiceSpec struct {
	Name    string            `json:"name"`
	Cwd     string            `json:"cwd,omitempty"`
	Command []string          `json:"command"`
	Env     map[string]string `json:"env,omitempty"`
	Health  *HealthCheck      `json:"health,omitempty"`
}

type HealthCheck struct {
	Type      string `json:"type"` // "tcp"|"http"
	Address   string `json:"address,omitempty"`
	URL       string `json:"url,omitempty"`
	TimeoutMs int64  `json:"timeout_ms,omitempty"`
}

type LaunchPlan struct {
	Services []ServiceSpec `json:"services"`
}
