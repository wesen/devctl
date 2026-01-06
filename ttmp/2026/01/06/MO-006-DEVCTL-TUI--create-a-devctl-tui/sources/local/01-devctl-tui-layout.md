---
Title: "DevCtl TUI Design - ASCII Screenshots"
Ticket: MO-006-DEVCTL-TUI
Status: active
Topics:
  - backend
  - ui-components
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
  - /tmp/devctl-tui.md
Summary: "Imported ASCII mockups for a devctl TUI layout; used as the baseline reference for screen structure and keybindings."
LastUpdated: 2026-01-06T15:24:23-05:00
WhatFor: "Baseline layout mockups for the devctl TUI."
WhenToUse: "When implementing or reviewing the TUI screen layout and interactions."
---

# DevCtl TUI Design - ASCII Screenshots

## 1. Main Dashboard (devctl up - running state)

```
┌─ DevCtl - moments ──────────────────────────────────────────────────── [↑/↓/q] ─┐
│                                                                                   │
│ ● System Status: Running                                      Uptime: 2h 34m 12s │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ Services (3)                                                    [l] logs [r] restart │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✓ backend           Running    PID 12847   Healthy   CPU 12%   MEM 245MB   │ │
│ │ ✓ frontend          Running    PID 12849   Healthy   CPU  3%   MEM  89MB   │ │
│ │ ✓ postgres          Running    PID 12843   Healthy   CPU  1%   MEM 156MB   │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Recent Events (5)                                             [f] follow [c] clear │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ 14:23:45  backend    ℹ  Request completed in 234ms                          │ │
│ │ 14:23:42  frontend   ℹ  Asset compilation complete                          │ │
│ │ 14:23:40  backend    ℹ  Connected to database                               │ │
│ │ 14:23:38  postgres   ℹ  Database ready for connections                      │ │
│ │ 14:23:35  system     ✓  All health checks passed                            │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Plugins (3 active)                                                                │
│  • moments-config   (priority: 10)   ops: 2   commands: 0                        │
│  • moments-build    (priority: 15)   ops: 3   commands: 1                        │
│  • moments-launch   (priority: 20)   ops: 4   commands: 2                        │
│                                                                                   │
│ [s] services [p] plugins [e] events [h] help [q] quit                            │
└───────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Service Detail View (pressed 'l' on backend)

```
┌─ Service: backend ────────────────────────────────────────────────── [ESC] back ─┐
│                                                                                   │
│ Status: ✓ Running                                           Started: 2h 34m ago  │
│ Health: Healthy (http://localhost:8083/rpc/v1/health)      Last check: 2s ago    │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ Process Info                                                                      │
│  PID:         12847                                                               │
│  Command:     go run ./cmd/moments-server serve                                   │
│  Working Dir: /home/user/moments/backend                                          │
│  CPU:         12.3%                                                               │
│  Memory:      245.8 MB                                                            │
│                                                                                   │
│ Environment                                                                       │
│  PORT=8083  DB_URL=postgresql://localhost:5432/moments  ENV=development           │
│                                                                                   │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ Live Logs                                                    [↑/↓] scroll [f] find │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ 14:23:45.234  INFO  Request GET /api/moments?limit=20                       │ │
│ │ 14:23:45.456  INFO  Query completed in 234ms (45 rows)                      │ │
│ │ 14:23:42.123  INFO  WebSocket connection established (client: web-abc123)   │ │
│ │ 14:23:40.891  INFO  Database connection pool initialized (max: 10)          │ │
│ │ 14:23:40.567  INFO  Starting HTTP server on :8083                           │ │
│ │ 14:23:40.234  INFO  Configuration loaded from backend/.env                  │ │
│ │ 14:23:39.901  INFO  Service starting...                                     │ │
│ │                                                                              │ │
│ │                                                                              │ │
│ │                                                                              │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ [r] restart [s] stop [k] kill [d] detach [ESC] back                              │
└───────────────────────────────────────────────────────────────────────────────────┘
```

## 3. Startup Sequence (devctl up - building phase)

```
┌─ DevCtl - moments ──────────────────────────────────────────────────────────────┐
│                                                                                   │
│ ⚙ System Status: Starting                                    Phase: Build [2/5]  │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ Pipeline Progress                                                                 │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✓ Config Mutation     3 plugins applied              0.2s                   │ │
│ │ ▶ Build               Running...                     5.3s                   │ │
│ │   Prepare             Pending                                                │ │
│ │   Validate            Pending                                                │ │
│ │   Launch              Pending                                                │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Build Steps (2/3 complete)                                                        │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✓ backend-deps        pnpm install                   2.1s                   │ │
│ │ ✓ frontend-deps       pnpm install                   3.2s                   │ │
│ │ ▶ backend-compile     go build ./cmd/moments-server  5.3s (running)         │ │
│ │   frontend-build      pnpm build                          (pending)         │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Live Output                                                                       │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ [backend-compile] Building target: cmd/moments-server                       │ │
│ │ [backend-compile] Compiling 247 packages...                                 │ │
│ │ [backend-compile] ██████████████████░░░░░░░░░░ 65%                          │ │
│ │                                                                              │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Applied Config Patches                                                            │
│  • services.backend.port → 8083                          (moments-config)        │
│  • services.postgres.image → postgres:15-alpine         (moments-config)        │
│  • build.cache_enabled → true                            (moments-build)         │
│                                                                                   │
│ [Ctrl+C] cancel                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
```

## 4. Plugin List View (pressed 'p')

```
┌─ Plugins ──────────────────────────────────────────────────────── [ESC] back ────┐
│                                                                                   │
│ Discovered Plugins (3)                                      Source: .devctl.yaml  │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ ┌─ moments-config ───────────────────────────────────────────────────────────┐  │
│ │ Status:   ✓ Active                                    Priority: 10         │  │
│ │ Path:     ./moments/plugins/devctl-config.sh                                │  │
│ │ Protocol: v1                                                                │  │
│ │                                                                             │  │
│ │ Capabilities:                                                               │  │
│ │   Ops:      config.mutate, validate.run                                     │  │
│ │   Streams:  (none)                                                          │  │
│ │   Commands: (none)                                                          │  │
│ │                                                                             │  │
│ │ Declares:                                                                   │  │
│ │   side_effects: none        idempotent: true                                │  │
│ └─────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                   │
│ ┌─ moments-build ────────────────────────────────────────────────────────────┐  │
│ │ Status:   ✓ Active                                    Priority: 15         │  │
│ │ Path:     ./moments/plugins/devctl-build.sh                                 │  │
│ │ Protocol: v1                                                                │  │
│ │                                                                             │  │
│ │ Capabilities:                                                               │  │
│ │   Ops:      build.run, prepare.run, validate.run                            │  │
│ │   Streams:  (none)                                                          │  │
│ │   Commands: clean-build                                                     │  │
│ └─────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                   │
│ ┌─ moments-launch ───────────────────────────────────────────────────────────┐  │
│ │ Status:   ✓ Active                                    Priority: 20         │  │
│ │ Path:     ./moments/plugins/devctl-launch.sh                                │  │
│ │ Protocol: v1                                                                │  │
│ │                                                                             │  │
│ │ Capabilities:                                                               │  │
│ │   Ops:      launch.plan, logs.list, logs.follow                             │  │
│ │   Streams:  logs.follow                                                     │  │
│ │   Commands: db-reset, db-migrate                                            │  │
│ └─────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                   │
│ [i] inspect [d] disable [e] enable [r] reload [ESC] back                         │
└───────────────────────────────────────────────────────────────────────────────────┘
```

## 5. Error State (validation failure)

```
┌─ DevCtl - moments ──────────────────────────────────────────────────────────────┐
│                                                                                   │
│ ✗ System Status: Failed                                   Phase: Validate [4/5]  │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ Validation Errors (2)                                                             │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✗ EPORT_IN_USE                                          (moments-config)    │ │
│ │   Port 8083 is already in use                                               │ │
│ │   Fix: Run 'devctl stop' or choose another port in config                   │ │
│ │                                                                              │ │
│ │ ✗ EENVMISSING                                           (moments-build)     │ │
│ │   Required environment variable not set: DATABASE_URL                       │ │
│ │   Fix: Add DATABASE_URL to backend/.env                                     │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Validation Warnings (1)                                                           │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ⚠ WDEPRECATED                                           (moments-build)     │ │
│ │   Node.js version 16 is deprecated, upgrade to 18+                          │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Pipeline Status                                                                   │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✓ Config Mutation     3 plugins applied              0.2s                   │ │
│ │ ✓ Build               4 steps completed              8.7s                   │ │
│ │ ✓ Prepare             2 steps completed              1.3s                   │ │
│ │ ✗ Validate            Failed with 2 errors           0.5s                   │ │
│ │ ⊘ Launch              Skipped due to errors                                 │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Actions                                                                           │
│  [r] retry   [f] fix manually   [l] view logs   [q] quit                         │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
```

## 6. Multi-Service Log Stream View (pressed 'e' for events)

```
┌─ Live Events ────────────────────────────────────────────────────── [ESC] back ──┐
│                                                                                   │
│ Following: All Services                            [f] filter [1-9] select service │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ 14:34:12.234  [backend]    INFO  POST /api/moments                          │ │
│ │ 14:34:12.156  [postgres]   INFO  Query: SELECT * FROM moments LIMIT 20      │ │
│ │ 14:34:11.987  [frontend]   INFO  HMR update: components/MomentList.tsx      │ │
│ │ 14:34:10.543  [backend]    INFO  WebSocket message received (123 bytes)     │ │
│ │ 14:34:09.321  [backend]    WARN  Slow query detected: 1.2s                  │ │
│ │ 14:34:08.765  [frontend]   INFO  Asset compiled: bundle.js (2.3MB)          │ │
│ │ 14:34:07.432  [system]     INFO  Health check passed (all services)         │ │
│ │ 14:34:05.210  [backend]    INFO  Cache hit: user-profile-456                │ │
│ │ 14:34:04.876  [postgres]   INFO  Connection pool: 3/10 in use               │ │
│ │ 14:34:03.543  [frontend]   INFO  Request GET /moments                       │ │
│ │ 14:34:02.109  [backend]    INFO  Session created: sess_abc123def456         │ │
│ │ 14:34:01.654  [frontend]   INFO  Component mounted: <MomentList>            │ │
│ │ 14:34:00.321  [backend]    INFO  Request completed in 45ms                  │ │
│ │ 14:33:59.987  [system]     INFO  Memory usage: backend=245MB frontend=89MB  │ │
│ │ 14:33:58.543  [postgres]   INFO  Checkpoint completed                       │ │
│ │ 14:33:57.210  [backend]    DEBUG Auth token validated                       │ │
│ │ 14:33:56.876  [frontend]   DEBUG React render cycle: 12ms                   │ │
│ │ 14:33:55.321  [backend]    INFO  Database query: 234ms                      │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Filters: ● backend  ● frontend  ● postgres  ● system      [space] toggle         │
│ Levels:  ● DEBUG   ● INFO   ● WARN   ● ERROR              [l] level menu         │
│                                                                                   │
│ Stats: 1,247 events (18/sec)   Buffer: 500 lines   Dropped: 0                    │
│                                                                                   │
│ [p] pause [c] clear [s] save [/] search [ESC] back                               │
└───────────────────────────────────────────────────────────────────────────────────┘
```
---

```yaml
# DevCtl TUI - Bubbletea Architecture

models:
  # Root model - coordinates all views
  root:
    fields:
      - current_view: ViewType  # dashboard | service_detail | startup | plugin_list | events | command_palette
      - dashboard: DashboardModel
      - service_detail: ServiceDetailModel
      - startup: StartupModel
      - width: int
      - height: int
    
  # Dashboard (Main view)
  dashboard:
    fields:
      - services: []ServiceStatus
      - recent_events: []Event  # ring buffer, max 5
      - plugins: []PluginSummary
      - uptime: Duration
      - selected_service: int  # -1 for none
      - event_scroll_offset: int
    update_on:
      - ServiceStatusUpdate
      - NewEvent
      - UptimeTick
      - KeyPress
    
  # Service detail view
  service_detail:
    fields:
      - service: ServiceStatus
      - process_info: ProcessInfo
      - health_info: HealthInfo
      - logs: []LogLine  # ring buffer
      - log_viewport: viewport.Model  # bubbletea viewport
      - log_scroll_offset: int
      - follow_mode: bool
    update_on:
      - ServiceStatusUpdate
      - NewLogLine
      - HealthCheckUpdate
      - ProcessStatsUpdate
      - KeyPress
      
  # Startup sequence view
  startup:
    fields:
      - pipeline: PipelineState
      - current_phase: PhaseType  # config_mutation | build | prepare | validate | launch
      - phases: []PhaseStatus
      - build_steps: []BuildStepStatus
      - live_output: []string  # ring buffer
      - output_viewport: viewport.Model
      - config_patches: []ConfigPatch
      - progress: ProgressState
    update_on:
      - PhaseStarted
      - PhaseCompleted
      - PhaseFailed
      - BuildStepStarted
      - BuildStepCompleted
      - BuildStepProgress
      - LiveOutputLine
      - ConfigPatchApplied
      - KeyPress

# Messages (Msg types for Update())

messages:
  # Time-based
  tick:
    fields:
      - timestamp: time.Time
    description: "Periodic tick for uptime, animations"
    
  uptime_tick:
    fields:
      - uptime: Duration
    description: "Update system uptime display"
    
  # Service lifecycle
  service_status_update:
    fields:
      - service_name: string
      - status: ServiceStatus
    description: "Service state changed (starting/running/failed/stopped)"
    
  process_stats_update:
    fields:
      - service_name: string
      - pid: int
      - cpu_percent: float
      - memory_mb: int
    description: "Process resource usage update"
    
  health_check_update:
    fields:
      - service_name: string
      - health: HealthStatus  # unknown | ok | unhealthy
      - last_check: time.Time
    description: "Health check result"
    
  # Logging
  new_log_line:
    fields:
      - service_name: string
      - timestamp: time.Time
      - level: string  # debug | info | warn | error
      - message: string
    description: "New log line from service"
    
  new_event:
    fields:
      - timestamp: time.Time
      - source: string  # service name or "system"
      - level: string
      - message: string
    description: "New event for dashboard recent events"
    
  # Pipeline (startup sequence)
  phase_started:
    fields:
      - phase: PhaseType
      - timestamp: time.Time
    description: "Pipeline phase started"
    
  phase_completed:
    fields:
      - phase: PhaseType
      - duration_ms: int
      - success: bool
    description: "Pipeline phase completed"
    
  phase_failed:
    fields:
      - phase: PhaseType
      - error: Error
    description: "Pipeline phase failed"
    
  build_step_started:
    fields:
      - step_name: string
      - command: string
    description: "Build step started"
    
  build_step_completed:
    fields:
      - step_name: string
      - duration_ms: int
      - success: bool
    description: "Build step completed"
    
  build_step_progress:
    fields:
      - step_name: string
      - percent: int  # 0-100
      - message: string
    description: "Build step progress update"
    
  live_output_line:
    fields:
      - source: string  # step name
      - line: string
    description: "Live output from build/prepare step"
    
  config_patch_applied:
    fields:
      - plugin: string
      - patch: ConfigPatch
    description: "Config patch applied during mutation phase"
    
  # Navigation
  view_changed:
    fields:
      - from: ViewType
      - to: ViewType
      - context: map[string]any  # e.g., service_name for detail view
    description: "User navigated to different view"
    
  service_selected:
    fields:
      - service_name: string
    description: "User selected service (enter detail view)"
    
  # User input
  key_press:
    fields:
      - key: string  # "up", "down", "l", "q", "esc", etc.
    description: "Raw key press (tea.KeyMsg wrapper)"
    
  # Window
  window_size:
    fields:
      - width: int
      - height: int
    description: "Terminal resize (tea.WindowSizeMsg wrapper)"

# Data structures

types:
  view_type:
    enum: [dashboard, service_detail, startup, plugin_list, events, command_palette]
    
  phase_type:
    enum: [config_mutation, build, prepare, validate, launch]
    
  service_status:
    fields:
      - name: string
      - pid: int
      - state: string  # starting | running | failed | stopped
      - health: string  # unknown | ok | unhealthy
      - cpu_percent: float
      - memory_mb: int
      - started_at: time.Time
      
  phase_status:
    fields:
      - phase: PhaseType
      - state: string  # pending | running | completed | failed
      - duration_ms: int
      - plugins_count: int  # for config_mutation
      
  build_step_status:
    fields:
      - name: string
      - command: string
      - state: string  # pending | running | completed | failed
      - duration_ms: int
      - progress_percent: int
      
  config_patch:
    fields:
      - plugin: string
      - key: string  # dotted path
      - value: string
      
  progress_state:
    fields:
      - current_phase: int  # 1-5
      - total_phases: int  # 5
      - current_step: int
      - total_steps: int
      - percent: int  # overall 0-100

# Command flow (Update() logic)

update_flows:
  dashboard_to_service_detail:
    trigger: key_press("l") on selected service
    actions:
      - emit: ServiceSelected{service_name}
      - emit: ViewChanged{from: dashboard, to: service_detail}
      - model: transition to service_detail view
      - cmd: start log follow subscription
      
  startup_phase_progression:
    trigger: PhaseCompleted
    actions:
      - model: update phases[i].state = completed
      - model: increment current_phase
      - emit: PhaseStarted{next_phase}
      - cmd: start next phase pipeline call
      
  service_logs_streaming:
    trigger: NewLogLine
    actions:
      - model: append to logs ring buffer
      - model: scroll viewport if follow_mode
      - model: update viewport content
      
  build_step_live_output:
    trigger: LiveOutputLine
    actions:
      - model: append to live_output ring buffer
      - model: update viewport content
      - model: auto-scroll to bottom

# Subscriptions (tea.Cmd producers)

subscriptions:
  uptime_ticker:
    interval: 1s
    produces: UptimeTick
    active_when: services running
    
  service_stats_poller:
    interval: 2s
    produces: ProcessStatsUpdate (per service)
    active_when: services running
    
  health_check_poller:
    interval: 5s
    produces: HealthCheckUpdate (per service)
    active_when: services running
    
  log_stream:
    type: streaming
    produces: NewLogLine
    active_when: service running OR service_detail view
    implementation: goroutine reading from supervisor
    
  pipeline_executor:
    type: one-shot per phase
    produces: PhaseStarted | PhaseCompleted | PhaseFailed | BuildStepStarted | etc.
    active_when: startup in progress
    implementation: engine.Pipeline calls with callback Cmd producers
    
  live_output_stream:
    type: streaming during build/prepare
    produces: LiveOutputLine
    active_when: build or prepare phase running
    implementation: goroutine reading from exec.Cmd stdout/stderr

# View rendering (View() -> string)

views:
  dashboard:
    layout:
      - header: status + uptime (lipgloss style)
      - services_box: lipgloss bordered table
      - events_box: lipgloss bordered list (recent 5)
      - plugins_summary: lipgloss list
      - footer: keybindings
    conditional:
      - selected_service != -1: highlight service row
      
  service_detail:
    layout:
      - header: service name + status + health
      - process_info_box: lipgloss table
      - env_vars: lipgloss key-value list
      - separator
      - logs_viewport: bubbletea viewport component
      - footer: keybindings
      
  startup:
    layout:
      - header: status + current phase
      - pipeline_progress: lipgloss list with icons (✓ ▶  )
      - build_steps: lipgloss list with progress bars
      - live_output_viewport: bubbletea viewport
      - config_patches: lipgloss list
      - footer: cancel keybinding
```

---

```yaml
# DevCtl Event Architecture - Watermill + Protobuf

# Watermill topology

pubsub:
  implementation: watermill-io/watermill (GoChannel for local, could be NATS for remote)
  marshaler: protobuf (with json encoding for payload any fields)
  
routers:
  supervisor_router:
    description: "Publishes service lifecycle, health, logs"
    subscribers:
      - topic: cmd.service.start
        handler: supervisor.HandleStartService
      - topic: cmd.service.stop
        handler: supervisor.HandleStopService
      - topic: cmd.service.restart
        handler: supervisor.HandleRestartService
    publishers:
      - service.status.changed
      - service.health.updated
      - service.logs.line
      - service.process.stats
      
  pipeline_router:
    description: "Publishes pipeline/build events during startup"
    subscribers:
      - topic: cmd.pipeline.run
        handler: pipeline.HandleRun
      - topic: cmd.pipeline.cancel
        handler: pipeline.HandleCancel
    publishers:
      - pipeline.phase.started
      - pipeline.phase.completed
      - pipeline.phase.failed
      - pipeline.build_step.started
      - pipeline.build_step.completed
      - pipeline.build_step.progress
      - pipeline.live_output
      - pipeline.config_patch.applied
      
  tui_router:
    description: "Bubbletea program subscribes to events, publishes commands"
    subscribers:
      - topic: service.status.changed
        handler: tui.HandleServiceStatus -> tea.Cmd
      - topic: service.health.updated
        handler: tui.HandleHealthUpdate -> tea.Cmd
      - topic: service.logs.line
        handler: tui.HandleLogLine -> tea.Cmd
      - topic: service.process.stats
        handler: tui.HandleProcessStats -> tea.Cmd
      - topic: pipeline.phase.started
        handler: tui.HandlePhaseStarted -> tea.Cmd
      - topic: pipeline.phase.completed
        handler: tui.HandlePhaseCompleted -> tea.Cmd
      - topic: pipeline.phase.failed
        handler: tui.HandlePhaseFailed -> tea.Cmd
      - topic: pipeline.build_step.started
        handler: tui.HandleBuildStepStarted -> tea.Cmd
      - topic: pipeline.build_step.completed
        handler: tui.HandleBuildStepCompleted -> tea.Cmd
      - topic: pipeline.build_step.progress
        handler: tui.HandleBuildStepProgress -> tea.Cmd
      - topic: pipeline.live_output
        handler: tui.HandleLiveOutput -> tea.Cmd
      - topic: pipeline.config_patch.applied
        handler: tui.HandleConfigPatch -> tea.Cmd
    publishers:
      - cmd.service.start
      - cmd.service.stop
      - cmd.service.restart
      - cmd.pipeline.run
      - cmd.pipeline.cancel
      - cmd.logs.follow
      - cmd.logs.unfollow

# Protobuf message definitions (proto3)

proto_messages:
  # Envelopes
  Envelope:
    fields:
      - id: string  # uuid
      - timestamp: google.protobuf.Timestamp
      - correlation_id: string  # for request/response tracking
      - payload: google.protobuf.Any  # actual message type
      
  # Commands (cmd.*)
  ServiceStartCmd:
    fields:
      - service_name: string
      - config: google.protobuf.Struct  # json config
      
  ServiceStopCmd:
    fields:
      - service_name: string
      - force: bool
      
  ServiceRestartCmd:
    fields:
      - service_name: string
      
  PipelineRunCmd:
    fields:
      - config: google.protobuf.Struct
      - dry_run: bool
      - phases: []string  # empty = all
      
  PipelineCancelCmd:
    fields:
      - reason: string
      
  LogsFollowCmd:
    fields:
      - service_name: string
      - since: google.protobuf.Duration  # e.g., -5m
      
  LogsUnfollowCmd:
    fields:
      - service_name: string
      
  # Events (service.*)
  ServiceStatusChanged:
    fields:
      - service_name: string
      - state: ServiceState  # enum: STARTING | RUNNING | FAILED | STOPPED
      - pid: int32
      - started_at: google.protobuf.Timestamp
      - error: string  # if state=FAILED
      
  ServiceHealthUpdated:
    fields:
      - service_name: string
      - health: HealthStatus  # enum: UNKNOWN | HEALTHY | UNHEALTHY
      - last_check: google.protobuf.Timestamp
      - check_type: string  # tcp | http
      - details: google.protobuf.Struct
      
  ServiceLogLine:
    fields:
      - service_name: string
      - timestamp: google.protobuf.Timestamp
      - level: LogLevel  # enum: DEBUG | INFO | WARN | ERROR
      - message: string
      - fields: google.protobuf.Struct
      
  ServiceProcessStats:
    fields:
      - service_name: string
      - pid: int32
      - cpu_percent: float
      - memory_mb: int64
      - timestamp: google.protobuf.Timestamp
      
  # Events (pipeline.*)
  PipelinePhaseStarted:
    fields:
      - phase: PipelinePhase  # enum: CONFIG_MUTATION | BUILD | PREPARE | VALIDATE | LAUNCH
      - timestamp: google.protobuf.Timestamp
      
  PipelinePhaseCompleted:
    fields:
      - phase: PipelinePhase
      - duration_ms: int64
      - success: bool
      - plugins_count: int32  # for config_mutation
      
  PipelinePhaseFailed:
    fields:
      - phase: PipelinePhase
      - error: Error
      
  PipelineBuildStepStarted:
    fields:
      - step_name: string
      - command: string
      - timestamp: google.protobuf.Timestamp
      
  PipelineBuildStepCompleted:
    fields:
      - step_name: string
      - duration_ms: int64
      - success: bool
      - error: string
      
  PipelineBuildStepProgress:
    fields:
      - step_name: string
      - percent: int32  # 0-100
      - message: string
      
  PipelineLiveOutput:
    fields:
      - source: string  # step name
      - line: string
      - stream: StreamType  # enum: STDOUT | STDERR
      
  PipelineConfigPatchApplied:
    fields:
      - plugin: string
      - key: string  # dotted path
      - value: google.protobuf.Struct
      
  # Shared types
  Error:
    fields:
      - code: string
      - message: string
      - details: google.protobuf.Struct
      
  # Enums
  ServiceState:
    values: [STARTING, RUNNING, FAILED, STOPPED]
    
  HealthStatus:
    values: [UNKNOWN, HEALTHY, UNHEALTHY]
    
  LogLevel:
    values: [DEBUG, INFO, WARN, ERROR]
    
  PipelinePhase:
    values: [CONFIG_MUTATION, BUILD, PREPARE, VALIDATE, LAUNCH]
    
  StreamType:
    values: [STDOUT, STDERR]

# Topic naming convention

topics:
  pattern: "{domain}.{entity}.{action}"
  examples:
    commands:
      - cmd.service.start
      - cmd.service.stop
      - cmd.service.restart
      - cmd.pipeline.run
      - cmd.pipeline.cancel
      - cmd.logs.follow
      - cmd.logs.unfollow
    events:
      - service.status.changed
      - service.health.updated
      - service.logs.line
      - service.process.stats
      - pipeline.phase.started
      - pipeline.phase.completed
      - pipeline.phase.failed
      - pipeline.build_step.started
      - pipeline.build_step.completed
      - pipeline.build_step.progress
      - pipeline.live_output
      - pipeline.config_patch.applied

# Watermill <-> Bubbletea bridge

bridge:
  pattern: "Adapter converts watermill.Message to tea.Cmd"
  
  implementation:
    WatermillSubscriber:
      description: "Goroutine subscribes to topics, produces tea.Cmd"
      pseudocode: |
        func Subscribe(topics []string) tea.Cmd {
          return func() tea.Msg {
            msg := <-subscriber.Output() // blocking
            return WatermillMsg{msg}
          }
        }
        
    TuiModel:
      fields:
        - pubsub: *watermill.PubSub
        - publisher: *watermill.Publisher
      methods:
        - PublishCommand(topic string, payload proto.Message) tea.Cmd
        
    Update:
      pseudocode: |
        func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
          switch msg := msg.(type) {
          case WatermillMsg:
            return m.handleWatermillMsg(msg.Message)
          case KeyMsg:
            if msg.String() == "l" {
              return m, m.PublishCommand("cmd.logs.follow", &LogsFollowCmd{...})
            }
          }
        }
        
  batching:
    description: "High-frequency events (logs, stats) batched before tea.Cmd"
    implementation: |
      LogBuffer:
        - accumulate ServiceLogLine for 100ms
        - produce single tea.Cmd with []ServiceLogLine
        
# WebSocket / WebUI extension

websocket:
  description: "Same watermill topics, different subscriber"
  
  ws_router:
    subscribers:
      - service.status.changed -> broadcast to all ws clients
      - service.health.updated -> broadcast to all ws clients
      - service.logs.line -> broadcast to subscribed clients
      - service.process.stats -> broadcast to all ws clients
      - pipeline.* -> broadcast to all ws clients
    publishers:
      - cmd.* from ws client actions
      
  protocol:
    format: json over websocket (protobuf -> json)
    frame:
      type: string  # "event" | "command"
      topic: string
      payload: object  # json-encoded protobuf
      correlation_id: string
      
  client_subscriptions:
    pattern: "Client sends subscribe/unsubscribe commands"
    messages:
      - type: subscribe, topics: ["service.logs.line"], filter: {service_name: "backend"}
      - type: unsubscribe, topics: ["service.logs.line"]

# Implementation packages

packages:
  glazed/pkg/devctl/events:
    files:
      - proto/messages.proto
      - proto/messages.pb.go  # generated
      - pubsub.go  # watermill setup
      - topics.go  # topic constants
      - marshaler.go  # protobuf + json marshaler
      
  glazed/pkg/devctl/bridge:
    files:
      - tui_adapter.go  # watermill -> tea.Cmd
      - publisher.go  # tea.Cmd -> watermill publish helpers
      
  glazed/pkg/devctl/websocket:
    files:
      - server.go  # ws server
      - client.go  # ws client connection
      - router.go  # watermill -> ws broadcast
      
  glazed/cmd/devctl/tui:
    files:
      - model.go  # tea.Model with pubsub field
      - subscriptions.go  # topic subscription setup
      - commands.go  # publish command helpers

# Message flow examples

flows:
  user_presses_l_on_backend:
    - tui: KeyPress("l") on selected service "backend"
    - tui: m.PublishCommand("cmd.logs.follow", &LogsFollowCmd{service_name: "backend"})
    - watermill: cmd.logs.follow -> supervisor_router
    - supervisor: start log stream goroutine
    - supervisor: publish ServiceLogLine on service.logs.line
    - watermill: service.logs.line -> tui_router
    - tui: HandleLogLine() -> tea.Cmd(NewLogLineMsg{})
    - tui: Update(NewLogLineMsg) -> append to viewport
    
  startup_pipeline_execution:
    - cli: devctl up
    - cli: m.PublishCommand("cmd.pipeline.run", &PipelineRunCmd{})
    - watermill: cmd.pipeline.run -> pipeline_router
    - pipeline: execute phases
    - pipeline: publish PipelinePhaseStarted on pipeline.phase.started
    - watermill: pipeline.phase.started -> tui_router
    - tui: HandlePhaseStarted() -> tea.Cmd(PhaseStartedMsg{})
    - tui: Update(PhaseStartedMsg) -> update startup view
    - pipeline: publish PipelineBuildStepProgress on pipeline.build_step.progress
    - watermill: pipeline.build_step.progress -> tui_router
    - tui: HandleBuildStepProgress() -> tea.Cmd(BuildStepProgressMsg{})
    - tui: Update(BuildStepProgressMsg) -> update progress bar
    
  service_status_polling:
    - supervisor: periodic goroutine (2s interval)
    - supervisor: read process stats
    - supervisor: publish ServiceProcessStats on service.process.stats
    - watermill: service.process.stats -> tui_router + ws_router
    - tui: HandleProcessStats() -> tea.Cmd(ProcessStatsUpdateMsg{})
    - tui: Update(ProcessStatsUpdateMsg) -> update dashboard service table
    - ws: broadcast to all connected clients

# Correlation tracking

correlation:
  pattern: "Request/response correlation via correlation_id"
  example:
    - tui: publish cmd.service.start with correlation_id="req-123"
    - supervisor: handle start, publish service.status.changed with correlation_id="req-123"
    - tui: match response via correlation_id, update UI state

# Graceful shutdown

shutdown:
  sequence:
    - tui: catch SIGINT/SIGTERM
    - tui: publish cmd.pipeline.cancel
    - tui: publish cmd.service.stop (all services)
    - tui: close watermill router
    - tui: tea.Quit()
```
