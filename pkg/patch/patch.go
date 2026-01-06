package patch

import (
	"strings"

	"github.com/pkg/errors"
)

type Config = map[string]any

type ConfigPatch struct {
	Set   map[string]any `json:"set,omitempty"`
	Unset []string       `json:"unset,omitempty"`
}

func Apply(cfg Config, p ConfigPatch) (Config, error) {
	if cfg == nil {
		cfg = Config{}
	}
	for _, key := range p.Unset {
		if err := unsetDotted(cfg, key); err != nil {
			return nil, err
		}
	}
	for key, value := range p.Set {
		if err := setDotted(cfg, key, value); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func Merge(a, b ConfigPatch) ConfigPatch {
	out := ConfigPatch{
		Set:   map[string]any{},
		Unset: []string{},
	}
	for k, v := range a.Set {
		out.Set[k] = v
	}
	for k, v := range b.Set {
		out.Set[k] = v
	}
	seen := map[string]struct{}{}
	for _, k := range append(append([]string{}, a.Unset...), b.Unset...) {
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out.Unset = append(out.Unset, k)
	}
	return out
}

func setDotted(cfg Config, dotted string, value any) error {
	parts := splitDotted(dotted)
	if len(parts) == 0 {
		return errors.Errorf("empty dotted key")
	}

	current := cfg
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, ok := current[part]
		if !ok {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		asMap, ok := next.(map[string]any)
		if !ok {
			return errors.Errorf("cannot set %q: path segment %q is not an object", dotted, part)
		}
		current = asMap
	}

	current[parts[len(parts)-1]] = value
	return nil
}

func unsetDotted(cfg Config, dotted string) error {
	parts := splitDotted(dotted)
	if len(parts) == 0 {
		return errors.Errorf("empty dotted key")
	}

	current := cfg
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, ok := current[part]
		if !ok {
			return nil
		}
		asMap, ok := next.(map[string]any)
		if !ok {
			return errors.Errorf("cannot unset %q: path segment %q is not an object", dotted, part)
		}
		current = asMap
	}
	delete(current, parts[len(parts)-1])
	return nil
}

func splitDotted(dotted string) []string {
	raw := strings.Split(dotted, ".")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
