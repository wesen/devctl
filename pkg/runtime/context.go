package runtime

import "context"

type requestContextKey string

const (
	requestContextRepoRootKey requestContextKey = "devctl.repo_root"
	requestContextCwdKey      requestContextKey = "devctl.cwd"
	requestContextDryRunKey   requestContextKey = "devctl.dry_run"
)

func WithRepoRoot(ctx context.Context, repoRoot string) context.Context {
	if repoRoot == "" {
		return ctx
	}
	return context.WithValue(ctx, requestContextRepoRootKey, repoRoot)
}

func WithCwd(ctx context.Context, cwd string) context.Context {
	if cwd == "" {
		return ctx
	}
	return context.WithValue(ctx, requestContextCwdKey, cwd)
}

func WithDryRun(ctx context.Context, dryRun bool) context.Context {
	if !dryRun {
		return ctx
	}
	return context.WithValue(ctx, requestContextDryRunKey, true)
}

func repoRootFromContext(ctx context.Context) string {
	v := ctx.Value(requestContextRepoRootKey)
	s, _ := v.(string)
	return s
}

func cwdFromContext(ctx context.Context) string {
	v := ctx.Value(requestContextCwdKey)
	s, _ := v.(string)
	return s
}

func dryRunFromContext(ctx context.Context) bool {
	v := ctx.Value(requestContextDryRunKey)
	b, _ := v.(bool)
	return b
}
