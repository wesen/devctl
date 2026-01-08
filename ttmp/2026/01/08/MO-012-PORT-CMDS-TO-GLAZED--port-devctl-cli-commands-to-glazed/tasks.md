# Tasks

## TODO

- [ ] Root: restructure devctl main.go to Glazed style (logging layer + help system + BuildCobraCommand)
- [x] Help: initialize HelpSystem and load devctl docs into it (pkg/doc)
- [x] Help: wire help_cmd.SetupCobraRootCommand and ensure cobra help output is correct
- [x] Repo flags: make repo-root/config/strict/dry-run/timeout command-local (no persistent root flags)
- [ ] Layer: add normalization+validation tests for RepoSettings (abs paths, timeout>0)
- [x] Port: status command to Glazed (tail-lines flag, JSON output)
- [ ] Port: plan command to Glazed (repo layer, JSON output, no-plugins {} behavior)
- [x] Port: plugins list command to Glazed (handshake inventory JSON)
- [ ] Port: logs command to Glazed (service required, follow mode)
- [ ] Port: down command to Glazed (stop+remove state)
- [ ] Port: up command to Glazed (flags, interactive prompts, dry-run JSON)
- [ ] Port: stream start command to Glazed (input-json/input-file validation, raw JSON mode)
- [ ] Port: tui command wrapper to Glazed (refresh/alt-screen/debug-logs flags)
- [x] Decide: keep smoketest* as Cobra-only vs port to Glazed; document decision
- [ ] Internal: keep __wrap-service isolated from startup logic; verify no Glazed init breaks it
- [ ] Dynamic commands: design how handshake-advertised commands will be registered in Glazed root
- [ ] Dynamic commands: ensure built-in verbs do not start plugins; preserve completion semantics
- [ ] Docs: add devctl help topics for the Glazed-ported CLI surface and flags
- [ ] Validation: run exhaustive matrix on fixture plugins/repos (no flag parity requirement)
- [x] Smoketest: move under hidden group 'dev smoketest' (no top-level smoketest* verbs)
- [x] Smoketest: refactor cmd layout to groups (cmd/devctl/cmds/dev/root.go + cmd/devctl/cmds/dev/smoketest/root.go + subcommands)
- [x] Smoketest: update all call sites (CI workflows, pkg/doc, scripts) from 'smoketest-*' to 'dev smoketest ...'
- [x] Smoketest: add backwards-compat decision (no shim vs temporary aliases) + document in help
- [x] Validation: cd devctl && GOWORK=off go test ./... -count=1
- [x] Validation: run smoketests (GOWORK=off go run ./cmd/devctl dev smoketest [root|failures|logs|supervise|e2e])
- [ ] Validation: repo context flags after verb (status/plan/plugins list/logs/stream start/up/down) with --repo-root and --config relative path cases
- [ ] Validation: fixture plugin 'command' exposes dynamic commands; ensure commands appear only when handshake.capabilities.commands is non-empty
- [ ] Validation: dynamic command discovery does not start plugins for built-in verbs; preserves completion behavior
- [ ] Validation: timeout behavior across ops using fixture plugin 'timeout' (no hangs; context deadline errors are actionable)
- [ ] Validation: stdout contamination failures using fixtures 'noisy-handshake' and 'noisy-after-handshake'
- [ ] Validation: streams behavior using fixtures 'stream' and 'streams-only-never-respond' (capability gating; no indefinite waits)
- [ ] Validation: pipeline behavior using fixture plugin 'pipeline' (config/build/prepare/validate/launch)
- [ ] Validation: validate and launch failures using fixtures 'validate-passfail' and 'launch-fail'
- [ ] Validation: logs follow + cancellation promptness (fixture plugin or smoketest logs)
- [ ] UI Validation (tmux): run devctl tui against fixture/e2e repo; verify basic navigation + log view + restart (defer while 'no TUI testing' instruction active)
