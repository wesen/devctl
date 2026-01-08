---
Title: Cobra↔Glazed Porting Friction Report
Ticket: MO-012-PORT-CMDS-TO-GLAZED
Status: active
Topics:
    - devctl
    - glazed
    - cli
    - refactor
    - docs
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/cmd/devctl/main.go
      Note: Current Cobra-first root wiring (logging + root flags + dynamic commands)
    - Path: glazed/cmd/glaze/main.go
      Note: Reference Glazed-first root wiring (help system + BuildCobraCommand)
    - Path: glazed/pkg/cli/cobra-parser.go
      Note: Parser config + middleware precedence model
    - Path: glazed/pkg/cli/cobra.go
      Note: Command builder surface area; multiple entry points
    - Path: glazed/pkg/cmds/middlewares/cobra.go
      Note: ParseFromCobraCommand behavior (cmd.Flags
    - Path: glazed/pkg/cmds/parameters/parameter-type.go
      Note: Parameter type gaps (no duration/path types)
    - Path: glazed/pkg/help/cmd/cobra.go
      Note: Help system Cobra integration surface
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-08T01:37:45.075632968-05:00
WhatFor: ""
WhenToUse: ""
---


# Cobra↔Glazed Porting Friction Report

## Goal

Capture (1) everything that felt confusing or unexpectedly difficult while trying to port a real-world Cobra CLI (`devctl`) to Glazed, and (2) concrete improvements to Glazed (and related docs / examples) that would make this kind of port repeatable, low-risk, and substantially faster.

This document is intentionally opinionated and exhaustive. It is written for:

- **Glazed maintainers** (what to improve in Glazed),
- **App maintainers** (how to structure an app for Glazed),
- **Porters** (what to watch out for).

## Executive summary (high-level)

Porting from a “classic Cobra app” to “Glazed commands built into Cobra” looks straightforward from the surface (both end up as Cobra commands), but in practice the hard parts cluster around:

1. **Two parallel flag systems** (Cobra flags + Glazed layers) that don’t compose cleanly for root/persistent flags.
2. **Help system + docs loading** being “app-level wiring” with no single canonical pattern for non-Glaze apps.
3. **Parameter type gaps** (notably durations and paths) that force “stringly-typed” parameters + custom normalization per app.
4. **Parsing and precedence** (defaults vs config vs env vs flags) being powerful but under-specified for typical ports.
5. **Dynamic command trees** (commands discovered at runtime) colliding with Glazed’s design assumptions (static command description + layer sets).
6. **Library ergonomics** (naming, duplication, and the number of concepts required to build the first working command).

The single most helpful improvement would be: **a small “application builder” API + reference template** that establishes a canonical, stable pattern for:

- root command creation,
- help system initialization and doc loading,
- global/persistent layer registration (flags and parsing),
- command registration,
- validation hooks,
- dynamic commands.

Everything else can be incremental once that “default happy path” exists.

## Scope notes (what this report is and isn’t)

This report focuses on “port friction” and “glazed API/docs friction”, not on the correctness of devctl itself.

When I say “confusing”, I mean at least one of:

- It’s easy to do the wrong thing while thinking you did the right thing.
- It’s difficult to tell if you’re following the intended pattern.
- Multiple plausible patterns exist and it’s unclear which is canonical.
- The abstraction boundary leaks (you must know Cobra internals and Glazed internals at the same time).

## Mental model mismatch: Cobra-first vs Glazed-first apps

### What a Cobra-first app expects

In a Cobra-first app:

- “Global flags” are *persistent flags* on the root.
- Every command automatically sees them (both `--flag cmd` and `cmd --flag` patterns have semantics; the former is typical for global flags).
- Parsing happens once (Cobra parses everything for you).
- Help is “good enough” by default.

### What a Glazed-first app expects

In a Glazed-first app (as shown by `glazed/cmd/glaze/main.go`):

- You build commands via Glazed and add them to a Cobra root.
- Parameters are defined via **layers** and parsed via a Glazed parser with a middleware pipeline.
- Help is a “system” with docs loaded at startup, and Cobra integration is explicit.

### What happens during a port

During a port, you’re in the worst case:

- You still need Cobra persistent/global flags for UX parity.
- You want Glazed parsing to be the source of truth for settings, defaults, validation, and help grouping.
- But Glazed’s `ParseFromCobraCommand` looks at `cmd.Flags()` and expects layers to register flags on each command, while Cobra persistent flags live on the root and are inherited in Cobra-specific ways.

This creates a “two worlds” situation where you can *easily* end up with:

- the same flag being registered twice,
- a flag existing but not being parsed into layers,
- a layer existing but not being reflected into Cobra help output,
- a flag working for some verbs but not others due to `cmd.Flags()` vs `cmd.InheritedFlags()` nuances.

## Confusing / troublesome areas (exhaustive list)

### 1) Persistent/global flags vs Glazed layers

**Problem:** There is no single canonical way to define “global flags” once, have them show up as persistent flags on the Cobra root, and also have them parsed as a Glazed layer for every Glazed command.

You typically want all of these at once:

1. Register once on root as `PersistentFlags()` (Cobra UX parity).
2. Show in help grouped under a “global flags” section.
3. Parse into a typed settings struct through Glazed parsing.
4. Run validation/normalization once (paths, timeouts, defaults).

But the Glazed plumbing is oriented around “add layer flags to the specific Cobra command you’re building”, which conflicts with “persistent root flags”. This becomes extra confusing because:

- Cobra flag inheritance is not trivial to reason about,
- Glazed’s parsing middleware currently uses `cmd.Flags()` and “changed” checks,
- and required/default semantics are split between the flag registration layer and the defaults middleware.

**What would make this easier:**

- A first-class concept: **PersistentParameterLayer**, something like:

  - `cli.WithPersistentLayers(repoLayer, loggingLayer, ...)` or
  - `layers.NewPersistentLayer(...)` or
  - `cli.AppBuilder.WithGlobalLayer(...)`

  that:
  - registers flags on the root command (persistent),
  - makes parsing work from *any* subcommand without duplicating flags,
  - and integrates with flag grouping.

- A parser that explicitly supports: “gather from local flags + inherited flags”, so app developers do not need to reason about the `cmd.Flags()` vs `cmd.InheritedFlags()` detail.

### 2) “Only provided” vs defaults: where truth lives

Glazed separates:

- **parsing from Cobra** (only flags that were changed),
- **defaults middleware** (applies defaults afterwards).

This is a good design for precedence, but it is very confusing during a port because Cobra users mentally model defaults as being part of flag registration.

Consequence:

- A port can “look correct” (defaults appear in help), but the final parsed layers differ depending on how the middlewares are configured.
- You can end up with “defaults exist but don’t apply” (or apply later than expected).

**What would make this easier:**

- A single “precedence table” doc that’s part of the “BuildCobraCommand” tutorial:
  - defaults < config < env < flags < arguments (or whatever is canonical).
- A debug command (or built-in debug flags) that prints the final resolved layer values *and* a per-parameter provenance (which parse step supplied it).
  - Glazed already tracks parse step metadata; a canonical UX around it would make ports much less error-prone.

### 3) Parameter type gaps: durations, paths, “repo root”

In devctl, the most important global settings are:

- `repo-root` (path normalization),
- `config` (path resolution relative to repo-root),
- `timeout` (duration),
- strict/dry-run booleans.

Glazed lacks a `duration` parameter type, so a natural “port” becomes:

- `timeout` as a string parameter parsed by `time.ParseDuration`.

This is both awkward and inconsistent with Cobra (`DurationVar`) which provides proper duration parsing and help.

Similarly, there is no first-class “path” type that:

- expands `~`,
- normalizes to absolute,
- optionally enforces existence,
- optionally resolves relative to a base (repo-root),
- and optionally provides helpful error messages and suggestions.

**What would make this easier:**

- Add `ParameterTypeDuration` (and optionally `ParameterTypePath`) to Glazed.
- Allow `ParameterDefinition` to attach normalization and validation hooks, e.g.:
  - `parameters.WithNormalize(func(any) (any, error))`
  - `parameters.WithValidate(func(any) error)`
  - and/or a layer-level `Validate(parsedLayer)` hook.

### 4) Layer ergonomics and discoverability

It’s hard to answer: “What is the minimal set of layers every app should add?”

In Glaze itself, the app uses:

- logging layer on root,
- help system + help Cobra integration,
- per-command layers (glazed layer, command settings layer, etc).

But for a porting app:

- you might want **no glaze output layer** (Writer/Bare commands),
- still want command settings layer (debugging, config loading),
- still want help system,
- and still want global layers.

It is not obvious what the “starter kit” is.

**What would make this easier:**

- A canonical “Glazed-in-Cobra (writer-mode) app template”:
  - `cmd/myapp/main.go` shows root wiring,
  - one `WriterCommand` and one `BareCommand`,
  - a global RepoSettings layer,
  - help system,
  - config/env/default precedence.

### 5) Multiple similar entry points in `cli` (BuildCobraCommand…)

There are multiple builder functions and options:

- `cli.BuildCobraCommand`
- `cli.BuildCobraCommandFromCommandAndFunc`
- dual mode options and toggles
- parser config objects and middleware constructors

This is powerful, but it creates a “pit of success” problem: it’s hard to know which one to start with and what the recommended path is.

**What would make this easier:**

- One recommended primary API for typical apps; everything else documented as advanced.
- A “decision tree” section in docs:
  - Do you need structured output formatting? Use GlazeCommand.
  - Do you just need typed flags + help integration? Use WriterCommand/BareCommand.
  - Do you need dual-mode? Here’s when and how.

### 6) Help system integration is not “one obvious way”

Glazed has a very capable help system, but porting apps need to answer:

- Do I need to embed docs?
- Where do docs live in my repo?
- How do I hook it into Cobra help consistently?
- How do I ensure help output goes to stdout vs stderr?

The `glaze` main shows one pattern, but it’s not packaged as an “app integration” helper and it’s easy to miss one step.

**What would make this easier:**

- A `help/app` helper package that exposes:
  - `help_app.Setup(rootCmd, opts...)` which:
    - initializes HelpSystem,
    - loads embedded docs FS (optional),
    - wires Cobra help,
    - configures output writer,
    - returns the help system.

### 7) Hidden commands and internal verbs (e.g. `__wrap-service`)

Porting apps often have internal commands that must not pay the cost of:

- help system init,
- dynamic command discovery,
- plugin loading,
- complex middleware parsing,
- etc.

In devctl, `__wrap-service` is a good example: it is an internal wrapper executed under supervision and should be “minimal overhead, minimal side effects”.

**What would make this easier:**

- First-class support for “fast path” commands:
  - an app should be able to declare: “for these commands, skip help/doc loading, skip dynamic discovery, skip config/env loading, and skip heavy middleware”.
- A documented pattern for this in Glazed examples and tutorials.

### 8) Dynamic commands and Glazed command descriptions

Glazed’s `CommandDescription` model expects:

- a (mostly) static verb tree,
- a fixed set of layers per command,
- a consistent parser pipeline.

Dynamic commands (discovered at runtime from plugin handshakes) are a legitimate use case, and Cobra supports it naturally, but it’s unclear how Glazed intends it to work.

Open questions a porter immediately hits:

- How do I attach layers to dynamic commands?
- How do I ensure global flags apply?
- How do I keep help consistent?
- How do I avoid dynamic discovery for built-in verbs and completions?

**What would make this easier:**

- A documented “Dynamic command registration” recipe:
  - how to construct a `CommandDescription` for dynamic commands (parents, name, help),
  - how to share global layers,
  - how to keep them hidden from help/completion until fully resolved (optional),
  - how to handle collisions (two plugins define same verb).

### 9) Flag grouping vs Cobra default help

Glazed adds richer flag grouping via annotations, but the interaction with:

- persistent flags,
- inherited flags,
- hidden flags,
- and Cobra’s default help template

is complex. It’s easy to end up with:

- duplicated flags in help,
- flags that exist but show up under “Other flags” unexpectedly,
- inconsistent “Global flags” sections.

**What would make this easier:**

- A “flag grouping and inheritance” doc page with:
  - how grouping is computed,
  - what counts as “local” vs “inherited” in Cobra,
  - how persistent flags should be grouped.

### 10) Testing the port is underspecified

When porting, you need a systematic way to ensure:

- flag parity (names, defaults, behavior),
- output parity (JSON shape, errors),
- help parity (or at least acceptable help UX),
- completion semantics,
- dynamic behavior (plugins not started for built-ins, etc).

**What would make this easier:**

- A “porting checklist” (some exists in MO-012 docs) turned into a reusable template:
  - `old cobra` vs `new glazed` outputs for `--help`,
  - command-by-command smoke scripts,
  - minimal automated tests around parsing and defaults.

### 11) Practical friction: “I needed to read 6 files to do 1 thing”

To understand how parsing works, I had to jump between:

- `cli` builder code,
- parser/middleware code,
- layer interfaces,
- parameter definitions,
- Cobra flag behaviors,
- help system behaviors.

This is not unique to Glazed, but it is a signal that:

- a higher-level “app builder” abstraction would have high leverage, and
- documentation could better “collapse” the mental model into fewer concepts.

## Concrete improvement proposals (actionable backlog)

This section translates the above into an implementable backlog. The goal is to make it easy to say “yes” to parts of it.

### A) Provide an application builder / template (highest leverage)

Create `glazed/pkg/app` (or similar) with:

- `type App struct { Root *cobra.Command; Help *help.HelpSystem; ... }`
- `NewApp(appName string, opts...)`
- `WithLoggingLayer()`
- `WithHelpSystem(embeddedFS, docRoot string)`
- `WithGlobalLayer(layer layers.ParameterLayer, mode Persistent|Local)`
- `RegisterCommand(cmd cmds.Command, opts...)`
- `RegisterDynamicCommandFactory(func(*cobra.Command) error)` (optional)

And ship:

- an example repository (or `cmd/example-app`) that demonstrates:
  - WriterCommand-only app (no glaze output),
  - a global “repo settings” layer,
  - help system with embedded docs,
  - config/env/default precedence.

### B) Add `ParameterTypeDuration` (+ tests)

Add `ParameterTypeDuration` and support it in:

- Cobra flag registration (as duration),
- parsing from Cobra flags,
- parsing from config/env,
- struct initialization.

This removes a large amount of “stringly-typed” port glue.

### C) Add a “path parameter” story (type + helpers)

Either:

1. Add `ParameterTypePath` and `ParameterTypePathList`, or
2. Provide a reusable normalization layer (recommended minimum):
   - `parameters.WithNormalizePath(…options…)`

Include standard options:

- absolute normalization,
- base directory resolution,
- existence checks,
- file vs dir constraints,
- `~` expansion.

### D) First-class persistent layer support

Make it easy to define one layer and have it show up as:

- root persistent flags,
- parsed values available everywhere,
- and grouped in help.

This should not require writing a custom `CobraParameterLayer` wrapper per app.

### E) Document canonical patterns (reduce ambiguity)

Add docs pages for:

- “Glazed in a Cobra-first app” (writer commands, persistent flags).
- “Precedence model” with concrete examples and debug output.
- “Dynamic command registration pattern”.
- “Fast path internal commands”.

### F) Provide better diagnostics for ports

Add built-in debugging features to the builder (or template), such as:

- `--print-parsed-parameters` already exists (good),
- add `--print-parsed-parameters --include-provenance`,
- add `--print-layer <slug>` to show one layer,
- and/or a `glazed debug` helper in apps (opt-in).

## devctl-specific notes (what was confusing in practice)

The devctl port surfaced specific pain points that likely apply to other real apps:

1. devctl has a “repo context” concept that wants normalization (abs repo root, config default relative to repo root, timeout validation). This maps naturally to a Glazed layer, but the persistent flag story makes it hard to do cleanly.
2. devctl has dynamic commands discovered from plugins. Cobra supports this easily, but it’s unclear how to do “dynamic Glazed commands” while keeping parsing and help consistent.
3. devctl has internal commands that must be “minimal overhead” (no help/doc init). This should be a first-class documented pattern.

## Proposed “golden path” porting workflow (what I wish existed)

If Glazed provided the improvements above, the port workflow could be:

1. Create app via `app.NewApp("devctl")` (logging + help system + docs FS in one call).
2. Define `RepoSettingsLayer` with `ParameterTypeDuration` and `ParameterTypePath`.
3. Add the layer as persistent with one method call.
4. Port each command by implementing `WriterCommand` and using `parsedLayers.InitializeStruct("repo", &RepoSettings{})`.
5. Add dynamic command factory with a documented helper.
6. Validate with the debug print flags and a parity checklist.

That would reduce port friction from “read a lot of framework internals” to “follow the template and fill in command logic”.
