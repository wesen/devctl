---
Title: Testing Diary
Ticket: MO-012-PORT-CMDS-TO-GLAZED
Status: active
Topics:
    - devctl
    - glazed
    - cli
    - refactor
    - docs
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/cmd/devctl/main.go
      Note: Help system initialization that triggered doc parsing
    - Path: devctl/go.mod
      Note: go mod tidy captured new help-system deps
    - Path: devctl/pkg/doc/topics/devctl-user-guide.md
      Note: Frontmatter quoting fix (Short contains ':')
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-08T02:00:43.053305035-05:00
WhatFor: ""
WhenToUse: ""
---


# Testing Diary

## Goal

Record the real test runs performed while porting devctl’s Cobra CLI toward Glazed (MO-012), including exact commands and failures, so regressions are easy to reproduce and future validation is consistent.

## Step 1: Post-Help-System Smoke + Unit Test Pass

This step validates the first “Glazed integration” changes (help system initialization + embedded devctl docs + first Glazed-ported command). The focus is to ensure the repository still builds, `go test` is green, and smoketests continue to exercise the real protocol/supervisor paths.

### What I did
- Ran `cd devctl && GOWORK=off go test ./... -count=1` to validate the module outside of `go.work`.
- Resolved missing `go.sum` entries by running `cd devctl && go mod tidy`.
- Ran smoketests (no TUI):
  - `cd devctl && GOWORK=off go run ./cmd/devctl dev smoketest`
  - `cd devctl && GOWORK=off go run ./cmd/devctl dev smoketest failures --timeout 20s`
  - `cd devctl && GOWORK=off go run ./cmd/devctl dev smoketest logs --timeout 20s`
  - `cd devctl && GOWORK=off go run ./cmd/devctl dev smoketest supervise --timeout 20s`
  - `cd devctl && GOWORK=off go run ./cmd/devctl dev smoketest e2e --timeout 60s`
- Fixed invalid YAML frontmatter in embedded devctl docs (quoting `Short:` values that contain `:`) after observing help-system parse errors during smoketest startup.

### Why
- `GOWORK=off go test ./...` catches module hygiene issues that can be masked by the workspace.
- Smoketests cover the “real world” paths: plugin handshake + calls + supervision + log capture + teardown.
- The Glazed help system loads embedded Markdown docs at startup; invalid frontmatter would silently drop docs and break `help` UX.

### What worked
- `cd devctl && go mod tidy` fixed the missing `go.sum` entries and `GOWORK=off go test ./... -count=1` passed afterwards.
- All smoketests passed after fixing the doc frontmatter issues (`dev`, `failures`, `logs`, `supervise`, `e2e`).

### What didn't work
- Initial `GOWORK=off go test ./... -count=1` failure (missing `go.sum` entries introduced by importing the Glazed help system):

```text
../glazed/pkg/help/help.go:12:2: missing go.sum entry for module providing package github.com/adrg/frontmatter (imported by github.com/go-go-golems/glazed/pkg/help)
../glazed/pkg/help/render.go:14:2: missing go.sum entry for module providing package github.com/charmbracelet/glamour (imported by github.com/go-go-golems/glazed/pkg/help)
../glazed/pkg/help/store/store.go:9:2: missing go.sum entry for module providing package github.com/mattn/go-sqlite3 (imported by github.com/go-go-golems/glazed/pkg/help/store)
../glazed/pkg/help/render.go:15:2: missing go.sum entry for module providing package github.com/kopoli/go-terminal-size (imported by github.com/go-go-golems/glazed/pkg/help)
.../bubbles/list/list.go:16:2: missing go.sum entry for module providing package github.com/sahilm/fuzzy (imported by github.com/charmbracelet/bubbles/list)
```

- Initial smoketest run printed help-system doc load errors due to invalid YAML frontmatter (unquoted `Short:` values containing `:`):

```json
{"level":"debug","error":"yaml: line 3: mapping values are not allowed in this context","file":"topics/devctl-user-guide.md","time":"...","message":"Failed to load section from file"}
```

### What I learned
- The Glazed help system uses YAML frontmatter parsing that is stricter than “human YAML”: plain scalars containing `:` should be quoted (especially `Short:` fields with “...: ...” phrasing).
- Importing `glazed/pkg/help` pulls additional dependencies that were previously unused by devctl; `go mod tidy` is necessary to keep `GOWORK=off` builds clean.

### What was tricky to build
- The failure mode for frontmatter issues is “soft”: sections fail to load but the CLI still runs, so the only obvious signal can be debug logs during startup.

### What warrants a second pair of eyes
- Whether we want to treat “help docs failed to load” as a hard error in devctl (fail fast) vs current “log + continue” behavior inherited from Glazed.

### What should be done in the future
- Add an explicit “docs load validation” test or a small startup self-check that fails CI if any embedded doc has invalid frontmatter.

### Code review instructions
- Review `devctl/pkg/doc/topics/*.md` frontmatter for quoting, especially `Short:` values that contain `:`.
- Validate:
  - `cd devctl && GOWORK=off go test ./... -count=1`
  - `cd devctl && GOWORK=off go run ./cmd/devctl dev smoketest e2e --timeout 60s`

### Technical details
- Fixture plugins live under `devctl/testdata/plugins/`.
