# MO-009: TUI Complete Features - Development Diary

## Overview

This diary documents the planning and implementation of all missing TUI features identified in the MO-008 gap analysis to bring the `devctl tui` to full design parity with MO-006.

---

## 2026-01-07

### Step 1: Ticket Creation (21:23)

Created this ticket to track the comprehensive implementation of all missing TUI features.

#### Background
- MO-006 defined the original TUI design with ASCII mockups
- MO-008 implemented basic styling with lipgloss and fixed visual issues
- Gap analysis (`MO-008/.../03-gap-analysis-vs-design.md`) identified 50+ missing features

#### What I did
- Created ticket MO-009-TUI-COMPLETE-FEATURES using docmgr
- Reviewed the original design document thoroughly
- Created comprehensive implementation plan with 8 phases, 75+ tasks

#### Key Documents Created
- `design/01-implementation-plan.md` - Full implementation plan with:
  - Data layer requirements (process stats, health checks, env vars)
  - UI changes for each view (dashboard, service, events, pipeline, plugins)
  - ASCII mockups showing target state
  - Code snippets and type definitions
  - Task dependencies graph
  - Recommended implementation order

- `tasks.md` - Checklist of all 75 tasks organized by phase

#### Design Decisions

1. **Data-First Approach**: Phase 1 focuses on data layer (process stats, health, env vars) before UI, because many features depend on this data being available.

2. **Events View First**: Recommended starting with Phase 4 (Events enhancements) because it has low dependencies and high visibility - quick wins.

3. **Plugin View Last**: Plugin list is lowest priority since plugins are mostly static config.

4. **New Widget for Progress Bars**: Phase 5 requires a new progress bar widget for build steps.

5. **Process Stats via /proc**: On Linux, read `/proc/[pid]/stat` for CPU/MEM. Fallback to `ps` on macOS.

---

### Step 2: Gap Analysis Summary (from MO-008)

#### Dashboard Missing Features
| Feature | Status | Phase |
|---------|--------|-------|
| Health column | ❌ | 2.1 |
| CPU % column | ❌ | 2.1 |
| Memory MB column | ❌ | 2.1 |
| Recent Events preview | ❌ | 2.2 |
| Plugins summary | ❌ | 2.3 |

#### Service Detail Missing Features
| Feature | Status | Phase |
|---------|--------|-------|
| Command line | ❌ | 3.1 |
| Working directory | ❌ | 3.1 |
| CPU/Memory stats | ❌ | 3.1 |
| Health check info | ❌ | 3.2 |
| Environment vars | ❌ | 3.3 |
| Stop keybinding | ❌ | 3.4 |

#### Events View Missing Features
| Feature | Status | Phase |
|---------|--------|-------|
| Service source column | ❌ | 4.1 |
| Log level column | ❌ | 4.2 |
| Service filter toggles | ❌ | 4.3 |
| Level filter toggles | ❌ | 4.4 |
| Stats line | ❌ | 4.5 |
| Pause toggle | ❌ | 4.6 |

#### Pipeline View Missing Features
| Feature | Status | Phase |
|---------|--------|-------|
| Progress bars | ❌ | 5.1 |
| Live output viewport | ❌ | 5.2 |
| Config patches display | ❌ | 5.3 |

#### Not Implemented
| Feature | Status | Phase |
|---------|--------|-------|
| Plugin List View | ❌ | 6 |

---

### Architecture Notes

#### Data Flow for Process Stats
```
/proc/[pid]/stat → ReadProcessStats() → StateSnapshot.ProcessStats
                                              ↓
                                     DashboardModel.View()
                                     ServiceModel.View()
```

#### Data Flow for Health Checks
```
Service Config (health endpoint) → Supervisor Health Poller → StateSnapshot.Health
                                                                     ↓
                                                          DashboardModel.View()
                                                          ServiceModel.View()
```

#### Events Enhancement Architecture
```
EventLogEntry {
    At:     time.Time
    Text:   string
    Source: string    // NEW: "backend", "system"
    Level:  LogLevel  // NEW: DEBUG/INFO/WARN/ERROR
}

EventLogModel {
    entries:        []EventLogEntry
    serviceFilters: map[string]bool  // NEW
    levelFilters:   map[LogLevel]bool // NEW
    paused:         bool              // NEW
    eventsPerSec:   float64           // NEW
}
```

---

### What Warrants Discussion

1. **Health Check Implementation**: Where should health check logic live? Options:
   - In supervisor as a goroutine
   - In TUI as polling
   - Separate health daemon

2. **Environment Sanitization**: Need clear rules for what to redact:
   - Keys containing: PASSWORD, SECRET, TOKEN, KEY, CREDENTIAL, API_KEY
   - Should we allow override via config?

3. **Live Build Output**: Currently build output isn't streamed to TUI. Need to:
   - Add output channel to build executor
   - Create PipelineLiveOutputMsg
   - Wire through Watermill or direct channel

4. **Plugin Enable/Disable**: The design shows enable/disable actions but current implementation may not support runtime plugin state changes.

---

### Next Steps

Recommended order for implementation:

1. **Quick Win**: Phase 4.1-4.2 (Events source/level columns)
   - Low effort, immediate improvement
   - No backend changes needed

2. **Foundation**: Phase 1.1 (Process stats)
   - Enables dashboard and service CPU/MEM
   - Requires new pkg/proc/stats.go

3. **Dashboard Polish**: Phase 2.1-2.2
   - High visibility improvement
   - Depends on Phase 1.1

4. **Continue from there based on capacity**

---

### Technical References

- Original Design: `MO-006-DEVCTL-TUI/.../01-devctl-tui-layout.md`
- Gap Analysis: `MO-008-IMPROVE-TUI-LOOKS/.../03-gap-analysis-vs-design.md`
- Current TUI Code: `pkg/tui/models/*.go`
- Current Widgets: `pkg/tui/widgets/*.go`
- Current Styles: `pkg/tui/styles/*.go`
- State Types: `pkg/state/types.go`
- TUI Domain: `pkg/tui/domain.go`

