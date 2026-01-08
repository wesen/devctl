---
Title: 'Streams: codebase analysis and TUI integration'
Ticket: MO-011-IMPLEMENT-STREAMS
Status: active
Topics:
    - streams
    - tui
    - plugins
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/cmd/devctl/cmds/tui.go
      Note: |-
        Current TUI bus wiring (what would need to emit/consume stream events).
        Current TUI bus wiring; shows missing stream runner/event types
    - Path: devctl/pkg/protocol/types.go
      Note: |-
        Handshake/request/response/event schemas and capabilities fields.
        Protocol frame schemas (handshake/request/response/event) and capabilities fields
    - Path: devctl/pkg/runtime/client.go
      Note: |-
        StartStream implementation, capability checks, read loops, Close semantics.
        StartStream implementation
    - Path: devctl/pkg/runtime/router.go
      Note: |-
        Stream multiplexing, buffering, end-of-stream closure behavior.
        Stream event multiplexing
    - Path: devctl/pkg/runtime/runtime_test.go
      Note: |-
        Tests that validate StartStream behavior and close semantics.
        Tests that validate StartStream behavior and stream channel closure
    - Path: devctl/testdata/plugins/long-running-plugin/plugin.py
      Note: |-
        Long-running logs.follow fixture (tick loop; end on stdin close).
        Long-running logs.follow stream fixture (tick loop; end on stdin close)
    - Path: devctl/testdata/plugins/stream/plugin.py
      Note: |-
        Minimal deterministic streaming fixture (hello/world/end).
        Deterministic hello/world stream fixture
ExternalSources: []
Summary: Textbook-style mapping of devctl plugin streaming (protocol + runtime) and the missing TUI plumbing required to use streams safely without hangs (especially in the presence of “streams-only” fixture plugins).
LastUpdated: 2026-01-07T16:20:07.146735192-05:00
WhatFor: Map how devctl “streams” are implemented today (protocol + runtime), where they are intended to be used (logs.follow, plugin-provided metrics/log aggregation), and what plumbing is required to integrate them into the Bubble Tea TUI.
WhenToUse: When implementing MO-011 (stream features), wiring plugin streaming into the TUI/CLI, or debugging missing/blocked streams and capability-gating hangs.
---


# Streams: codebase analysis and TUI integration

## Goal

Provide a “textbook-style” walkthrough of:
- the stream protocol surface (frames, capabilities, event semantics),
- the Go runtime implementation (`runtime.Client.StartStream` + `router`),
- where streams exist but are not yet used in production code,
- and how streams are expected to integrate into the `devctl` TUI (and related fixtures).

This document is intentionally concrete: it names the exact files and functions that exist today, highlights mismatches between design docs and code, and proposes integration points and message flows.

## Executive summary (what exists vs what’s missing)

**What exists (already implemented):**
- Protocol frame types include `event` and the handshake includes `capabilities.streams` (`devctl/pkg/protocol/types.go`).
- `runtime.Client.StartStream(ctx, op, input)` is implemented and tested (`devctl/pkg/runtime/client.go`, `devctl/pkg/runtime/runtime_test.go`).
- A stream multiplexer exists (`devctl/pkg/runtime/router.go`) which:
  - routes responses by `request_id`,
  - routes events by `stream_id`,
  - buffers early events until a subscriber subscribes,
  - closes subscribers on `event=end` or fatal protocol errors.
- Plugin fixtures exist that demonstrate streaming behavior:
  - `devctl/testdata/plugins/stream/plugin.py` (`stream.start` → “hello/world” → end)
  - `devctl/testdata/plugins/long-running-plugin/plugin.py` (`logs.follow` → “tick N” loop → end on stdin close)

**What is missing (no production integration yet):**
- No production code calls `StartStream` today (only tests and docs). CLI `devctl logs --follow` tails files; the TUI tails `.devctl/logs/*.log` directly.
- The current TUI does not expose a “start stream” command nor does its bus carry stream events (its event pipeline is limited to snapshots + pipeline lifecycle).
- The Plugins view in the current TUI does not introspect handshake capabilities; it only shows configured plugin IDs/paths (so it can’t show stream capabilities yet).

**High-impact gotcha:**
- `StartStream` currently allows starting a stream op if it appears in either `capabilities.ops` *or* `capabilities.streams`. That means a plugin that only lists `streams` (but does not implement/respond to the op) can still be invoked and may hang until timeout. This is not hypothetical: the “logger” fixture in the comprehensive TUI setup advertises `streams: ["logs.aggregate"]` but never responds to requests.

## Terminology and mental model

“Stream” is overloaded in this repo. Disambiguate early:

1) **Protocol streams (the subject of this ticket)**  
   A plugin starts a stream by responding to a request with `output.stream_id`, then emits `event` frames tagged by that `stream_id` until it emits `event="end"`.

2) **Stdout/stderr “streams” for supervised services (already in TUI)**  
   The TUI’s Service view uses `tab` to switch between “stdout” and “stderr” log files. This is not the plugin stream protocol; it’s just file tailing (`devctl/pkg/tui/models/service_model.go`).

In this document, “stream” means (1) unless explicitly stated otherwise.

## The protocol surface (NDJSON frames)

### Frame schemas (Go types)

Defined in `devctl/pkg/protocol/types.go`:

```go
type Capabilities struct {
    Ops      []string      `json:"ops,omitempty"`
    Streams  []string      `json:"streams,omitempty"`
    Commands []CommandSpec `json:"commands,omitempty"`
}

type Request struct {
    Type      FrameType       `json:"type"`        // "request"
    RequestID string          `json:"request_id"`
    Op        string          `json:"op"`
    Ctx       RequestContext  `json:"ctx"`
    Input     json.RawMessage `json:"input,omitempty"`
}

type Response struct {
    Type      FrameType       `json:"type"`        // "response"
    RequestID string          `json:"request_id"`
    Ok        bool            `json:"ok"`
    Output    json.RawMessage `json:"output,omitempty"`
    Error     *Error          `json:"error,omitempty"`
}

type Event struct {
    Type     FrameType      `json:"type"`          // "event"
    StreamID string         `json:"stream_id"`
    Event    string         `json:"event"`         // e.g. "log" | "end"
    Level    string         `json:"level,omitempty"`
    Message  string         `json:"message,omitempty"`
    Fields   map[string]any `json:"fields,omitempty"`
    Ok       *bool          `json:"ok,omitempty"`  // usually on end
}
```

### Streaming contract (as documented)

The plugin authoring guide describes streams as “follow-style” ops:
- `devctl/pkg/doc/topics/devctl-plugin-authoring.md` §5.4 “Event (stdout, streaming)”

And the older protocol design doc gives a concrete `logs.follow` schema:
- `devctl/ttmp/2026/01/06/MO-005-IMPROVE-STARTDEV--improve-startdev-sh-script-analysis-and-enhancement/design-doc/04-script-plugin-protocol-config-mutation-build-prepare-validate-launch-monitor-logs-commands.md`

Relevant excerpt (simplified):

```json
// request input
{"source":"backend","since":"-5m"}

// response output
{"stream_id":"stream-logs-1"}

// events
{"type":"event","stream_id":"stream-logs-1","event":"log","fields":{"source":"backend"},"message":"..."}
{"type":"event","stream_id":"stream-logs-1","event":"end","ok":true}
```

## The runtime implementation (what actually runs)

### Client API

Defined in `devctl/pkg/runtime/client.go`:

```go
type Client interface {
    SupportsOp(op string) bool
    Call(ctx context.Context, op string, input any, output any) error
    StartStream(ctx context.Context, op string, input any) (streamID string, events <-chan protocol.Event, err error)
    Close(ctx context.Context) error
}
```

### `StartStream` behavior (request-started, event-driven)

Implementation file: `devctl/pkg/runtime/client.go`

Operationally:
1) Build and send a `protocol.Request` (`type=request`) to the plugin process.
2) Wait for a `protocol.Response` with `output.stream_id`.
3) Subscribe to that `stream_id` in the router and return the event channel.
4) As `event` frames arrive on stdout, they are multiplexed by `stream_id` and delivered to subscribers.
5) When an event with `event="end"` arrives, the router closes the subscribers for that stream.

Pseudocode approximation (based on the real code):

```go
func (c *client) StartStream(ctx, op, input) (streamID string, events <-chan Event, err error) {
    // NOTE: current code allows op if it is in either ops OR streams
    if op not in handshake.capabilities.ops AND op not in handshake.capabilities.streams:
        return E_UNSUPPORTED

    rid := nextRequestID()
    respCh := router.register(rid)
    writeFrame(Request{request_id: rid, op: op, input: json(input)})

    resp := await(respCh or ctx.Done)
    if resp.ok == false: return OpError{code: resp.error.code, ...}

    streamID := resp.output.stream_id
    eventsCh := router.subscribe(streamID)
    return streamID, eventsCh, nil
}
```

### Stream multiplexing and buffering (`router`)

Implementation file: `devctl/pkg/runtime/router.go`

Key invariants:
- Events are keyed by `Event.StreamID`.
- If events arrive before anyone subscribes, they are buffered in `router.buffer[streamID]`.
- When a subscriber subscribes, it receives buffered events first.
- If the buffered events already contain an `end`, the subscribe call immediately returns a channel that is pre-filled with buffered events and then closed.
- On any fatal stdout/protocol error, `failAll`:
  - fails pending requests,
  - and closes all stream subscriber channels.

Router data model (literal fields in code):

```go
type router struct {
    pending map[string]chan protocol.Response
    streams map[string][]chan protocol.Event
    buffer  map[string][]protocol.Event
    fatal   error
}
```

This design is specifically meant to handle an important race:
- the plugin may emit an event immediately after its response,
- and the Go client must not lose it even if `subscribe(streamID)` happens slightly later.

### What tests prove today

Tests: `devctl/pkg/runtime/runtime_test.go`

Proven behaviors:
- `TestRuntime_Stream`: basic StartStream, receive `log` events, stop on `end`.
- `TestRuntime_StreamClosesOnClientClose`: stream channel closes when the client is closed (plugin termination forces EOF → router.failAll → closes subscriber channels).

Fixtures used by tests:
- `devctl/testdata/plugins/stream/plugin.py`
- `devctl/testdata/plugins/long-running-plugin/plugin.py`

## Where streams are supposed to be used (design intent)

This repo already documents several planned stream use cases:

### 1) Log following via plugin (`logs.follow`)

The older protocol design explicitly models `logs.list` + `logs.follow` as plugin ops.

This gives devctl a path to support:
- logs that are not local files (remote dev envs, docker logs, aggregator processes),
- log discovery (multiple sources, not just supervised service stdout/stderr),
- uniform “follow” semantics even when there is no `.devctl/state.json` (no supervised process).

### 2) “Streamy” plugin features (metrics, log aggregation, monitors)

The comprehensive fixture generator for the TUI creates plugins that advertise streams:
- `logs.aggregate`
- `metrics.stream`

File: `devctl/ttmp/2026/01/06/MO-009-TUI-COMPLETE-FEATURES--complete-tui-features-per-mo-006-design/scripts/setup-comprehensive-fixture.sh`

Important: those fixture plugins are for *display and robustness testing*, not for correct streaming behavior.
In particular, the “logger” plugin:
- advertises `streams: ["logs.aggregate"]`,
- has `ops: []`,
- and never responds to requests (it just consumes stdin).

This fixture is a forcing function for capability gating: a naive “start stream if listed in `capabilities.streams`” implementation will hang.

### 3) Live pipeline output and progress (UI scaffolding exists)

The current `PipelineModel` has UI-level message types for:
- `PipelineLiveOutputMsg`
- `PipelineStepProgressMsg`
- `PipelineConfigPatchesMsg`

Files:
- `devctl/pkg/tui/msgs.go`
- `devctl/pkg/tui/models/pipeline_model.go`

But the current TUI bus/transformer/forwarder does not produce or forward these message types (see the “design fray” section in `devctl/ttmp/2026/01/07/MO-010-DEVCTL-CLEANUP-PASS--devctl-cleanup-pass/analysis/03-devctl-fundamentals-events-ui-messages-textbook-walkthrough.md`).

Streams are a plausible transport for that missing data, but implementing it requires a clear protocol shape (what event types are emitted, how to associate to a run/step, etc.).

## How streams would integrate into the current TUI (once implemented)

### Current TUI architecture (what exists today)

Entry point: `devctl/cmd/devctl/cmds/tui.go`

Core components:
- `Bus` (in-memory Watermill pubsub): `devctl/pkg/tui/bus.go`
- Domain → UI transformer: `devctl/pkg/tui/transform.go`
- UI → BubbleTea forwarder: `devctl/pkg/tui/forward.go`
- State snapshots via polling: `devctl/pkg/tui/state_watcher.go`
- Action execution (up/down/restart): `devctl/pkg/tui/action_runner.go`

The key architectural pattern is:
1) side-effectful subsystems publish domain events to `TopicDevctlEvents`,
2) the transformer re-publishes UI events to `TopicUIMessages`,
3) the forwarder converts UI events into `tea.Msg` and calls `Program.Send(...)`.

### Why streams are currently “invisible” to the TUI

There is no message type for “stream started” or “stream event” in:
- `devctl/pkg/tui/topics.go` (domain/ui types),
- `devctl/pkg/tui/transform.go` (domain→ui mapping),
- `devctl/pkg/tui/forward.go` (ui→tea mapping),
- `devctl/pkg/tui/msgs.go` (Bubble Tea msg types).

There is also no handler that runs `runtime.Client.StartStream` in response to user actions.

### Integration pattern A (recommended): add a dedicated stream runner + typed events

Add a new UI request type and a new domain event type.

#### Proposed API (TUI-level)

New request payloads (conceptual):

```go
// devctl/pkg/tui/streams.go (new)
type StreamStartRequest struct {
    PluginID string         `json:"plugin_id"` // optional, if resolved elsewhere
    Op       string         `json:"op"`        // e.g. "logs.follow"
    Input    map[string]any `json:"input,omitempty"`
}

type StreamStopRequest struct {
    StreamID string `json:"stream_id"`
}

type StreamEvent struct {
    StreamID string         `json:"stream_id"`
    Op       string         `json:"op,omitempty"`
    PluginID string         `json:"plugin_id,omitempty"`
    Event    protocol.Event `json:"event"`
}
```

New envelope types (conceptual additions to `devctl/pkg/tui/topics.go`):
- domain: `DomainTypeStreamEvent`, `DomainTypeStreamStarted`, `DomainTypeStreamStopped`
- ui: `UITypeStreamEvent`, `UITypeStreamStarted`, `UITypeStreamStopped`

#### Proposed runner (pseudocode)

```go
// handler listens for UITypeStreamStartRequest on TopicUIActions (or a new topic)
func RegisterUIStreamRunner(bus *Bus, opts RootOptions) {
  bus.AddHandler("devctl-ui-streams", TopicUIActions, func(msg *message.Message) error {
    // parse envelope -> StreamStartRequest
    // resolve plugin spec (from repo config)
    // start plugin client via runtime.Factory
    // IMPORTANT: capability gating should be on capabilities.ops (not only streams)
    // start stream, then publish StreamEvent domain events for each protocol.Event
  })
}
```

Inside the stream loop:

```go
_, events, err := client.StartStream(ctx, req.Op, req.Input)
if err != nil { publish error; return }
for ev := range events {
  publish(DomainTypeStreamEvent, StreamEvent{StreamID: streamID, Event: ev, ...})
}
```

#### Where does the stream render?

You have options:
- append stream events to the existing Events view (`EventLogModel`) for “global visibility”,
- or introduce a new view/model dedicated to streams (e.g., a metrics stream pane),
- or feed `logs.follow` stream events into the Service view instead of file tailing.

Which one is “right” depends on what stream ops you ship first:
- `logs.follow` → Service view is the natural sink.
- `metrics.stream` → a separate “metrics” view or dashboard overlay may be better.

### Integration pattern B: “hybrid logs” (file tail when supervised, stream when not)

This aligns with existing CLI/TUI behavior and reduces risk:
- If the service is supervised, logs exist as files (`state.ServiceRecord.StdoutLog/StderrLog`) → keep tailing files.
- If no supervised service exists (or logs are not local files), fall back to plugin `logs.list/logs.follow`.

This is consistent with the MO-005 design note:
> “devctl logs --follow <service>: supervisor logs (or delegate to logs.follow plugin when no supervised service exists)”

### Capability and fixture-driven constraints (don’t reintroduce hangs)

To make streaming robust in the presence of real-world plugins and existing fixtures:

- Treat `capabilities.ops` as authoritative for whether devctl should send a request at all.
- Treat `capabilities.streams` as informational / UX-facing (“this op is stream-producing”), unless you explicitly decide on a stricter rule like: `StartStream allowed only if op in ops AND op in streams`.

Why this matters:
- The comprehensive fixture’s “logger” plugin advertises a stream capability but never responds.
- `StartStream` allowing “streams-only ops” is enough to recreate the “hang until timeout” failure mode.

## Plugin fixtures that expose stream functionality

### 1) Minimal stream fixture (deterministic hello/world)

File: `devctl/testdata/plugins/stream/plugin.py`

- Handshake: `capabilities.ops = ["stream.start"]`
- Behavior:
  - responds to `stream.start` with `stream_id = "s1"`
  - emits `event=log` twice (“hello”, “world”)
  - emits `event=end`

### 2) Long-running logs.follow fixture (cancellation/close semantics)

File: `devctl/testdata/plugins/long-running-plugin/plugin.py`

- Handshake: `capabilities.ops = ["logs.follow"]`
- Behavior:
  - responds with `stream_id = "stream-<rid>"`
  - emits `event=log` every 100ms (“tick N”)
  - emits `event=end` when stdin closes (test uses `client.Close(...)`)

### 3) Comprehensive fixture “stream-advertising” plugins (negative testing)

File: `devctl/ttmp/2026/01/06/MO-009-TUI-COMPLETE-FEATURES--complete-tui-features-per-mo-006-design/scripts/setup-comprehensive-fixture.sh`

- `plugins/logger.py`: advertises `streams: ["logs.aggregate"]` but does not implement any ops and never responds.
- `plugins/metrics.py`: advertises `streams: ["metrics.stream"]` and `ops: ["metrics.collect"]` (but responds `ok=true` for any request in the fixture script).

These are excellent fixtures for validating that:
- stream initiation is properly capability-gated,
- timeouts are short and failures are surfaced without hanging the whole UI.

## “Delta map”: MO-006 TUI stream design vs current implementation

The MO-006 layout source doc describes a richer event bus model with topics like `cmd.logs.follow` and `service.logs.line`.

File: `devctl/ttmp/2026/01/06/MO-006-DEVCTL-TUI--create-a-devctl-tui/sources/local/01-devctl-tui-layout.md`

Key differences vs code today:
- Design doc: TUI publishes `cmd.logs.follow` and subscribes to `service.logs.line`.
- Current code: TUI publishes only `tui.action.request` (up/down/restart) and tails service logs from disk; there is no log-follow command in the bus.
- Design doc: Plugins view shows handshake capabilities (ops/streams/commands).
- Current code: Plugins view uses `StateWatcher.readPlugins()` (config-only) and explicitly leaves ops/streams/commands “would come from introspection”.

This delta is important because implementing streams in the TUI can either:
- move the current code toward the MO-006 design (topic-based bus + log events), or
- keep the current envelope/event pipeline and add a smaller stream subsystem.

## Suggested implementation breakdown for MO-011 (actionable checklist)

1) **Decide capability semantics**
   - Define whether stream-start ops must be in `capabilities.ops` (recommended), and what `capabilities.streams` means operationally.
2) **Define first stream op(s) and schemas**
   - Start with `logs.follow` (input: `{source, since}` as per MO-005) or a minimal in-repo stream op for the TUI.
3) **Add a stream runner to the TUI**
   - New request message type(s) (start/stop).
   - Start plugin client(s), call `StartStream`, publish stream events into the bus.
4) **Add bus plumbing**
   - Add domain/ui types, transformer mapping, forwarder mapping, Bubble Tea message types.
5) **Choose a UI surface**
   - Service view integration for `logs.follow`,
   - or a new Stream view for metrics/log aggregation.
6) **Validate with fixtures**
   - Positive: `devctl/testdata/plugins/stream` and `long-running-plugin`
   - Negative: comprehensive fixture “logger” (must not hang)

## Related documents (high-value pointers)

- Plugin protocol authoring guide: `devctl/pkg/doc/topics/devctl-plugin-authoring.md`
- Runtime client reference (textbook): `devctl/ttmp/2026/01/07/MO-010-DEVCTL-CLEANUP-PASS--devctl-cleanup-pass/reference/02-runtime-client-plugin-protocol-ops-streams-and-commands.md`
- Capability enforcement analysis (ops/commands/streams): `devctl/ttmp/2026/01/06/MO-009-TUI-COMPLETE-FEATURES--complete-tui-features-per-mo-006-design/analysis/05-capability-checking-and-safe-plugin-invocation-ops-commands-streams.md`
- TUI event pipeline analysis (“design fray”): `devctl/ttmp/2026/01/07/MO-010-DEVCTL-CLEANUP-PASS--devctl-cleanup-pass/analysis/03-devctl-fundamentals-events-ui-messages-textbook-walkthrough.md`
