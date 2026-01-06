package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/devctl/pkg/config"
	"github.com/go-go-golems/devctl/pkg/runtime"
	"github.com/pkg/errors"
)

type Options struct {
	RepoRoot string
}

func Discover(cfg *config.File, opts Options) ([]runtime.PluginSpec, error) {
	if opts.RepoRoot == "" {
		return nil, errors.New("missing RepoRoot")
	}
	if cfg == nil {
		cfg = &config.File{}
	}

	seen := map[string]struct{}{}
	out := make([]runtime.PluginSpec, 0, len(cfg.Plugins))
	for _, p := range cfg.Plugins {
		if p.ID == "" {
			return nil, errors.New("plugin missing id")
		}
		if _, ok := seen[p.ID]; ok {
			return nil, errors.Errorf("duplicate plugin id %q", p.ID)
		}
		seen[p.ID] = struct{}{}
		if p.Path == "" {
			return nil, errors.Errorf("plugin %q missing path", p.ID)
		}

		spec, err := toSpec(opts.RepoRoot, p)
		if err != nil {
			return nil, err
		}
		out = append(out, spec)
	}

	autoSpecs, err := scanPluginsDir(opts.RepoRoot, seen)
	if err != nil {
		return nil, err
	}
	out = append(out, autoSpecs...)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].ID < out[j].ID
	})

	return out, nil
}

func toSpec(repoRoot string, p config.Plugin) (runtime.PluginSpec, error) {
	workDir := p.WorkDir
	if workDir == "" {
		workDir = repoRoot
	} else if !filepath.IsAbs(workDir) {
		workDir = filepath.Join(repoRoot, workDir)
	}

	path := p.Path
	if filepath.IsAbs(path) {
		// ok
	} else if hasPathSep(path) {
		path = filepath.Join(repoRoot, path)
	}

	if hasPathSep(path) {
		if _, err := os.Stat(path); err != nil {
			return runtime.PluginSpec{}, errors.Wrapf(err, "plugin %q path not found: %s", p.ID, path)
		}
	}

	return runtime.PluginSpec{
		ID:       p.ID,
		Path:     path,
		Args:     p.Args,
		Env:      p.Env,
		WorkDir:  workDir,
		Priority: p.Priority,
	}, nil
}

func hasPathSep(s string) bool {
	for _, c := range s {
		if c == '/' || c == '\\' {
			return true
		}
	}
	return false
}

func scanPluginsDir(repoRoot string, seen map[string]struct{}) ([]runtime.PluginSpec, error) {
	pluginsDir := filepath.Join(repoRoot, "plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "read plugins dir")
	}

	var out []runtime.PluginSpec
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "devctl-") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			return nil, errors.Wrap(err, "stat plugin entry")
		}
		if info.Mode()&0o111 == 0 {
			continue
		}

		id := strings.TrimPrefix(name, "devctl-")
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		path := filepath.Join(pluginsDir, name)
		out = append(out, runtime.PluginSpec{
			ID:       id,
			Path:     path,
			WorkDir:  repoRoot,
			Env:      map[string]string{},
			Priority: 1000,
		})
	}

	return out, nil
}
