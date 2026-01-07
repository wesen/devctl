---
Title: Comprehensive Fixture Bug Report
Ticket: MO-009-TUI-COMPLETE-FEATURES
Status: active
Topics:
    - bugs
    - testing
DocType: analysis
Intent: short-term
Owners: []
RelatedFiles:
    - scripts/setup-comprehensive-fixture.sh
    - pkg/supervise/supervisor.go
    - pkg/tui/state_watcher.go
    - pkg/tui/models/dashboard_model.go
ExternalSources: []
Summary: "Bug report for issues discovered during comprehensive fixture testing"
LastUpdated: 2026-01-07T04:45:00-05:00
WhatFor: "Track and fix issues found during TUI testing"
WhenToUse: "When fixing these bugs"
---

# Comprehensive Fixture Bug Report

## Issue Summary

Three issues were discovered when testing the TUI with the comprehensive fixture:

| # | Severity | Issue | Status |
|---|----------|-------|--------|
| 1 | **Critical** | "wrapper did not report child start" error | Open |
| 2 | **Medium** | Plugins view shows empty list | Open |
| 3 | **Low** | Dashboard doesn't show pipeline progress | Open |

---

## Issue 1: Wrapper Did Not Report Child Start

### Symptom

When running `devctl up`, the pipeline fails at the "supervise" phase with:
```
Error: wrapper did not report child start
```

### Root Cause

The supervisor (`pkg/supervise/supervisor.go`) expects a **wrapper binary** to be available. The wrapper is responsible for:
1. Starting the actual service process
2. Writing a "ready" file to signal successful start
3. Managing process lifecycle

The code waits up to 2 seconds for the ready file:

```go:200:222:pkg/supervise/supervisor.go
deadline := time.Now().Add(2 * time.Second)
for {
    if _, err := os.Stat(readyPath); err == nil {
        break
    }
    if time.Now().After(deadline) {
        _ = terminatePIDGroup(context.Background(), pid, 1*time.Second)
        return state.ServiceRecord{}, errors.New("wrapper did not report child start")
    }
    time.Sleep(10 * time.Millisecond)
}
```

### Investigation Findings

The wrapper is actually the **devctl binary itself** with a hidden `__wrap-service` command:
- Located in `cmd/devctl/cmds/wrap_service.go`
- Invoked as: `devctl __wrap-service --service foo --cwd /path -- actual-command args...`
- Writes a "ready file" when the child process starts successfully

The `wrapperExe` is set via `os.Executable()`:
```go
wrapperExe, _ := os.Executable()
sup := supervise.New(supervise.Options{..., WrapperExe: wrapperExe})
```

### Possible Causes

1. **Ready file not being written**: The wrapper writes the ready file at line 110 of `wrap_service.go`:
   ```go
   if readyFile != "" {
       _ = os.MkdirAll(filepath.Dir(readyFile), 0o755)
       _ = os.WriteFile(readyFile, []byte(fmt.Sprintf("%d\n", child.Process.Pid)), 0o644)
   }
   ```
   If `child.Start()` fails (line 96), the ready file is never written.

2. **Child process failing to start**: If the actual service command fails (e.g., binary not found), the wrapper exits early.

3. **Timing issue**: The supervisor waits only 2 seconds for the ready file. On slow systems or with cold go build caches, this might not be enough.

### Likely Root Cause

The wrapped command is likely failing to start. The comprehensive plugin passes paths like:
```
command: ["/tmp/devctl-xxx/bin/http-echo", "--port", "34425"]
```

But the error "wrapper did not report child start" indicates the wrapper process started, but never wrote the ready file. This could mean:
- The child process failed to start within the wrapper
- The wrapper itself crashed before writing the ready file

### Debugging Steps

1. Run the wrapper manually to see if it works:
   ```bash
   devctl __wrap-service --service test --cwd /tmp \
     --stdout-log /tmp/test.stdout --stderr-log /tmp/test.stderr \
     --exit-info /tmp/test.exit --ready-file /tmp/test.ready \
     -- /tmp/devctl-xxx/bin/http-echo --port 8080
   ```

2. Check if the ready file directory is writable

3. Add more verbose logging to the wrapper

---

## Issue 2: Plugins View Shows Empty List

### Symptom

The Plugins view (accessible via Tab cycling) shows "No plugins configured" even though the `.devctl.yaml` has 3 plugins defined.

### Root Cause

In `pkg/tui/state_watcher.go`, the `readPlugins()` function checks if the plugin path exists:

```go:141:168:pkg/tui/state_watcher.go
func (w *StateWatcher) readPlugins() []PluginSummary {
    // ...
    for _, p := range cfg.Plugins {
        status := "active"
        // Check if plugin path exists
        pluginPath := p.Path
        if pluginPath != "" && pluginPath[0] != '/' {
            pluginPath = w.RepoRoot + "/" + pluginPath  // BUG: Wrong for commands
        }
        if _, err := os.Stat(pluginPath); err != nil {
            status = "error"  // But plugin is still added
        }
        // ...
    }
}
```

**The bug**: For plugins with `path: python3`, the code:
1. Sees that "python3" doesn't start with "/"
2. Prepends repo root: `/tmp/devctl-xxx/python3`
3. `os.Stat()` fails because that path doesn't exist
4. Sets status to "error" but still adds the plugin

However, the **actual bug** is likely in the config loader or how plugins are read. Let me trace further...

**Wait** - Looking again, the plugin IS being added with status "error". So the plugins list should have 3 items. The issue must be elsewhere.

### Additional Investigation

Check if:
1. `config.LoadOptional()` is successfully loading the config
2. `cfg.Plugins` is being populated correctly
3. The `StateSnapshot.Plugins` is being passed to the TUI correctly
4. The `PluginModel.WithPlugins()` is being called

### Likely Root Cause

The plugin path check is incorrect. For plugins where `path` is a command (like `python3`), not a file path, we should:
- Use `exec.LookPath()` to find the command
- Or skip the existence check for non-path values

### Proposed Fix

```go
func (w *StateWatcher) readPlugins() []PluginSummary {
    // ...
    for _, p := range cfg.Plugins {
        status := "active"
        
        // Only check file existence for actual file paths
        pluginPath := p.Path
        if pluginPath != "" && !isCommand(pluginPath) {
            if pluginPath[0] != '/' {
                pluginPath = filepath.Join(w.RepoRoot, pluginPath)
            }
            if _, err := os.Stat(pluginPath); err != nil {
                status = "error"
            }
        } else if pluginPath != "" {
            // Check if command exists in PATH
            if _, err := exec.LookPath(pluginPath); err != nil {
                status = "error"
            }
        }
        // ...
    }
}

func isCommand(path string) bool {
    // If it contains no slashes, it's likely a command name
    return !strings.Contains(path, "/")
}
```

---

## Issue 3: Dashboard Doesn't Show Pipeline Progress

### Symptom

When `devctl up` is running, pressing Tab to view the Dashboard shows only service status. There's no indication that a pipeline is running or what phase it's in.

### Expected Behavior

The dashboard should show:
- A "Pipeline Running" indicator when a pipeline is active
- Current phase (e.g., "Building...", "Validating...")
- Quick summary of progress

### Current State

The `DashboardModel` only receives `StateSnapshotMsg` and `EventLogAppendMsg`. It doesn't receive any pipeline-related messages.

### Proposed Enhancement

Add a pipeline status section to the dashboard:

```go
type DashboardModel struct {
    // existing...
    pipelineRunning bool
    pipelinePhase   string
    pipelineRunID   string
}

// In RootModel.Update(), forward pipeline messages to dashboard:
case tui.PipelineRunStartedMsg:
    m.dashboard = m.dashboard.WithPipelineStarted(v.Run)
    // existing pipeline update...

case tui.PipelinePhaseStartedMsg:
    m.dashboard = m.dashboard.WithPipelinePhase(v.Event.Phase)
    // existing pipeline update...

case tui.PipelineRunFinishedMsg:
    m.dashboard = m.dashboard.WithPipelineFinished()
    // existing pipeline update...
```

Dashboard View would then render:
```
╭──────────────────────────────────────────────────────────────────────────────╮
│ Pipeline: up                                                       Running   │
│ Phase: build (5.5s)                                                          │
│ ████████████████████░░░░░░░░░░ 65%                                           │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## Priority Order for Fixes

1. **Issue 1 (Critical)**: Without fixing the wrapper issue, services can't start and most TUI features can't be tested.

2. **Issue 2 (Medium)**: Plugins view is broken, but the rest of the TUI works.

3. **Issue 3 (Low)**: Enhancement, not a bug. Pipeline view already works, this is just about showing a summary on dashboard.

---

## Action Items

- [ ] **1.1** Investigate wrapper binary location and setup
- [ ] **1.2** Ensure fixture script builds/includes wrapper if needed
- [ ] **1.3** Document wrapper requirements

- [ ] **2.1** Fix `readPlugins()` to handle command paths correctly
- [ ] **2.2** Use `exec.LookPath()` for command detection
- [ ] **2.3** Test plugins view with fixture

- [ ] **3.1** Add pipeline status to DashboardModel
- [ ] **3.2** Forward pipeline messages to dashboard
- [ ] **3.3** Render pipeline status section in dashboard view

---

## Environment

- **OS**: Linux
- **Test Method**: Comprehensive fixture script
- **TUI Features Tested**: Pipeline view (partial), Plugins view (failed)

---

## Screenshots

The user-provided screenshot shows:
- Pipeline view working (phases, build steps, validation warnings visible)
- Error at supervise phase: "wrapper did not report child start"
- No services started due to the error

