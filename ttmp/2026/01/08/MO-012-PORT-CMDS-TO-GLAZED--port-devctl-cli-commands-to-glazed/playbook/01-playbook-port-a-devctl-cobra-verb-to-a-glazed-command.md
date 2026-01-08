---
Title: 'Playbook: Port a devctl Cobra verb to a Glazed command'
Ticket: MO-012-PORT-CMDS-TO-GLAZED
Status: active
Topics:
    - devctl
    - glazed
    - cli
    - refactor
    - docs
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/cmd/devctl/cmds/common.go
      Note: Repo layer helpers used by ported commands
    - Path: devctl/cmd/devctl/cmds/plugins.go
      Note: Example of a group subcommand (plugins list) ported to Glazed
    - Path: devctl/cmd/devctl/cmds/status.go
      Note: Example of a root verb ported to a Glazed WriterCommand
    - Path: devctl/cmd/devctl/main.go
      Note: Help system setup used during the port
    - Path: devctl/pkg/doc/topics/devctl-user-guide.md
      Note: Docs examples for command-local repo flags
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-08T02:12:30.82121294-05:00
WhatFor: ""
WhenToUse: ""
---


# Playbook: Port a devctl Cobra verb to a Glazed command

## Purpose

Provide a repeatable, low-risk, intern-friendly procedure for porting a single devctl Cobra verb (or subcommand) to a Glazed command while keeping behavior stable and ensuring tests/smoketests remain green.

This playbook assumes we are in the middle of a migration: devctl is a mixed Cobra/Glazed CLI where individual verbs are being ported one at a time.

## Environment Assumptions

- You are working in the devctl repo (`cd devctl`) inside the mono-workspace, but you validate with `GOWORK=off`.
- `go` is installed and `python3` is available (smoketests and fixture plugins use Python).
- You do not need exact flag parity with the old Cobra CLI.
- Repo-context flags are command-local (used after the verb), not persistent root flags.

**Key files and patterns already established in devctl:**
- Repo-context layer + normalization helpers: `devctl/cmd/devctl/cmds/common.go`
  - `getRepoLayer()`
  - `RepoContextFromParsedLayers(...)`
- Help system wiring + doc embedding: `devctl/cmd/devctl/main.go`, `devctl/pkg/doc/doc.go`
- Example Glazed-ported verbs:
  - `devctl/cmd/devctl/cmds/status.go`
  - `devctl/cmd/devctl/cmds/plugins.go` (`plugins list`)

## Commands

```bash
# Always validate the module outside of go.work:
cd devctl
GOWORK=off go test ./... -count=1

# Basic smoketest (no TUI):
GOWORK=off go run ./cmd/devctl dev smoketest --timeout 20s

# If you touched plugin/protocol/supervision behavior, run the full suite:
GOWORK=off go run ./cmd/devctl dev smoketest failures --timeout 20s
GOWORK=off go run ./cmd/devctl dev smoketest logs --timeout 20s
GOWORK=off go run ./cmd/devctl dev smoketest supervise --timeout 20s
GOWORK=off go run ./cmd/devctl dev smoketest e2e --timeout 60s
```

## Exit Criteria

- The ported command works manually against a representative repo (or fixture repo) without changing its output format.
- `GOWORK=off go test ./... -count=1` passes.
- At least `dev smoketest` passes; run additional smoketests if you touched related areas.
- Ticket bookkeeping is updated (tasks checked, diary/changelog updated, and changes committed).

## Porting Procedure (Step-by-Step)

This is the exact workflow to follow for each command.

### 0) Decide the “command shape” (root verb vs group/subcommand)

devctl commands live in `devctl/cmd/devctl/cmds/`.

- Root verb example: `devctl status` → `devctl/cmd/devctl/cmds/status.go`
- Group/subcommand example: `devctl plugins list` → `devctl/cmd/devctl/cmds/plugins.go`

**Rule of thumb for the migration:**
- If the group root is mostly just a container (`plugins`, `stream`), keep the group root as Cobra for now, and port only the leaf subcommand(s) to Glazed.
- If you are porting an entire group at once, consider porting the group root too (but do this only when you have tests and can validate completion/help behavior).

### 1) Read the existing Cobra implementation and identify inputs/outputs

For the Cobra command you are porting:
- List its flags and defaults (including repo-context flags).
- Identify required values and validations (required flags, mutual exclusivity, etc.).
- Identify the output contract:
  - JSON (must be stable)
  - streaming text
  - “ok” style acknowledgement
  - interactive TUI (not covered in this playbook)

Write this down in the ticket as a short checklist. It becomes your “definition of done”.

### 2) Create a Glazed command type (WriterCommand is the default choice)

Most devctl verbs emit JSON or text and should be ported as `WriterCommand`.

**Skeleton (copy/paste template):**

```go
type MyCommand struct {
    *glazedcmds.CommandDescription
}

var _ glazedcmds.WriterCommand = (*MyCommand)(nil)

type MySettings struct {
    // Example flag:
    Foo string `glazed.parameter:"foo"`
}

func NewMyCommand() (*MyCommand, error) {
    repoLayer, err := getRepoLayer()
    if err != nil {
        return nil, err
    }

    return &MyCommand{
        CommandDescription: glazedcmds.NewCommandDescription(
            "my-verb",
            glazedcmds.WithShort("One-line summary"),
            glazedcmds.WithFlags(
                parameters.NewParameterDefinition(
                    "foo",
                    parameters.ParameterTypeString,
                    parameters.WithDefault(""),
                    parameters.WithHelp("Explain the flag"),
                ),
            ),
            glazedcmds.WithLayersList(repoLayer),
        ),
    }, nil
}

func (c *MyCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
    s := MySettings{}
    if err := parsedLayers.InitializeStruct(layers.DefaultSlug, &s); err != nil {
        return err
    }

    rc, err := RepoContextFromParsedLayers(parsedLayers)
    if err != nil {
        return err
    }

    // Implement the command body using rc.RepoRoot/rc.ConfigPath/rc.Timeout/etc.
    // Write output to w (not cmd.OutOrStdout()).
    return nil
}
```

**Important implementation notes:**
- Always include the repo layer via `getRepoLayer()` if the command depends on repo context.
- Parse repo context via `RepoContextFromParsedLayers(parsedLayers)` (do not re-parse Cobra flags manually).
- Write output to the provided `io.Writer` `w`.
- Keep the output contract stable (same JSON shape, same stdout/stderr separation).

### 3) Build the Cobra command from the Glazed command

Devctl currently registers commands manually in Cobra; for now, we keep that pattern and just build the leaf command:

```go
cmd, err := cli.BuildCobraCommand(glazedCmd, cli.WithParserConfig(cli.CobraParserConfig{AppName: "devctl"}))
cobra.CheckErr(err)
```

**Where to register it:**
- Root verbs: return the built Cobra command from `new<Verb>Cmd()` (see `status.go`).
- Group subcommands: add the built Cobra command to the Cobra group root (see `plugins.go`).

### 4) Ensure repo-context flags show up on the ported command

Repo-context flags are command-local and should appear on the ported command automatically if you added the repo layer.

Manually validate:

```bash
cd devctl
GOWORK=off go run ./cmd/devctl <verb> --help
```

You should see:
- `--repo-root`, `--config`, `--strict`, `--dry-run`, `--timeout`
- plus the verb’s own flags

### 5) Update documentation if usage changed (especially flag placement)

If your port changes usage examples, update devctl docs:
- `devctl/pkg/doc/topics/devctl-user-guide.md`
- `devctl/pkg/doc/topics/devctl-scripting-guide.md`
- `devctl/pkg/doc/topics/devctl-plugin-authoring.md`
- `devctl/pkg/doc/topics/devctl-tui-guide.md` (only if relevant)

**Frontmatter gotcha:** YAML fields like `Short:` must be quoted if they contain `:` (colon), or the help-system frontmatter parser will reject the doc.

### 6) Run tests and smoketests (minimum and expanded)

Always:

```bash
cd devctl
GOWORK=off go test ./... -count=1
GOWORK=off go run ./cmd/devctl dev smoketest --timeout 20s
```

If you touched protocol/runtime/supervise/logging/dynamic commands:

```bash
GOWORK=off go run ./cmd/devctl dev smoketest failures --timeout 20s
GOWORK=off go run ./cmd/devctl dev smoketest logs --timeout 20s
GOWORK=off go run ./cmd/devctl dev smoketest supervise --timeout 20s
GOWORK=off go run ./cmd/devctl dev smoketest e2e --timeout 60s
```

### 7) Update ticket bookkeeping (tasks, diary, changelog)

For MO-012, keep these in sync:
- Check off tasks: `docmgr task check --ticket MO-012-PORT-CMDS-TO-GLAZED --id <N>`
- Update changelog entries for meaningful steps (include commit hash): `docmgr changelog update ...`
- Keep the diary up to date with what changed and why:
  - `devctl/ttmp/.../reference/01-diary.md`
  - If you ran real validation, also update `devctl/ttmp/.../reference/02-testing-diary.md`

### 8) Commit cleanly (code first, docs second)

Keep commits focused and reviewable:
- Commit code changes first (the port itself, tests).
- Commit ticket docs (diary/playbook/changelog/tasks) separately.

Recommended command sequence:

```bash
git status --porcelain
git diff --stat

# Stage intentionally
git add path/to/code.go path/to/tests.go
git diff --cached --stat
git commit -m "devctl: port <verb> to Glazed"

# Then docs
git add devctl/ttmp/...
git commit -m "MO-012: update diary and playbook"
```

## Common Failure Modes (and How to Fix Them)

### `GOWORK=off go test` fails with missing `go.sum` entries

Symptom:
- `missing go.sum entry for module providing package ...`

Fix:

```bash
cd devctl
go mod tidy
GOWORK=off go test ./... -count=1
```

### Help docs silently don’t load

Symptom:
- `devctl help <slug>` says “not found” or the topic is missing.

Common cause:
- Invalid YAML frontmatter, especially unquoted `Short:` strings containing `:`.

Fix:
- Quote the YAML scalar:
  - `Short: "Some text: with colon"`

### Repo flags don’t appear on a ported command

Cause:
- You forgot to add `repoLayer` via `glazedcmds.WithLayersList(repoLayer)`.

Fix:
- Ensure `New<Cmd>()` calls `getRepoLayer()` and includes the layer.

### Confusion about “Parents” / groups

Cause:
- `glazedcmds.WithParents("plugins")` is metadata; it does not automatically create Cobra group commands when you manually `root.AddCommand(...)`.

Fix:
- Either:
  - Add the built command under an existing Cobra group command, or
  - Switch to a registration helper that creates parents (out of scope for the intern playbook unless you’re explicitly instructed).
