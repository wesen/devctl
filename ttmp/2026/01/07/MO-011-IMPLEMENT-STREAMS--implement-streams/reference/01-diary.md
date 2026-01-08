---
Title: Diary
Ticket: MO-011-IMPLEMENT-STREAMS
Status: active
Topics:
    - streams
    - tui
    - plugins
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/pkg/runtime/client.go
      Note: |-
        StartStream implementation and current capability gating behavior (ops vs streams).
        StartStream capability check nuance discovered during investigation
    - Path: devctl/pkg/runtime/router.go
      Note: |-
        Stream event routing, buffering, and end-of-stream channel closure logic.
        Router buffering explains why early events are not lost
    - Path: devctl/pkg/tui/action_runner.go
      Note: |-
        Current TUI event pipeline (actions + pipeline phases) where a future stream runner would plug in.
        Current action/event pipeline that will need a stream runner sibling
    - Path: devctl/ttmp/2026/01/07/MO-011-IMPLEMENT-STREAMS--implement-streams/analysis/01-streams-codebase-analysis-and-tui-integration.md
      Note: |-
        Main deliverable analysis document for MO-011.
        Main MO-011 analysis deliverable
ExternalSources: []
Summary: 'Work log for MO-011 stream analysis: ticket creation, stream-related code inventory, and synthesis into a TUI integration plan.'
LastUpdated: 2026-01-07T16:20:08.05684409-05:00
WhatFor: Capture the investigation trail and key discoveries about devctl stream plumbing (protocol/runtime) and the gaps to integrate streams into the current TUI architecture.
WhenToUse: When continuing MO-011 implementation work, reviewing why particular files were identified as integration points, or validating that analysis assumptions match code.
---


# Diary

## Goal

Record the step-by-step investigation and documentation work for MO-011, including what was searched/read, what conclusions were drawn about streams, and how to validate/review the resulting analysis.

## Step 1: Create ticket + inventory stream plumbing

This step created the MO-011 ticket workspace and then did a codebase-wide inventory of “stream” concepts, focusing on devctl’s plugin protocol streams (`event` frames keyed by `stream_id`) rather than the TUI’s stdout/stderr log switching. The output is a textbook-style analysis that ties together protocol docs, runtime implementation, fixtures, and the current TUI event bus so implementation can proceed with fewer surprises.

The key outcome was confirming that streams are already implemented and tested in `devctl/pkg/runtime`, but there is effectively no production integration yet: the CLI and TUI follow service logs by tailing files, and the TUI bus does not carry stream events. The analysis also surfaced a correctness/robustness foot-gun: `StartStream` currently allows “streams-only” capability declarations, which would hang against an existing fixture plugin that advertises a stream but never responds.

### What I did
- Created ticket workspace and initial docs via `docmgr`.
- Searched for stream-related code paths and fixtures with `rg`, then read the relevant Go and plugin files with `sed`.
- Cross-referenced prior ticket docs (MO-005/MO-006/MO-009/MO-010) to understand intended stream usage and expected TUI integration.
- Wrote the textbook analysis document `analysis/01-streams-codebase-analysis-and-tui-integration.md`.

### Why
- Streams are a partially implemented feature with multiple similarly named “streams” concepts; mapping the ground truth prevents implementing the wrong thing.
- TUI integration requires touching multiple layers (bus types, transformer, forwarder, models); doing a call-graph style inventory first reduces churn.
- Existing fixture plugins intentionally misbehave (streams advertised but no responses); the analysis needs to bake in hang-prevention constraints.

### What worked
- `docmgr ticket create-ticket` and `docmgr doc add` produced a consistent ticket workspace under `devctl/ttmp/2026/01/07/`.
- `rg` surfaced the critical stream implementation files quickly (`runtime/client.go`, `runtime/router.go`, runtime tests, and plugin authoring docs).
- Prior docs (especially MO-010’s runtime client reference and MO-005’s logs.follow schema) provided concrete protocol shapes to anchor the analysis.

### What didn't work
- `rg -n "PipelineLiveOutput|StepProgress|ConfigPatches" devctl/pkg/tui/action_runner.go` returned no matches (confirming the current action runner does not emit live pipeline output/progress/config patch events).

### What I learned
- `runtime.Client.StartStream` exists and is tested, but there are no production call sites using it today; the “streams feature” is currently dormant outside tests/docs.
- The current TUI is event-driven (Watermill → transformer → forwarder → Bubble Tea), but it has no stream event types; adding streams implies adding new domain/UI envelopes and a dedicated runner.
- Stream capability semantics matter: a fixture plugin advertises `capabilities.streams` without implementing any ops or responses, so treating “streams list” as an invocation permission will recreate timeout/hang failure modes.

### What was tricky to build
- Terminology collision: “stream” refers both to protocol streams (`event` frames) and to local stdout/stderr log files in the Service view. The analysis had to disambiguate these to avoid misleading integration guidance.
- Design drift: MO-006’s TUI layout docs describe a richer topic-based bus (e.g., `cmd.logs.follow` → `service.logs.line`), while the current TUI implementation uses a different envelope scheme and tails log files directly.
- Capability semantics drift: docs and fixtures sometimes treat `capabilities.streams` as a declaration of stream-producing ops, but the runtime currently treats it as an allowlist for starting a stream (which is dangerous with misbehaving fixtures).

### What warrants a second pair of eyes
- The proposed capability semantics (treat `ops` as authoritative for stream-start requests) should be sanity-checked against the intended protocol contract and existing fixtures/docs.
- The “best first UI surface” for streams (Service view vs new Streams view vs Events view) is a product decision; the analysis presents options but implementation should confirm the UX direction.

### What should be done in the future
- Implement MO-011: add a stream runner + bus plumbing + UI surface, and validate against both “good” streaming fixtures and “bad” streams-advertising fixtures.

### Code review instructions
- Start with `devctl/ttmp/2026/01/07/MO-011-IMPLEMENT-STREAMS--implement-streams/analysis/01-streams-codebase-analysis-and-tui-integration.md`.
- Validate key claims by spot-checking:
  - `devctl/pkg/runtime/client.go` (`StartStream` capability check and response parsing),
  - `devctl/pkg/runtime/router.go` (buffering + `event=end` behavior),
  - `devctl/pkg/tui/transform.go` / `devctl/pkg/tui/forward.go` (no stream message types today),
  - fixture script `devctl/ttmp/2026/01/06/MO-009-TUI-COMPLETE-FEATURES--complete-tui-features-per-mo-006-design/scripts/setup-comprehensive-fixture.sh` (streams advertised without response behavior).

### Technical details
- Commands run (representative):
  - `docmgr ticket create-ticket --ticket MO-011-IMPLEMENT-STREAMS --title "Implement streams" --topics streams,tui,plugins`
  - `docmgr doc add --ticket MO-011-IMPLEMENT-STREAMS --doc-type analysis --title "Streams: codebase analysis and TUI integration"`
  - `rg -n "\\bstream(s|ing)?\\b" -S devctl moments pinocchio glazed`
  - `sed -n '1,260p' devctl/pkg/runtime/client.go`
  - `sed -n '1,260p' devctl/pkg/runtime/router.go`
