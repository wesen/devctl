package models

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-go-golems/devctl/pkg/tui"
	"github.com/go-go-golems/devctl/pkg/tui/styles"
	"github.com/go-go-golems/devctl/pkg/tui/widgets"
)

type EventLogModel struct {
	max     int
	entries []tui.EventLogEntry

	width  int
	height int

	searching bool
	search    textinput.Model
	filter    string

	serviceFilters map[string]bool
	serviceOrder   []string

	levelMenu    bool
	levelFilters map[tui.LogLevel]bool

	vp viewport.Model
}

func NewEventLogModel() EventLogModel {
	search := textinput.New()
	search.Placeholder = "filter…"
	search.Prompt = "/ "
	search.CharLimit = 200

	m := EventLogModel{
		max:            200,
		entries:        nil,
		search:         search,
		serviceFilters: map[string]bool{},
		serviceOrder:   nil,
		levelFilters: map[tui.LogLevel]bool{
			tui.LogLevelDebug: true,
			tui.LogLevelInfo:  true,
			tui.LogLevelWarn:  true,
			tui.LogLevelError: true,
		},
	}
	m.vp = viewport.New(0, 0)
	return m
}

func (m EventLogModel) WithSize(width, height int) EventLogModel {
	m.width, m.height = width, height
	m = m.resizeViewport()
	return m
}

func (m EventLogModel) Update(msg tea.Msg) (EventLogModel, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := v.Width, v.Height
		if w <= 0 {
			w = 80
		}
		if h <= 0 {
			h = 24
		}
		m.width, m.height = w, h
		m = m.resizeViewport()
		return m, nil
	case tea.KeyMsg:
		if m.levelMenu {
			switch v.String() {
			case "esc", "enter", "l":
				m.levelMenu = false
				return m, nil
			case "a":
				m = m.setAllLevelFilters(true)
				m = m.refreshViewportContent(false)
				return m, nil
			case "n":
				m = m.setAllLevelFilters(false)
				m = m.refreshViewportContent(false)
				return m, nil
			case "d":
				m = m.toggleLevelFilter(tui.LogLevelDebug)
				return m, nil
			case "i":
				m = m.toggleLevelFilter(tui.LogLevelInfo)
				return m, nil
			case "w":
				m = m.toggleLevelFilter(tui.LogLevelWarn)
				return m, nil
			case "e":
				m = m.toggleLevelFilter(tui.LogLevelError)
				return m, nil
			default:
				return m, nil
			}
		}

		if m.searching {
			switch v.String() {
			case "esc":
				m.searching = false
				m.search.Blur()
				return m, nil
			case "enter":
				m.filter = strings.TrimSpace(m.search.Value())
				m.searching = false
				m.search.Blur()
				m = m.refreshViewportContent(true)
				return m, nil
			}

			var cmd tea.Cmd
			m.search, cmd = m.search.Update(v)
			return m, cmd
		}

		switch v.String() {
		case "/":
			m.searching = true
			m.search.SetValue(m.filter)
			m.search.CursorEnd()
			m.search.Focus()
			return m, nil
		case "ctrl+l":
			m.filter = ""
			m.search.SetValue("")
			m = m.refreshViewportContent(true)
			return m, nil
		case "c":
			m.entries = nil
			m = m.refreshViewportContent(true)
			return m, nil
		case "l":
			m.levelMenu = true
			return m, nil
		case " ":
			m = m.toggleServiceByName("system")
			return m, nil
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(v.String()[0] - '1')
			m = m.toggleServiceFilter(idx)
			return m, nil
		}

		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(v)
		return m, cmd
	}
	return m, nil
}

func (m EventLogModel) Append(e tui.EventLogEntry) EventLogModel {
	e = normalizeEventLogEntry(e)
	m.entries = append(m.entries, e)
	if m.max > 0 && len(m.entries) > m.max {
		m.entries = append([]tui.EventLogEntry{}, m.entries[len(m.entries)-m.max:]...)
	}
	m = m.ensureServiceKnown(e.Source)
	m = m.refreshViewportContent(true)
	return m
}

func (m EventLogModel) View() string {
	theme := styles.DefaultTheme()

	var sections []string

	// Header with filter info
	titleRight := "[1-9] toggle  [space] system  [l] levels  [/] text filter  [c] clear  [↑/↓] scroll"
	if m.filter != "" {
		titleRight = fmt.Sprintf("filter=%q  %s", m.filter, titleRight)
	}

	// Search input if active
	if m.searching {
		sections = append(sections, m.search.View())
	}

	filterBar := theme.TitleMuted.Render(m.renderServiceFilterBar())
	levelBar := theme.TitleMuted.Render(m.renderLevelFilterBar())

	// Events viewport
	if len(m.entries) == 0 {
		content := lipgloss.JoinVertical(lipgloss.Left,
			filterBar,
			levelBar,
			theme.TitleMuted.Render("(no events yet)"),
		)
		emptyBox := widgets.NewBox(fmt.Sprintf("Events (%d)", len(m.entries))).
			WithTitleRight(titleRight).
			WithContent(content).
			WithSize(m.width, m.boxHeight())
		sections = append(sections, emptyBox.Render())
	} else {
		content := lipgloss.JoinVertical(lipgloss.Left, filterBar, levelBar, m.vp.View())
		eventsBox := widgets.NewBox(fmt.Sprintf("Events (%d)", len(m.entries))).
			WithTitleRight(titleRight).
			WithContent(content).
			WithSize(m.width, m.boxHeight())
		sections = append(sections, eventsBox.Render())
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m EventLogModel) resizeViewport() EventLogModel {
	usableHeight := m.height - m.boxChromeHeight()
	if m.searching {
		usableHeight--
	}
	if usableHeight < 3 {
		usableHeight = 3
	}
	m.vp.Width = maxInt(0, m.width)
	m.vp.Height = usableHeight
	m.vp.HighPerformanceRendering = false
	m = m.refreshViewportContent(false)
	return m
}

func (m EventLogModel) refreshViewportContent(gotoBottom bool) EventLogModel {
	theme := styles.DefaultTheme()

	if len(m.entries) == 0 {
		m.vp.SetContent("")
		return m
	}

	for _, entry := range m.entries {
		entry = normalizeEventLogEntry(entry)
		m = m.ensureServiceKnown(entry.Source)
	}

	lines := make([]string, 0, len(m.entries))
	for _, e := range m.entries {
		e = normalizeEventLogEntry(e)
		if m.filter != "" && !strings.Contains(e.Text, m.filter) {
			continue
		}

		if enabled, ok := m.serviceFilters[e.Source]; ok && !enabled {
			continue
		}
		if enabled, ok := m.levelFilters[e.Level]; ok && !enabled {
			continue
		}

		ts := e.At
		if ts.IsZero() {
			ts = time.Now()
		}

		icon := styles.LogLevelIcon(string(e.Level))
		style := theme.TitleMuted
		switch e.Level {
		case tui.LogLevelError:
			style = theme.StatusDead
		case tui.LogLevelWarn:
			style = lipgloss.NewStyle().Foreground(theme.Warning)
		case tui.LogLevelDebug:
			style = theme.TitleMuted
		case tui.LogLevelInfo:
			style = theme.TitleMuted
		}

		line := lipgloss.JoinHorizontal(lipgloss.Center,
			style.Render(icon),
			" ",
			theme.TitleMuted.Render(ts.Format("15:04:05")),
			" ",
			theme.TitleMuted.Render(fmt.Sprintf("[%s]", e.Source)),
			"  ",
			style.Render(e.Text),
		)
		lines = append(lines, line)
	}
	m.vp.SetContent(strings.Join(lines, "\n") + "\n")
	if gotoBottom {
		m.vp.GotoBottom()
	}
	return m
}

func (m EventLogModel) boxChromeHeight() int {
	// 2 borders + 1 title line + 2 fixed filter lines.
	return 5
}

func (m EventLogModel) boxHeight() int {
	return m.vp.Height + m.boxChromeHeight()
}

func (m EventLogModel) ensureServiceKnown(source string) EventLogModel {
	source = strings.TrimSpace(source)
	if source == "" {
		source = "system"
	}

	if m.serviceFilters == nil {
		m.serviceFilters = map[string]bool{}
	}
	if _, ok := m.serviceFilters[source]; ok {
		return m
	}

	m.serviceFilters[source] = true
	m.serviceOrder = append(m.serviceOrder, source)
	sort.Strings(m.serviceOrder)
	return m
}

func (m EventLogModel) toggleServiceFilter(idx int) EventLogModel {
	if idx < 0 || idx >= len(m.serviceOrder) {
		return m
	}
	return m.toggleServiceByName(m.serviceOrder[idx])
}

func (m EventLogModel) toggleServiceByName(name string) EventLogModel {
	name = strings.TrimSpace(name)
	if name == "" {
		return m
	}
	m = m.ensureServiceKnown(name)
	m.serviceFilters[name] = !m.serviceFilters[name]
	m = m.refreshViewportContent(false)
	return m
}

func (m EventLogModel) toggleLevelFilter(level tui.LogLevel) EventLogModel {
	if m.levelFilters == nil {
		m.levelFilters = map[tui.LogLevel]bool{}
	}
	if _, ok := m.levelFilters[level]; !ok {
		m.levelFilters[level] = true
	}
	m.levelFilters[level] = !m.levelFilters[level]
	m = m.refreshViewportContent(false)
	return m
}

func (m EventLogModel) setAllLevelFilters(enabled bool) EventLogModel {
	if m.levelFilters == nil {
		m.levelFilters = map[tui.LogLevel]bool{}
	}
	m.levelFilters[tui.LogLevelDebug] = enabled
	m.levelFilters[tui.LogLevelInfo] = enabled
	m.levelFilters[tui.LogLevelWarn] = enabled
	m.levelFilters[tui.LogLevelError] = enabled
	return m
}

func (m EventLogModel) renderServiceFilterBar() string {
	if len(m.serviceOrder) == 0 {
		return "Filters: (none)"
	}
	var parts []string
	parts = append(parts, "Filters:")
	for i, name := range m.serviceOrder {
		icon := "●"
		if enabled, ok := m.serviceFilters[name]; ok && !enabled {
			icon = "○"
		}
		label := fmt.Sprintf("%s %s", icon, name)
		if i < 9 {
			label = fmt.Sprintf("[%d]%s", i+1, label)
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "  ")
}

func (m EventLogModel) renderLevelFilterBar() string {
	levels := []tui.LogLevel{tui.LogLevelDebug, tui.LogLevelInfo, tui.LogLevelWarn, tui.LogLevelError}

	var parts []string
	parts = append(parts, "Levels:")
	for _, lvl := range levels {
		icon := "●"
		if enabled, ok := m.levelFilters[lvl]; ok && !enabled {
			icon = "○"
		}
		parts = append(parts, fmt.Sprintf("%s %s", icon, lvl))
	}
	if m.levelMenu {
		parts = append(parts, "[d/i/w/e] toggle", "[a] all", "[n] none", "[esc] close")
	} else {
		parts = append(parts, "[l] menu")
	}
	return strings.Join(parts, "  ")
}

func normalizeEventLogEntry(e tui.EventLogEntry) tui.EventLogEntry {
	if strings.TrimSpace(e.Source) == "" {
		e.Source = "system"
	}
	if e.Level == "" {
		e.Level = tui.LogLevelInfo
	}
	return e
}
