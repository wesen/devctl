---
Title: 'Go Runner Architecture: NDJSON Plugin Protocol Runner'
Ticket: MO-005-IMPROVE-STARTDEV
Status: active
Topics:
    - dev-tooling
    - scripts
    - bash
    - dev-environment
DocType: design-doc
Intent: long-term
Owners:
    - team
RelatedFiles:
    - Path: moments/ttmp/2026/01/06/MO-005-IMPROVE-STARTDEV--improve-startdev-sh-script-analysis-and-enhancement/analysis/01-startdev-sh-complete-step-by-step-analysis.md
      Note: Behavioral reference for MVP parity with startdev.sh
    - Path: moments/ttmp/2026/01/06/MO-005-IMPROVE-STARTDEV--improve-startdev-sh-script-analysis-and-enhancement/design-doc/04-script-plugin-protocol-config-mutation-build-prepare-validate-launch-monitor-logs-commands.md
      Note: Protocol spec this Go architecture implements
    - Path: moments/ttmp/2026/01/06/MO-005-IMPROVE-STARTDEV--improve-startdev-sh-script-analysis-and-enhancement/tasks.md
      Note: Go-structured task breakdown aligned with this architecture
ExternalSources: []
Summary: Go architecture (packages, APIs, file layout) for implementing the generic NDJSON stdio plugin runner described in design-doc/04.
LastUpdated: 2026-01-06T14:20:16-05:00
WhatFor: Define the Go architecture for the generic runner/CLI and split work into implementable Go tasks.
WhenToUse: When implementing/reviewing the runner, or onboarding contributors to the protocol and code layout.
---


# Go Runner Architecture: NDJSON Plugin Protocol Runner

## Executive Summary

Build a Go CLI (working name: `devctl`) that runs a **script-first NDJSON stdio plugin protocol** (design-doc/04). The Go program owns reliability and UX: plugin discovery, protocol enforcement (stdout purity), request/response correlation, event streams, deterministic composition (patch merges), and plan-mode service supervision (start/stop/health/logs). Plugins stay easy to author (bash/python) and own repo-specific behavior.

## Problem Statement

`startdev.sh` is a practical dev orchestrator but is hard to extend safely: it mixes config, build, process supervision, and log UX into one bash script. We want a generic runner that:

- stays teammate-friendly (plugins are scripts/binaries),
- remains deterministic when multiple plugins participate (explicit merge rules),
- is safe and debuggable (stdout protocol invariants, cancellation, timeouts, structured errors),
- provides good UX without hard-coding repo-specific logic.

## Proposed Solution

### Layering

1. **Protocol**: message types + validation + JSON (un)marshal.
2. **Runtime**: spawn plugin process; enforce stdout=NDJSON-only; correlate request/response; route events.
3. **Engine**: phase pipeline + deterministic composition (patches, validations, launch plans).
4. **CLI**: Cobra commands + config loading + plugin discovery + dynamic plugin commands.

### MVP behavior

- Plan-mode launch (`launch.plan`) is MVP: plugins return service specs; runner supervises.
- Controller-mode launch is kept as an extension point but can be deferred.

### “Generic runner” boundary

- Runner is generic and repo-agnostic.
- Repo-specific behavior lives in plugins and config (e.g., `pnpm install` as `prepare.run` step, not core runner code).

### Go module placement

This ticket’s code should live inside the existing Go module at `glazed/` (so we do not introduce a new `go.mod`). The CLI binary is a new `cmd/` entrypoint, with reusable packages under `pkg/`.

### Proposed file layout

```text
glazed/
  cmd/
    devctl/
      main.go
      cmds/
        root.go
        up.go
        down.go
        status.go
        logs.go
        plugins.go
        exec.go              # run plugin command.run directly
        dynamic_commands.go  # commands.list → cobra subcommands
  pkg/
    devctl/
      config/
        config.go            # repo config model + load/merge
      discovery/
        discovery.go         # find plugins, ordering, filtering
      protocol/
        types.go             # Handshake/Request/Response/Event + payload types
        validate.go          # protocol validation helpers
        errors.go            # protocol + runtime error codes
      runtime/
        factory.go           # start plugin processes + handshake
        client.go            # Client: Call + StartStream
        io.go                # NDJSON reader/writer (stdout purity)
        router.go            # request_id + stream_id routing
        process_unix.go      # process group termination (build tags if needed)
      patch/
        patch.go             # ConfigPatch apply + dotted-path helpers
      engine/
        pipeline.go          # phase orchestration
        merge.go             # deterministic merge policies + strictness
      supervise/
        supervisor.go        # run launch.plan services
        health.go            # tcp/http health checks
        logs.go              # capture/follow logs per service
      testdata/
        plugins/
          ok-bash/...
          noisy-stdout/...
          slow-timeout/...

moments/
  plugins/                   # repo-owned plugins (not inside Go module)
    devctl-*.sh
```

### Package APIs (signatures)

#### Protocol (`glazed/pkg/devctl/protocol`)

```go
package protocol

type ProtocolVersion string

const ProtocolV1 ProtocolVersion = "v1"

type FrameType string

const (
    FrameHandshake FrameType = "handshake"
    FrameRequest   FrameType = "request"
    FrameResponse  FrameType = "response"
    FrameEvent     FrameType = "event"
)

type Capabilities struct {
    Ops      []string `json:"ops,omitempty"`
    Streams  []string `json:"streams,omitempty"`
    Commands []string `json:"commands,omitempty"`
}

type Handshake struct {
    Type            FrameType        `json:"type"` // "handshake"
    ProtocolVersion ProtocolVersion  `json:"protocol_version"`
    PluginName      string           `json:"plugin_name"`
    Capabilities    Capabilities     `json:"capabilities"`
    Declares        map[string]any   `json:"declares,omitempty"`
}

type RequestContext struct {
    RepoRoot    string `json:"repo_root,omitempty"`
    Cwd         string `json:"cwd,omitempty"`
    DeadlineMs  int64  `json:"deadline_ms,omitempty"`
    DryRun      bool   `json:"dry_run,omitempty"`
}

type Request struct {
    Type      FrameType       `json:"type"` // "request"
    RequestID string          `json:"request_id"`
    Op        string          `json:"op"`
    Ctx       RequestContext  `json:"ctx"`
    Input     json.RawMessage `json:"input,omitempty"`
}

type Response struct {
    Type      FrameType       `json:"type"` // "response"
    RequestID string          `json:"request_id"`
    Ok        bool            `json:"ok"`
    Output    json.RawMessage `json:"output,omitempty"`
    Warnings  []Note          `json:"warnings,omitempty"`
    Notes     []Note          `json:"notes,omitempty"`
    Error     *Error          `json:"error,omitempty"`
}

type Event struct {
    Type     FrameType      `json:"type"` // "event"
    StreamID string         `json:"stream_id"`
    Event    string         `json:"event"` // "log"|"status"|"end"|...
    Level    string         `json:"level,omitempty"`
    Message  string         `json:"message,omitempty"`
    Fields   map[string]any `json:"fields,omitempty"`
    Ok       *bool          `json:"ok,omitempty"` // for end events
}

type Error struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}

type Note struct {
    Level   string `json:"level"`
    Message string `json:"message"`
}

func ValidateHandshake(h Handshake) error
func ValidateFrameType(t FrameType) error
```

#### Runtime (`glazed/pkg/devctl/runtime`)

```go
package runtime

type PluginID string

type PluginSpec struct {
    ID       PluginID
    Path     string
    Args     []string
    Env      map[string]string
    WorkDir  string
    Priority int // lower runs earlier
}

type Client interface {
    Spec() PluginSpec
    Handshake() protocol.Handshake

    SupportsOp(op string) bool

    Call(ctx context.Context, op string, input any, output any) error
    StartStream(ctx context.Context, op string, input any) (streamID string, events <-chan protocol.Event, err error)

    Close(ctx context.Context) error
}

type Factory interface {
    Start(ctx context.Context, spec PluginSpec) (Client, error)
}
```

#### Config patching (`glazed/pkg/devctl/patch`)

```go
package patch

type Config = map[string]any

type ConfigPatch struct {
    Set   map[string]any `json:"set,omitempty"`   // dotted keys
    Unset []string       `json:"unset,omitempty"` // dotted keys
}

func Apply(cfg Config, p ConfigPatch) (Config, error)
func Merge(a, b ConfigPatch) ConfigPatch
```

#### Engine (`glazed/pkg/devctl/engine`)

```go
package engine

type Strictness int

const (
    StrictWarn Strictness = iota
    StrictError
)

type Options struct {
    Strictness Strictness
    DryRun     bool
    Timeout    time.Duration
    RepoRoot   string
}

type Pipeline struct {
    Clients []runtime.Client
    Opts    Options
}

func (p *Pipeline) MutateConfig(ctx context.Context, cfg patch.Config) (patch.Config, error)
func (p *Pipeline) Build(ctx context.Context, cfg patch.Config, steps []string) (BuildResult, error)
func (p *Pipeline) Prepare(ctx context.Context, cfg patch.Config, steps []string) (PrepareResult, error)
func (p *Pipeline) Validate(ctx context.Context, cfg patch.Config) (ValidateResult, error)
func (p *Pipeline) LaunchPlan(ctx context.Context, cfg patch.Config) (LaunchPlan, error)
```

#### Supervisor (`glazed/pkg/devctl/supervise`)

```go
package supervise

type Supervisor interface {
    Start(ctx context.Context, plan engine.LaunchPlan) error
    Stop(ctx context.Context) error
    Status(ctx context.Context) ([]ServiceStatus, error)
    LogsFollow(ctx context.Context, service string) (<-chan string, error)
}

type ServiceStatus struct {
    Name   string
    PID    int
    State  string // starting|running|failed|stopped
    Health string // unknown|ok|unhealthy
}
```

### Runtime pseudocode (critical flows)

#### Start plugin process + handshake

```go
func (f *FactoryImpl) Start(ctx context.Context, spec PluginSpec) (Client, error) {
    cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
    cmd.Dir = spec.WorkDir
    cmd.Env = mergeEnv(os.Environ(), spec.Env)
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    stdin,  _ := cmd.StdinPipe()

    if err := cmd.Start(); err != nil { return nil, err }

    hs, err := readHandshakeWithTimeout(ctx, stdout, 5*time.Second)
    if err != nil { terminateProcessGroup(cmd); return nil, err }
    if err := protocol.ValidateHandshake(hs); err != nil { terminateProcessGroup(cmd); return nil, err }

    c := newClient(spec, hs, cmd, stdin)
    c.router = newRouter()
    go c.readStdoutFrames(stdout) // routes responses/events; detects non-JSON
    go c.readStderrLines(stderr)  // human logs
    return c, nil
}
```

#### Call (request/response correlation)

```go
func (c *client) Call(ctx context.Context, op string, input any, output any) error {
    rid := c.nextRequestID()
    pending := c.router.registerPending(rid)

    req := protocol.Request{
        Type: protocol.FrameRequest, RequestID: rid, Op: op,
        Ctx: wireCtx(ctx, c.repoRoot), Input: mustMarshal(input),
    }
    if err := c.writeFrame(req); err != nil { c.router.cancelPending(rid); return err }

    select {
    case resp := <-pending:
        if !resp.Ok { return errors.Wrap(protocolError(resp.Error), "plugin call failed") }
        if output != nil { return json.Unmarshal(resp.Output, output) }
        return nil
    case <-ctx.Done():
        c.router.cancelPending(rid)
        return ctx.Err()
    }
}
```

### CLI command mapping (MVP)

- `devctl up`: discovery → start plugins → pipeline (mutate/build/prepare/validate/launch.plan) → supervisor start → status output
- `devctl down`: supervisor stop → plugin Close
- `devctl status`: supervisor status
- `devctl logs --follow <service>`: supervisor logs (or delegate to `logs.follow` plugin when no supervised service exists)
- `devctl plugins list`: discovery + handshake summaries

### Repo config format (proposed)

Default config file: `.devctl.yaml` at repo root (or `--config` override). Minimal shape:

```yaml
plugins:
  - id: moments-config
    path: ./moments/plugins/devctl-config.sh
    priority: 10
    workdir: .
    env:
      FOO: bar
  - id: moments-launch
    path: ./moments/plugins/devctl-launch.sh
    priority: 20
strictness: warn # or "error"
```

Go model:

```go
package config

type File struct {
    Plugins    []Plugin `yaml:"plugins"`
    Strictness string   `yaml:"strictness,omitempty"`
}

type Plugin struct {
    ID       string            `yaml:"id"`
    Path     string            `yaml:"path"`
    Priority int               `yaml:"priority,omitempty"`
    WorkDir  string            `yaml:"workdir,omitempty"`
    Env      map[string]string `yaml:"env,omitempty"`
    Args     []string          `yaml:"args,omitempty"`
}
```

### Dynamic commands wiring (pseudocode)

```go
func addPluginCommands(root *cobra.Command, pipeline *engine.Pipeline) error {
    cmds := collectCommandsFromPlugins(ctx, pipeline.Clients) // calls commands.list
    for _, cmdSpec := range cmds {
        root.AddCommand(&cobra.Command{
            Use:   cmdSpec.Name,
            Short: cmdSpec.Help,
            RunE: func(cmd *cobra.Command, argv []string) error {
                // call command.run with {name, argv, config}
                return runPluginCommand(ctx, pipeline, cmdSpec.Name, argv)
            },
        })
    }
    return nil
}
```

## Design Decisions

### NDJSON framing + stdout/stderr split

- Plugin stdout must be JSON frames only (one JSON object per line).
- Any non-JSON stdout is a protocol error (`E_PROTOCOL_STDOUT_CONTAMINATION`).
- Plugin stderr is treated as human logs (captured/streamed).

Implementation note: avoid `bufio.Scanner` default token limit; use `bufio.Reader.ReadBytes('\n')` (or increase scanner buffer).

### Concurrency model

Per plugin process:
- one goroutine reads stdout frames and routes them (responses/events),
- one goroutine reads stderr and forwards to logger/UI,
- stdin writes are serialized (single writer goroutine or mutex).

Requests are correlated by `request_id`; streams are routed by `stream_id`.

### Context + cancellation + process cleanup

- All public APIs take `context.Context`.
- On cancellation/shutdown, terminate plugin process group (unix): `SIGTERM` → grace → `SIGKILL`.
  - Use `exec.Cmd` + `SysProcAttr{Setpgid:true}` to avoid leaking child processes.

### Deterministic composition rules

- Config patches: apply in plugin order; later `set` wins on same dotted key; `unset` keys remove.
- Validate: overall `valid = AND`; errors/warnings append.
- Launch plan: merge services by name; collisions governed by strictness policy.
- Commands: merge by name; collisions governed by strictness policy.

## Alternatives Considered

### Length-prefixed framing (v2)

More robust for large payloads, but much harder for bash; keep as future evolution if needed.

### Plugins return full config blobs

Rejected: merges become arbitrary and hard to reason about; patch-based merges stay explicit and deterministic.

### Controller-mode only launch

Rejected for MVP: plan-mode enables a simpler, testable supervisor and covers most repos.

## Implementation Plan

Each step should land with tests for the component it introduces.

1. Scaffold `devctl` Cobra CLI + logging (zerolog) under the existing Go module.
2. Implement protocol types + validation (`handshake/request/response/event`).
3. Implement plugin runtime (spawn + handshake + stdout purity + call correlation + event routing).
4. Implement dotted-path config patch apply + merge (`set`/`unset`).
5. Implement engine pipeline (config mutate + validate first; then build/prepare).
6. Implement plan-mode service supervisor (start/stop/status, tcp/http health checks, log capture).
7. Wire CLI commands: `up/down/status/logs/plugins`.
8. Implement plugin command registration (`commands.list` + dynamic cobra subcommands; `command.run` dispatch).
9. Add test harness with fake plugins (ok/noisy/slow) to cover framing errors, timeouts, and stream routing.
10. Publish plugin authoring guide + templates (bash/python) matching v1 protocol.

## Open Questions

1. Do we need controller-mode launch (`launch.controller.start`) in the initial MVP?
2. What should default strictness be for collisions (warn vs error) for services/commands?
3. Is dotted-path patching sufficient, or do we need JSONPath-like keys?

## References

- `design-doc/04-script-plugin-protocol-config-mutation-build-prepare-validate-launch-monitor-logs-commands.md`
- `analysis/01-startdev-sh-complete-step-by-step-analysis.md`
