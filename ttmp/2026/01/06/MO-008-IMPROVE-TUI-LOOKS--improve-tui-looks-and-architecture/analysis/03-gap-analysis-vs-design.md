# Gap Analysis: MO-006 Design vs Current Implementation

Date: 2026-01-07  
Ticket: MO-008-IMPROVE-TUI-LOOKS  
Reference: `MO-006-DEVCTL-TUI/.../01-devctl-tui-layout.md`

## Summary

Comparing the original ASCII mockups and architecture spec in MO-006 to the current TUI implementation reveals significant feature gaps. The current implementation provides basic functionality but lacks many of the advanced features and views outlined in the design.

---

## 1. Main Dashboard View

### Design (Section 1)
```
│ ● System Status: Running                                      Uptime: 2h 34m 12s │
│ Services (3)                                                    [l] logs [r] restart │
│ │ ✓ backend           Running    PID 12847   Healthy   CPU 12%   MEM 245MB   │ │
│ Recent Events (5)                                             [f] follow [c] clear │
│ │ 14:23:45  backend    ℹ  Request completed in 234ms                          │ │
│ Plugins (3 active)                                                                │
│  • moments-config   (priority: 10)   ops: 2   commands: 0                        │
```

### Current Implementation
- ✅ System status with uptime (in header)
- ✅ Services table with: Name, Status (Running/Dead), PID
- ❌ **Missing: Health column** (Healthy/Unhealthy)
- ❌ **Missing: CPU % column**
- ❌ **Missing: Memory MB column**
- ❌ **Missing: Recent Events preview box** (last 5 events)
- ❌ **Missing: Plugins summary section**
- ✅ Footer keybindings

### Priority: High
The dashboard should show at-a-glance health/resource info without navigating.

---

## 2. Service Detail View

### Design (Section 2)
```
│ Status: ✓ Running                                           Started: 2h 34m ago  │
│ Health: Healthy (http://localhost:8083/rpc/v1/health)      Last check: 2s ago    │
│ Process Info                                                                      │
│  PID:         12847                                                               │
│  Command:     go run ./cmd/moments-server serve                                   │
│  Working Dir: /home/user/moments/backend                                          │
│  CPU:         12.3%                                                               │
│  Memory:      245.8 MB                                                            │
│ Environment                                                                       │
│  PORT=8083  DB_URL=postgresql://localhost:5432/moments  ENV=development           │
│ Live Logs                                                    [↑/↓] scroll [f] find │
│ [r] restart [s] stop [k] kill [d] detach [ESC] back                              │
```

### Current Implementation
- ✅ Service name + status (Running/Dead)
- ✅ PID
- ❌ **Missing: Started/uptime for service**
- ❌ **Missing: Health check info** (URL, last check)
- ❌ **Missing: Command line** (how service was started)
- ❌ **Missing: Working directory**
- ❌ **Missing: CPU %**
- ❌ **Missing: Memory usage**
- ❌ **Missing: Environment variables section**
- ✅ Log path (stdout/stderr)
- ✅ Stream selector (stdout/stderr)
- ✅ Follow mode
- ✅ Log viewport with scrolling
- ✅ Filter search
- ❌ **Missing: [s] stop, [d] detach** keybindings

### Priority: High
Process info and health are critical for debugging.

---

## 3. Startup/Pipeline View

### Design (Section 3)
```
│ ⚙ System Status: Starting                                    Phase: Build [2/5]  │
│ Pipeline Progress                                                                 │
│ │ ✓ Config Mutation     3 plugins applied              0.2s                   │ │
│ │ ▶ Build               Running...                     5.3s                   │ │
│ Build Steps (2/3 complete)                                                        │
│ │ ✓ backend-deps        pnpm install                   2.1s                   │ │
│ │ ▶ backend-compile     go build ./cmd/moments-server  5.3s (running)         │ │
│ Live Output                                                                       │
│ │ [backend-compile] ██████████████████░░░░░░░░░░ 65%                          │ │
│ Applied Config Patches                                                            │
│  • services.backend.port → 8083                          (moments-config)        │
```

### Current Implementation
- ✅ Pipeline phases with icons (✓/▶/○/✗)
- ✅ Phase durations
- ✅ Build steps list with status
- ✅ Validation errors/warnings
- ❌ **Missing: Progress percentage** (visual bar)
- ❌ **Missing: Live output viewport** (real-time build output)
- ❌ **Missing: Config patches applied section**
- ❌ **Missing: Step command display** (what's running)
- ❌ **Missing: Overall phase progress indicator** ([2/5])

### Priority: Medium
Live output and progress bars would improve UX during builds.

---

## 4. Plugin List View

### Design (Section 4)
```
│ ┌─ moments-config ───────────────────────────────────────────────────────────┐  │
│ │ Status:   ✓ Active                                    Priority: 10         │  │
│ │ Path:     ./moments/plugins/devctl-config.sh                                │  │
│ │ Protocol: v1                                                                │  │
│ │ Capabilities:                                                               │  │
│ │   Ops:      config.mutate, validate.run                                     │  │
│ │   Streams:  (none)                                                          │  │
│ │   Commands: (none)                                                          │  │
│ │ Declares:                                                                   │  │
│ │   side_effects: none        idempotent: true                                │  │
│ └─────────────────────────────────────────────────────────────────────────────┘  │
│ [i] inspect [d] disable [e] enable [r] reload [ESC] back                         │
```

### Current Implementation
- ❌ **Plugin List View: NOT IMPLEMENTED**
- No ViewPlugin view type
- No plugin discovery/inspection UI
- No enable/disable/reload functionality

### Priority: Low
Plugins are mostly static config, not runtime-critical.

---

## 5. Error State / Validation View

### Design (Section 5)
```
│ Validation Errors (2)                                                             │
│ │ ✗ EPORT_IN_USE                                          (moments-config)    │ │
│ │   Port 8083 is already in use                                               │ │
│ │   Fix: Run 'devctl stop' or choose another port in config                   │ │
│ Validation Warnings (1)                                                           │
│ │ ⚠ WDEPRECATED                                           (moments-build)     │ │
│ │   Node.js version 16 is deprecated, upgrade to 18+                          │ │
│ Actions                                                                           │
│  [r] retry   [f] fix manually   [l] view logs   [q] quit                         │
```

### Current Implementation
- ✅ Validation errors/warnings displayed in Pipeline view
- ✅ Error codes and messages
- ❌ **Missing: Fix suggestions** (actionable hints)
- ❌ **Missing: Plugin attribution** (which plugin produced error)
- ❌ **Missing: [r] retry action**
- ❌ **Missing: Expandable error details**

### Priority: Medium
Fix suggestions and retry would reduce friction.

---

## 6. Multi-Service Log Stream View (Events)

### Design (Section 6)
```
│ Following: All Services                            [f] filter [1-9] select service │
│ │ 14:34:12.234  [backend]    INFO  POST /api/moments                          │ │
│ │ 14:34:12.156  [postgres]   INFO  Query: SELECT * FROM moments LIMIT 20      │ │
│ Filters: ● backend  ● frontend  ● postgres  ● system      [space] toggle         │
│ Levels:  ● DEBUG   ● INFO   ● WARN   ● ERROR              [l] level menu         │
│ Stats: 1,247 events (18/sec)   Buffer: 500 lines   Dropped: 0                    │
│ [p] pause [c] clear [s] save [/] search [ESC] back                               │
```

### Current Implementation (EventLogModel)
- ✅ Event timeline with timestamps
- ✅ Icons based on content (error/success/warning)
- ✅ Filter search (/)
- ✅ Clear events (c)
- ❌ **Missing: Service source column** ([backend], [postgres])
- ❌ **Missing: Log level column** (DEBUG/INFO/WARN/ERROR)
- ❌ **Missing: Service filter toggles** (● backend ● frontend)
- ❌ **Missing: Level filter toggles** (● DEBUG ● INFO)
- ❌ **Missing: Stats line** (events/sec, buffer size)
- ❌ **Missing: Pause toggle** ([p])
- ❌ **Missing: Save to file** ([s])

### Priority: Medium
Service/level filtering would help with noisy logs.

---

## 7. Architecture Features (from Watermill/Protobuf section)

### Design Spec
- Watermill pub/sub for event-driven architecture
- Protobuf messages for typed events
- Command topics (cmd.service.start, cmd.pipeline.run, etc.)
- Event topics (service.status.changed, pipeline.phase.started, etc.)
- Health check polling (every 5s)
- Process stats polling (every 2s)
- WebSocket bridge for web UI

### Current Implementation
- ✅ Basic state polling (refresh interval)
- ✅ Action requests (up, down, restart, kill)
- ❌ **Missing: Watermill integration**
- ❌ **Missing: Protobuf messages**
- ❌ **Missing: Health check polling**
- ❌ **Missing: Process stats polling (CPU/MEM)**
- ❌ **Missing: WebSocket bridge**
- ❌ **Missing: Correlation ID tracking**

### Priority: Low (architectural)
Current polling approach works; event-driven would be more scalable.

---

## 8. Keybindings Gap

### Design vs Current

| Keybinding | Design | Current |
|------------|--------|---------|
| [s] services | ✓ | ❌ (uses tab) |
| [p] plugins | ✓ | ❌ (not implemented) |
| [e] events | ✓ | ❌ (uses tab) |
| [h] help | ✓ | ✅ (uses ?) |
| [r] restart | ✓ | ✅ |
| [s] stop | ✓ | ❌ |
| [k] kill | ✓ | ✅ (uses x) |
| [d] detach | ✓ | ❌ |
| [f] find/filter | ✓ | ✅ (uses /) |
| [f] follow | ✓ | ✅ |
| [c] clear | ✓ | ✅ |
| [p] pause | ✓ | ❌ |
| [/] search | ✓ | ✅ |

---

## Summary: Feature Completeness

| Category | Complete | Partial | Missing |
|----------|----------|---------|---------|
| Dashboard - Services | | ✅ | Health/CPU/MEM |
| Dashboard - Events Preview | | | ❌ |
| Dashboard - Plugins | | | ❌ |
| Service Detail - Process Info | | ✅ | CPU/MEM/Cmd/Env |
| Service Detail - Health | | | ❌ |
| Service Detail - Logs | ✅ | | |
| Pipeline - Phases | ✅ | | |
| Pipeline - Steps | ✅ | | |
| Pipeline - Live Output | | | ❌ |
| Pipeline - Progress Bars | | | ❌ |
| Plugin List View | | | ❌ |
| Events - Timeline | ✅ | | |
| Events - Service Filter | | | ❌ |
| Events - Level Filter | | | ❌ |
| Watermill Integration | | | ❌ |

---

## Recommended Implementation Order

### Phase A: Dashboard Enhancements (High Value, Medium Effort)
1. Add Health column to services table (requires health check data)
2. Add CPU/MEM columns (requires process stats polling)
3. Add Recent Events preview box (last 5)
4. Add Plugins summary (optional)

### Phase B: Service Detail Enhancements (High Value, Medium Effort)
1. Add process info: command, working dir
2. Add CPU/MEM display
3. Add health check info (if available)
4. Add environment variables section
5. Add stop/detach keybindings

### Phase C: Events View Improvements (Medium Value, Low Effort)
1. Add service source column
2. Add log level column with icons
3. Add service filter toggles
4. Add level filter toggles
5. Add stats line

### Phase D: Pipeline Live Output (Medium Value, High Effort)
1. Add live output viewport
2. Add progress bars
3. Add config patches display

### Phase E: Plugin View (Low Value, Medium Effort)
1. Create PluginModel
2. Add plugin list view
3. Add inspect/enable/disable actions

### Phase F: Event-Driven Architecture (Low Value, High Effort)
1. Integrate Watermill
2. Define protobuf messages
3. Replace polling with subscriptions

---

## Data Requirements

To implement missing features, the TUI needs additional data from the supervisor/backend:

| Feature | Required Data | Source |
|---------|--------------|--------|
| Health column | health check results | supervisor polling |
| CPU/MEM columns | process stats | /proc/[pid]/stat or ps |
| Service command | exec command args | state.json enhancement |
| Working dir | service cwd | state.json enhancement |
| Environment vars | env map | state.json enhancement |
| Live build output | stdout/stderr stream | build step execution |
| Config patches | patch list | config mutation phase |

---

## References

- Design: `MO-006-DEVCTL-TUI/.../01-devctl-tui-layout.md`
- Current models: `pkg/tui/models/*.go`
- State snapshot: `pkg/tui/domain.go`

