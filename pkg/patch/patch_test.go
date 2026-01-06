package patch

import "testing"

func TestApply_SetNested(t *testing.T) {
	cfg := Config{}
	out, err := Apply(cfg, ConfigPatch{
		Set: map[string]any{
			"services.backend.port": 8083,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	services := out["services"].(map[string]any)
	backend := services["backend"].(map[string]any)
	if backend["port"].(int) != 8083 {
		t.Fatalf("expected 8083, got %#v", backend["port"])
	}
}

func TestApply_UnsetMissing(t *testing.T) {
	cfg := Config{"a": map[string]any{"b": 1}}
	_, err := Apply(cfg, ConfigPatch{Unset: []string{"a.c"}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMerge_LaterWins(t *testing.T) {
	out := Merge(
		ConfigPatch{Set: map[string]any{"a.b": 1}},
		ConfigPatch{Set: map[string]any{"a.b": 2}},
	)
	if out.Set["a.b"].(int) != 2 {
		t.Fatalf("expected 2, got %#v", out.Set["a.b"])
	}
}
