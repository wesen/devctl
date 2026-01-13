# devctl - Dev environment orchestrator

`devctl` turns repo-specific startup knowledge into a repeatable, testable workflow. Your plugins describe what to run; devctl handles ordering, process supervision, state, and logs.

## Features

- NDJSON stdio plugin protocol (v2) with strict, debuggable boundaries.
- Pipeline orchestration: config -> build -> prepare -> validate -> launch -> supervise.
- Process supervision with health checks, PID tracking, and structured logs.
- TUI dashboard for start/stop, logs, events, pipeline, and plugins.
- Dynamic plugin commands (`devctl <command>`) defined by plugins.
- Stream operations for long-running protocol events (`devctl stream start`).
- Mergeable outputs from multiple plugins (priority ordering, strict collisions).
- Built-in smoke tests and fixtures for plugin protocol behavior.
- Companion `log-parse` tool for JS-based log parsing and tagging.

## Installation

Choose one of the following methods (mirroring other go-go-golems CLIs):

### Homebrew
```bash
brew tap go-go-golems/go-go-go
brew install go-go-golems/go-go-go/devctl
```

### apt-get (Debian/Ubuntu)
```bash
echo "deb [trusted=yes] https://apt.fury.io/go-go-golems/ /" | sudo tee /etc/apt/sources.list.d/fury.list
sudo apt-get update
sudo apt-get install devctl
```

### yum (RHEL/CentOS/Fedora)
```bash
sudo bash -c 'cat > /etc/yum.repos.d/fury.repo <<EOF
[fury]
name=Gemfury Private Repo
baseurl=https://yum.fury.io/go-go-golems/
enabled=1
gpgcheck=0
EOF'
sudo yum install devctl
```

### go install
```bash
go install github.com/go-go-golems/devctl/cmd/devctl@latest
```

### Download binaries
Download prebuilt binaries from GitHub Releases.

### Run from source
```bash
git clone https://github.com/go-go-golems/devctl
cd devctl
go run ./cmd/devctl --help
```

## Quick start

1) Create a `.devctl.yaml` at your repo root:

```yaml
plugins:
  - id: myrepo
    path: python3
    args:
      - ./devctl-plugin.py
    priority: 10
```

2) Write a minimal plugin (`devctl-plugin.py`):

```python
#!/usr/bin/env python3
import json
import sys

def emit(obj):
    sys.stdout.write(json.dumps(obj) + "\n")
    sys.stdout.flush()

emit({
    "type": "handshake",
    "protocol_version": "v2",
    "plugin_name": "myrepo",
    "capabilities": {"ops": ["config.mutate", "validate.run", "launch.plan"]},
})

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    req = json.loads(line)
    rid = req.get("request_id", "")
    op = req.get("op", "")

    if op == "config.mutate":
        emit({"type": "response", "request_id": rid, "ok": True,
              "output": {"config_patch": {"set": {"services.api.port": 8080}, "unset": []}}})
    elif op == "validate.run":
        emit({"type": "response", "request_id": rid, "ok": True,
              "output": {"valid": True, "errors": [], "warnings": []}})
    elif op == "launch.plan":
        emit({"type": "response", "request_id": rid, "ok": True,
              "output": {"services": [{"name": "api", "command": ["bash", "-lc", "python3 -m http.server 8080"]}]}})
    else:
        emit({"type": "response", "request_id": rid, "ok": False,
              "error": {"code": "E_UNSUPPORTED", "message": f"unsupported op: {op}"}})
```

Make it executable:

```bash
chmod +x devctl-plugin.py
```

3) Run the standard workflow:

```bash
devctl plugins list
devctl plan
devctl up
devctl status
devctl logs --service api --follow
devctl down
```

## CLI workflow (plan -> up -> observe -> down)

```bash
# Inspect
devctl plugins list
devctl plan

# Start
devctl up
devctl status

# Observe
devctl logs --service api
devctl logs --service api --stderr --follow

# Stop
devctl down
```

Common repo flags (command-local):

- `--repo-root <path>`: repo root (defaults to cwd)
- `--config <file>`: config file (defaults to `.devctl.yaml`)
- `--timeout <dur>`: per-op timeout (default `30s`)
- `--dry-run`: best-effort no side effects
- `--strict`: error on config/service collisions

## The pipeline (phases and ownership)

```
config.mutate -> build.run -> prepare.run -> validate.run -> launch.plan -> supervise
```

- Plugins compute facts (config, validation, steps, services).
- devctl runs the lifecycle (ordering, timeouts, processes, logs, health).

## TUI (always-on dashboard)

```bash
devctl tui
```

Key bindings (highlights):

- `tab`: switch views (Dashboard, Events, Pipeline, Plugins)
- `u`: start environment
- `d`: stop environment
- `r`: restart environment
- `l` / `enter`: open logs for selected service
- `?`: help overlay
- `q`: quit

Full reference: `devctl help devctl-tui-guide`

### Screenshots

![devctl TUI dashboard](docs/screenshots/devctl-tui-dashboard.png)
![devctl TUI pipeline](docs/screenshots/devctl-tui-pipeline.png)
![devctl TUI plugins](docs/screenshots/devctl-tui-plugins.png)

## Plugins and protocol (NDJSON v2)

Hard rules:

- stdout is protocol only (one JSON object per line).
- stderr is for humans.
- the first frame is always a handshake.

Minimal handshake:

```json
{"type":"handshake","protocol_version":"v2","plugin_name":"example","capabilities":{"ops":["config.mutate","validate.run","launch.plan"]}}
```

Useful docs:

- `devctl help devctl-user-guide`
- `devctl help devctl-scripting-guide`
- `devctl help devctl-plugin-authoring`
- `docs/plugin-authoring.md`

## Streams and dynamic commands

- Plugins can expose stream ops; run them via `devctl stream start --op <name>`.
- Plugins can expose custom commands by advertising `command.run` in the handshake.

Example command execution:

```bash
# Plugin exposes a command named "db-reset"
devctl db-reset -- --force
```

## State and logs

devctl keeps per-repo state in `.devctl/`:

```
.devctl/
  state.json
  logs/
    api.stdout.log
    api.stderr.log
    api.exit.json
```

Delete `.devctl/` to reset local state.
Add `.devctl/` to `.gitignore`.

## log-parse (JS log parsing companion)

The repo includes a companion CLI for JavaScript-driven log parsing and tagging.

```bash
# From devctl/ root
cat examples/log-parse/sample-json-lines.txt | \
  go run ./cmd/log-parse --module examples/log-parse/parser-json.js
```

Full guide: `devctl help log-parse-guide`

## Help and docs

- List help topics: `devctl help --all`
- Open the interactive help TUI: `devctl help --ui`

## Shell completion

Static completion scripts are available via cobra:

```bash
# Bash
devctl completion bash | sudo tee /etc/bash_completion.d/devctl >/dev/null
# Zsh
devctl completion zsh > ~/.zfunc/_devctl
# Fish
devctl completion fish > ~/.config/fish/completions/devctl.fish
# PowerShell
devctl completion powershell | Out-String | Invoke-Expression
```

## Smoke tests (dev-only)

Use the built-in smoke tests to validate protocol behavior and supervision:

```bash
go run ./cmd/devctl dev smoketest e2e
go run ./cmd/devctl dev smoketest supervise
go run ./cmd/devctl dev smoketest logs
go run ./cmd/devctl dev smoketest failures
```

## Development

- Go 1.25+
- Build: `go build ./...`
- Test: `go test ./...`
- Lint: `golangci-lint run -v` or `make lint`

## License

MIT
