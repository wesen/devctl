package cmds

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-go-golems/devctl/pkg/config"
	"github.com/go-go-golems/devctl/pkg/runtime"
	glazedlayers "github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const repoLayerSlug = "repo"

type RepoSettings struct {
	RepoRoot string `glazed.parameter:"repo-root"`
	Config   string `glazed.parameter:"config"`
	Strict   bool   `glazed.parameter:"strict"`
	DryRun   bool   `glazed.parameter:"dry-run"`
	Timeout  string `glazed.parameter:"timeout"` // duration string, e.g. "30s"
}

type RepoContext struct {
	RepoRoot   string
	ConfigPath string
	Cwd        string
	Strict     bool
	DryRun     bool
	Timeout    time.Duration
}

func (rc RepoContext) RequestMeta() runtime.RequestMeta {
	return runtime.RequestMeta{
		RepoRoot: rc.RepoRoot,
		Cwd:      rc.Cwd,
		DryRun:   rc.DryRun,
	}
}

func repoContextFromSettings(settings RepoSettings, cwd string) (RepoContext, error) {
	repoRoot := settings.RepoRoot
	if repoRoot == "" {
		repoRoot = cwd
	}
	repoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return RepoContext{}, err
	}

	cfgPath := settings.Config
	if cfgPath == "" {
		cfgPath = config.DefaultPath(repoRoot)
	} else if !filepath.IsAbs(cfgPath) {
		cfgPath = filepath.Join(repoRoot, cfgPath)
	}

	timeoutStr := settings.Timeout
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return RepoContext{}, errors.Wrap(err, "parse --timeout (expected duration like 30s)")
	}
	if timeout <= 0 {
		return RepoContext{}, errors.New("timeout must be > 0")
	}

	return RepoContext{
		RepoRoot:   repoRoot,
		ConfigPath: cfgPath,
		Cwd:        cwd,
		Strict:     settings.Strict,
		DryRun:     settings.DryRun,
		Timeout:    timeout,
	}, nil
}

func RepoContextFromParsedLayers(parsedLayers *glazedlayers.ParsedLayers) (RepoContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return RepoContext{}, err
	}
	settings := RepoSettings{}
	if err := parsedLayers.InitializeStruct(repoLayerSlug, &settings); err != nil {
		return RepoContext{}, err
	}
	return repoContextFromSettings(settings, cwd)
}

func RepoContextFromCobra(cmd *cobra.Command) (RepoContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return RepoContext{}, err
	}

	repoRoot, err := cmd.Flags().GetString("repo-root")
	if err != nil {
		return RepoContext{}, err
	}
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return RepoContext{}, err
	}
	strict, err := cmd.Flags().GetBool("strict")
	if err != nil {
		return RepoContext{}, err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return RepoContext{}, err
	}
	timeoutStr, err := cmd.Flags().GetString("timeout")
	if err != nil {
		return RepoContext{}, err
	}

	return repoContextFromSettings(RepoSettings{
		RepoRoot: repoRoot,
		Config:   cfgPath,
		Strict:   strict,
		DryRun:   dryRun,
		Timeout:  timeoutStr,
	}, cwd)
}

var (
	repoLayerOnce sync.Once
	repoLayerInst *glazedlayers.ParameterLayerImpl
	repoLayerErr  error
)

func getRepoLayer() (*glazedlayers.ParameterLayerImpl, error) {
	repoLayerOnce.Do(func() {
		layer, err := glazedlayers.NewParameterLayer(repoLayerSlug, "Repository")
		if err != nil {
			repoLayerErr = err
			return
		}
		layer.Description = "Repository context shared by most devctl commands"
		layer.AddFlags(
			parameters.NewParameterDefinition(
				"repo-root",
				parameters.ParameterTypeString,
				parameters.WithDefault(""),
				parameters.WithHelp("Repository root (defaults to current directory)"),
			),
			parameters.NewParameterDefinition(
				"config",
				parameters.ParameterTypeString,
				parameters.WithDefault(""),
				parameters.WithHelp("Path to config file (defaults to .devctl.yaml under repo-root)"),
			),
			parameters.NewParameterDefinition(
				"strict",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Treat merge collisions as errors"),
			),
			parameters.NewParameterDefinition(
				"dry-run",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Do not perform destructive side effects (best-effort)"),
			),
			parameters.NewParameterDefinition(
				"timeout",
				parameters.ParameterTypeString,
				parameters.WithDefault("30s"),
				parameters.WithHelp("Default timeout for plugin operations (duration like 30s)"),
			),
		)

		repoLayerInst = layer
	})
	return repoLayerInst, repoLayerErr
}

func AddRepoFlags(cmd *cobra.Command) {
	layer, err := getRepoLayer()
	cobra.CheckErr(err)
	cobra.CheckErr(layer.AddLayerToCobraCommand(cmd))
}

type rootOptions struct {
	RepoRoot string
	Config   string
	Strict   bool
	DryRun   bool
	Timeout  time.Duration
}

func getRootOptions(cmd *cobra.Command) (rootOptions, error) {
	rc, err := RepoContextFromCobra(cmd)
	if err != nil {
		return rootOptions{}, err
	}
	return rootOptions{
		RepoRoot: rc.RepoRoot,
		Config:   rc.ConfigPath,
		Strict:   rc.Strict,
		DryRun:   rc.DryRun,
		Timeout:  rc.Timeout,
	}, nil
}

func requestMetaFromRootOptions(opts rootOptions) (runtime.RequestMeta, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return runtime.RequestMeta{}, err
	}
	return runtime.RequestMeta{
		RepoRoot: opts.RepoRoot,
		Cwd:      cwd,
		DryRun:   opts.DryRun,
	}, nil
}
