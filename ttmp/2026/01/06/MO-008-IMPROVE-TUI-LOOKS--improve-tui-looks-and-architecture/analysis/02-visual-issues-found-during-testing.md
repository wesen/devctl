# Visual Issues Found During TUI Testing

Date: 2026-01-07  
Ticket: MO-008-IMPROVE-TUI-LOOKS  
Testing Method: tmux capture-pane with 80x24, 100x30, and 120x40 terminal sizes

## Summary

After implementing the lipgloss styling in Phases 1-6, visual testing revealed several layout and rendering issues. The most critical is the ServiceModel overflowing its allocated height when displaying dead services.

## Issues Found

### Issue 1: ServiceModel Content Overflow (Critical)

**Symptom**: When viewing a dead service, the header bar ("DevCtl — service") is pushed off the top of the screen.

**Root Cause**: The `ServiceModel.View()` method creates dynamic-height sections that can exceed the allocated `m.height`:
- Process info box: `len(infoLines) + 3` ≈ 10 lines
- Exit info box (for dead services): variable height (8-15 lines depending on stderr tail)
- Log viewport box: `m.vp.Height + 3`

The `reservedViewportLines()` calculation is based on the OLD plain-text layout and doesn't account for the new box borders (+3 lines per box).

**Screenshots**:

```
Dead service at 80x24 - header is completely cut off:
╭────────────────────────────────────────────────────────────────────────────╮
│ ✗ Exit: exit_code=2  PID 3734164                                           │
│ Exited: 2026-01-06 20:48:13                                                │
... (no header visible above)
```

**Fix Required**: 
1. Update `reservedViewportLines()` to account for box borders
2. Consider fixed-height sections with scrolling within them
3. Or calculate total available height and dynamically size the viewport

**Files to Modify**:
- `pkg/tui/models/service_model.go` - `reservedViewportLines()` and `View()`

---

### Issue 2: Stray Border Characters After Header Separator

**Symptom**: A stray `╯` character appears after the header separator line on some views.

**Example**:
```
 DevCtl — service   ✓ Running     Uptime: 11m 6s  [tab] switch [?] help [q] quit
━━━━━━━━━━━━━━━━━━━━━━━━━━                                                     ╯
                                                                               ^
                                                                               stray char
```

**Root Cause**: When the content from a previous render is longer than the current render, old characters remain on screen. This is a classic terminal rendering issue where the screen isn't being fully cleared.

**Fix Required**:
1. Ensure full-width rendering of each line
2. Or use lipgloss width constraints to pad lines to terminal width
3. Investigate if bubbletea's Clear() should be called on view changes

**Files to Modify**:
- `pkg/tui/models/root_model.go` - View composition

---

### Issue 3: Line Truncation Without Ellipsis

**Symptom**: Long file paths and stderr lines are hard-wrapped in the middle of text without proper ellipsis or handling.

**Example**:
```
│Path: /tmp/devctl-tui-fixture-GkbBIS/.devctl/logs/http-20260106-              │
│204811.stdout.log                                                             │
```

**Root Cause**: The box content exceeds the box width and wraps.

**Fix Required**:
1. Truncate long strings with ellipsis before rendering
2. Use lipgloss `MaxWidth` or `Truncate` for constrained content

**Files to Modify**:
- `pkg/tui/models/service_model.go` - path and stderr rendering
- `pkg/tui/widgets/box.go` - consider adding content truncation

---

### Issue 4: PID Truncation in Dashboard Table

**Symptom**: PID values are truncated with `…` in the dashboard table.

**Example**:
```
│> ✓ http              Running         PID 37340…                              │
```

**Root Cause**: The table column widths are not optimized for the content.

**Fix Required**:
1. Adjust column widths in `DashboardModel.View()` 
2. Or make PID column wider to fit 7-digit PIDs

**Files to Modify**:
- `pkg/tui/models/dashboard_model.go` - table column definitions
- `pkg/tui/widgets/table.go` - column width calculation

---

### Issue 5: Large Empty Space in Dashboard View

**Symptom**: The dashboard view has a lot of empty space below the services box at larger terminal sizes.

**Root Cause**: The services box uses a fixed height based on number of services, leaving the rest of the screen empty.

**Suggested Enhancement**:
1. Add a "Recent Events" preview box below services
2. Or expand services box to fill available height

---

### Issue 6: Resize Handling for ServiceModel

**Symptom**: After resizing the terminal while in ServiceModel, the header remains cut off.

**Root Cause**: When window resize happens:
1. RootModel receives `tea.WindowSizeMsg`
2. RootModel calls `applyChildSizes()` which calls `m.service.WithSize()`
3. ServiceModel.WithSize() calls `resizeViewport()` which updates `m.vp.Height`
4. BUT the View() method still calculates box heights that don't fit

The resize IS being propagated, but the View() method's height calculation is wrong.

**Fix Required**: Same as Issue 1 - fix the height calculations in ServiceModel.View()

---

## Testing Matrix Results

| View | 80x24 | 100x30 | 120x40 |
|------|-------|--------|--------|
| Dashboard | ✓ OK | ✓ OK | ✓ OK |
| Service (alive) | ⚠ stray char | ⚠ stray char | ⚠ stray char |
| Service (dead) | ✗ header cut off | ✗ header cut off | ✗ header cut off |
| Events | ✓ OK | ✓ OK | ✓ OK |
| Pipeline (empty) | ⚠ stray char | ⚠ stray char | ⚠ stray char |
| Help overlay | ✓ OK | ✓ OK | ✓ OK |

## Priority Ranking

1. **Critical**: Issue 1 (ServiceModel overflow) - Makes dead service view unusable
2. **High**: Issue 2 (Stray characters) - Visual artifact on multiple views
3. **Medium**: Issue 3 (Line truncation) - Readability issue
4. **Low**: Issue 4 (PID truncation) - Minor visual issue
5. **Enhancement**: Issue 5 (Empty space) - Not a bug

## Recommended Fix Order

1. Fix `reservedViewportLines()` in ServiceModel to use accurate line counts
2. Update ServiceModel.View() to constrain total height
3. Add full-width line rendering to prevent stray characters
4. Add string truncation helpers for long paths

## Code References

### ServiceModel.reservedViewportLines() (current)
```go:pkg/tui/models/service_model.go
func (m ServiceModel) reservedViewportLines() int {
    lines := 0
    lines += 2  // Header + key help (OLD: was plain text)
    lines += 3  // Blank + path line + blank
    // ... doesn't account for box borders (+3 per box)
}
```

### ServiceModel.View() height issues
```go:pkg/tui/models/service_model.go
// Info box height
infoBox := widgets.NewBox("Service: "+m.name).
    WithSize(m.width, len(infoLines)+3)  // +3 for borders

// Log box height
logBox := widgets.NewBox(logTitle).
    WithSize(m.width, m.vp.Height+3)  // +3 for borders

// Exit info box has variable height too
```

### RootModel.applyChildSizes()
```go:pkg/tui/models/root_model.go
func (m RootModel) applyChildSizes() RootModel {
    childHeight := m.height - m.headerLines()  // Correct
    m.service = m.service.WithSize(m.width, childHeight)  // Passes correct height
    // But ServiceModel.View() doesn't respect this height constraint
}
```

## Window Resize Event Bubbling

**Confirmed Working**: The resize event bubbling is actually working correctly:
1. `tea.WindowSizeMsg` is received by RootModel
2. RootModel calls `applyChildSizes()` 
3. Child models receive new dimensions via `WithSize()`
4. Child viewports are resized via `resizeViewport()`

The issue is NOT with event bubbling - it's with how ServiceModel.View() calculates its content height.

