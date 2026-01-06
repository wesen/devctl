package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntime_HandshakeAndCall(t *testing.T) {
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	plugin := filepath.Join(repoRoot, "..", "..", "testdata", "plugins", "ok-python", "plugin.py")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	f := NewFactory(FactoryOptions{HandshakeTimeout: 2 * time.Second, ShutdownTimeout: 2 * time.Second})
	c, err := f.Start(ctx, PluginSpec{
		ID:      "t",
		Path:    "python3",
		Args:    []string{plugin},
		WorkDir: repoRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	var out struct {
		Pong bool `json:"pong"`
	}
	if err := c.Call(ctx, "ping", map[string]any{"message": "hi"}, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Pong {
		t.Fatalf("expected pong=true")
	}
}
