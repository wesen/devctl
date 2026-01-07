package models

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-go-golems/devctl/pkg/state"
	"github.com/go-go-golems/devctl/pkg/tui"
	"github.com/go-go-golems/devctl/pkg/tui/styles"
	"github.com/go-go-golems/devctl/pkg/tui/widgets"
)

type LogStream string

const (
	LogStdout LogStream = "stdout"
	LogStderr LogStream = "stderr"
)

type logTickMsg struct{}

type logStreamState struct {
	path      string
	offset    int64
	carry     string
	lines     []string
	lastErr   string
	seenFirst bool
}

type ServiceModel struct {
	width  int
	height int

	last *tui.StateSnapshot

	name string

	active LogStream
	follow bool

	searching bool
	search    textinput.Model
	filter    string

	exitInfo    *state.ExitInfo
	exitInfoErr string

	tailLines int
	maxLines  int
	tickEvery time.Duration

	stdout logStreamState
	stderr logStreamState

	vp viewport.Model
}

func NewServiceModel() ServiceModel {
	search := textinput.New()
	search.Placeholder = "filter…"
	search.Prompt = "/ "
	search.CharLimit = 200

	m := ServiceModel{
		active:    LogStdout,
		follow:    true,
		search:    search,
		tailLines: 200,
		maxLines:  2000,
		tickEvery: 250 * time.Millisecond,
	}
	m.vp = viewport.New(0, 0)
	return m
}

func (m ServiceModel) WithSize(width, height int) ServiceModel {
	m.width, m.height = width, height
	m = m.resizeViewport()
	return m
}

func (m ServiceModel) WithSnapshot(s tui.StateSnapshot) ServiceModel {
	m.last = &s
	m = m.syncPathsFromSnapshot()
	m = m.syncExitInfoFromSnapshot()
	return m
}

func (m ServiceModel) WithService(name string) ServiceModel {
	m.name = name
	m.active = LogStdout
	m.follow = true
	m.searching = false
	m.filter = ""
	m.search.SetValue("")
	m.search.Blur()
	m.stdout = logStreamState{}
	m.stderr = logStreamState{}
	m.exitInfo = nil
	m.exitInfoErr = ""
	m = m.syncPathsFromSnapshot()
	m = m.syncExitInfoFromSnapshot()
	m = m.loadInitialTail()
	return m
}

func (m ServiceModel) Update(msg tea.Msg) (ServiceModel, tea.Cmd) {
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
		case "tab":
			if m.active == LogStdout {
				m.active = LogStderr
			} else {
				m.active = LogStdout
			}
			m = m.refreshViewportContent(true)
			return m, nil
		case "f":
			m.follow = !m.follow
			if m.follow {
				return m, m.tickCmd()
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(v)
		if cmd != nil {
			return m, cmd
		}
		return m, nil
	case logTickMsg:
		if m.name == "" || !m.follow {
			return m, nil
		}
		m = m.tickReadAll()
		m = m.refreshViewportContent(true)
		return m, m.tickCmd()
	}
	return m, nil
}

func (m ServiceModel) View() string {
	theme := styles.DefaultTheme()

	if m.name == "" {
		return theme.TitleMuted.Render("No service selected.")
	}

	rec, alive, found := m.lookupService()
	if !found {
		box := widgets.NewBox("Service: " + m.name).
			WithContent(theme.TitleMuted.Render("No record for this service in the current state snapshot.")).
			WithSize(m.width, 5)
		return box.Render()
	}

	// Calculate fixed section heights first
	const infoBoxHeight = 6  // Compact info: status + stream + follow + border
	const logBoxBorder = 3   // Top border + title + bottom border
	exitBoxHeight := 0
	if !alive {
		exitBoxHeight = m.exitInfoHeight()
	}
	errBoxHeight := 0
	if errText := m.activeState().lastErr; errText != "" {
		errBoxHeight = 3 // border + content + border
	}
	searchHeight := 0
	if m.searching {
		searchHeight = 1
	}

	// Log viewport gets remaining space
	usedHeight := infoBoxHeight + exitBoxHeight + errBoxHeight + searchHeight + logBoxBorder
	logViewportHeight := m.height - usedHeight
	if logViewportHeight < 3 {
		logViewportHeight = 3
	}

	var sections []string

	// Compact process info box
	statusIcon := styles.StatusIcon(alive)
	statusText := "Running"
	statusStyle := theme.StatusRunning
	if !alive {
		statusText = "Dead"
		statusStyle = theme.StatusDead
	}

	// Build compact info line
	stdoutTab := "stdout"
	stderrTab := "stderr"
	if m.active == LogStdout {
		stdoutTab = "[stdout]"
	} else {
		stderrTab = "[stderr]"
	}

	followText := "off"
	if m.follow {
		followText = "on"
	}

	// Truncate path if too long
	path := m.activeState().path
	if path == "" {
		path = "(unknown)"
	}
	maxPathLen := m.width - 10
	if maxPathLen > 0 && len(path) > maxPathLen {
		path = "..." + path[len(path)-maxPathLen+3:]
	}

	infoContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center,
			statusStyle.Render(statusIcon),
			" ",
			theme.Title.Render(statusText),
			"  ",
			theme.TitleMuted.Render(fmt.Sprintf("PID %d", rec.PID)),
			"    ",
			theme.TitleMuted.Render(stdoutTab),
			"/",
			theme.TitleMuted.Render(stderrTab),
			"  ",
			theme.TitleMuted.Render("follow:"+followText),
		),
		theme.TitleMuted.Render(path),
	)

	if m.filter != "" {
		infoContent = lipgloss.JoinVertical(lipgloss.Left,
			infoContent,
			theme.TitleMuted.Render(fmt.Sprintf("filter: %q", m.filter)),
		)
	}

	infoBox := widgets.NewBox("Service: "+m.name).
		WithTitleRight("[esc] back").
		WithContent(infoContent).
		WithSize(m.width, infoBoxHeight)

	sections = append(sections, infoBox.Render())

	// Exit info for dead services (compact)
	if !alive && exitBoxHeight > 0 {
		exitContent := m.renderCompactExitInfo(theme, exitBoxHeight-2) // -2 for border
		sections = append(sections, exitContent)
	}

	// Log error if present
	if errText := m.activeState().lastErr; errText != "" {
		// Truncate error text
		if len(errText) > m.width-10 {
			errText = errText[:m.width-13] + "..."
		}
		errBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Error).
			Padding(0, 1).
			Width(m.width - 4).
			Render(lipgloss.JoinHorizontal(lipgloss.Center,
				theme.StatusDead.Render(styles.IconError),
				" ",
				theme.StatusDead.Render(errText),
			))
		sections = append(sections, errBox)
	}

	// Search input if active
	if m.searching {
		sections = append(sections, m.search.View())
	}

	// Log viewport in a box - uses remaining height
	logTitle := fmt.Sprintf("Logs (%s)", m.active)
	logBox := widgets.NewBox(logTitle).
		WithTitleRight("[↑/↓] scroll  [f] follow  [/] filter").
		WithContent(m.vp.View()).
		WithSize(m.width, logViewportHeight)

	sections = append(sections, logBox.Render())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m ServiceModel) exitInfoHeight() int {
	// Fixed compact height to ensure header stays visible
	// Exit line + stderr header + 2 stderr lines = 4 content lines + 2 border = 6
	return 6
}

func (m ServiceModel) renderCompactExitInfo(theme styles.Theme, maxLines int) string {
	if m.exitInfo == nil {
		msg := "unknown"
		if m.exitInfoErr != "" {
			msg = m.exitInfoErr
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Warning).
			Padding(0, 1).
			Width(m.width - 4).
			Render(lipgloss.JoinHorizontal(lipgloss.Center,
				theme.StatusDead.Render(styles.IconError),
				" ",
				theme.Title.Render("Exit: "),
				theme.TitleMuted.Render(msg),
			))
	}

	ei := m.exitInfo
	var lines []string

	// Exit status line with more info condensed
	exitKind := "unknown"
	exitIcon := styles.IconError
	if ei.Signal != "" {
		exitKind = "signal " + ei.Signal
		exitIcon = styles.IconWarning
	} else if ei.ExitCode != nil {
		exitKind = fmt.Sprintf("code=%d", *ei.ExitCode)
		if *ei.ExitCode == 0 {
			exitIcon = styles.IconSuccess
		}
	}

	exitedAt := ""
	if !ei.ExitedAt.IsZero() {
		exitedAt = " @ " + ei.ExitedAt.Format("15:04:05")
	}

	errSuffix := ""
	if ei.Error != "" {
		errSuffix = "  err: " + ei.Error
		if len(errSuffix) > 30 {
			errSuffix = errSuffix[:27] + "..."
		}
	}

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center,
		theme.StatusDead.Render(exitIcon),
		" ",
		theme.Title.Render("Exit: "),
		theme.TitleMuted.Render(exitKind),
		theme.TitleMuted.Render(exitedAt),
		theme.StatusDead.Render(errSuffix),
	))

	// Stderr tail - show just 2 lines max, strictly truncated to prevent wrapping
	stderrLines := ei.StderrTail
	maxStderr := 2
	if len(stderrLines) > maxStderr {
		stderrLines = stderrLines[len(stderrLines)-maxStderr:]
	}
	if len(stderrLines) > 0 {
		lines = append(lines, theme.TitleMuted.Render("stderr:"))
		maxLineLen := m.width - 14 // Account for box border, padding, "! " prefix
		if maxLineLen < 20 {
			maxLineLen = 20
		}
		for _, line := range stderrLines {
			if len(line) > maxLineLen {
				line = line[:maxLineLen-3] + "..."
			}
			lines = append(lines, theme.StatusDead.Render("! "+line))
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Error).
		Padding(0, 1).
		Width(m.width - 4).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m ServiceModel) tickCmd() tea.Cmd {
	if m.tickEvery <= 0 {
		m.tickEvery = 250 * time.Millisecond
	}
	return tea.Tick(m.tickEvery, func(time.Time) tea.Msg { return logTickMsg{} })
}

func (m ServiceModel) resizeViewport() ServiceModel {
	// Calculate viewport height based on available space
	// Reserve space for: info box, optional exit box, optional error box, log box borders
	const infoBoxHeight = 6
	const logBoxBorder = 3

	_, alive, found := m.lookupService()
	exitBoxHeight := 0
	if found && !alive {
		exitBoxHeight = m.exitInfoHeight()
	}

	errBoxHeight := 0
	if errText := m.activeState().lastErr; errText != "" {
		errBoxHeight = 3
	}

	searchHeight := 0
	if m.searching {
		searchHeight = 1
	}

	reservedHeight := infoBoxHeight + exitBoxHeight + errBoxHeight + searchHeight + logBoxBorder
	vpHeight := m.height - reservedHeight
	if vpHeight < 3 {
		vpHeight = 3
	}

	m.vp.Width = maxInt(0, m.width-4) // Account for box borders
	m.vp.Height = vpHeight
	m.vp.HighPerformanceRendering = false
	m = m.refreshViewportContent(false)
	return m
}

// recalculateViewportHeight recalculates viewport height after state changes
func (m ServiceModel) recalculateViewportHeight() ServiceModel {
	return m.resizeViewport()
}

func (m ServiceModel) activeState() *logStreamState {
	if m.active == LogStderr {
		return &m.stderr
	}
	return &m.stdout
}

func (m ServiceModel) lookupService() (*state.ServiceRecord, bool, bool) {
	if m.last == nil || m.last.State == nil {
		return nil, false, false
	}
	for i := range m.last.State.Services {
		svc := &m.last.State.Services[i]
		if svc.Name == m.name {
			alive := false
			if m.last.Alive != nil {
				alive = m.last.Alive[svc.Name]
			}
			return svc, alive, true
		}
	}
	return nil, false, false
}

func (m ServiceModel) syncPathsFromSnapshot() ServiceModel {
	rec, _, found := m.lookupService()
	if !found || rec == nil {
		return m
	}
	m.stdout.path = rec.StdoutLog
	m.stderr.path = rec.StderrLog
	return m
}

func (m ServiceModel) syncExitInfoFromSnapshot() ServiceModel {
	rec, alive, found := m.lookupService()
	if !found || rec == nil {
		m.exitInfo = nil
		m.exitInfoErr = ""
		return m.recalculateViewportHeight()
	}
	if alive {
		m.exitInfo = nil
		m.exitInfoErr = ""
		return m.recalculateViewportHeight()
	}

	m.exitInfo = nil
	m.exitInfoErr = ""
	if rec.ExitInfo == "" {
		m.exitInfoErr = "no exit info recorded"
		return m.recalculateViewportHeight()
	}

	ei, err := state.ReadExitInfo(rec.ExitInfo)
	if err != nil {
		m.exitInfoErr = err.Error()
		return m.recalculateViewportHeight()
	}
	m.exitInfo = ei
	return m.recalculateViewportHeight()
}

func (m ServiceModel) renderStyledExitInfo(theme styles.Theme) string {
	if m.exitInfo == nil {
		msg := "unknown"
		if m.exitInfoErr != "" {
			msg = m.exitInfoErr
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Warning).
			Padding(0, 1).
			Width(m.width - 4).
			Render(lipgloss.JoinHorizontal(lipgloss.Center,
				theme.StatusDead.Render(styles.IconError),
				" ",
				theme.Title.Render("Exit: "),
				theme.TitleMuted.Render(msg),
			))
	}

	ei := m.exitInfo
	var lines []string

	// Exit status line
	exitKind := "unknown"
	exitIcon := styles.IconError
	if ei.Signal != "" {
		exitKind = "signal " + ei.Signal
		exitIcon = styles.IconWarning
	} else if ei.ExitCode != nil {
		if *ei.ExitCode == 0 {
			exitKind = "exit_code=0 (success)"
			exitIcon = styles.IconSuccess
		} else {
			exitKind = fmt.Sprintf("exit_code=%d", *ei.ExitCode)
		}
	}

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center,
		theme.StatusDead.Render(exitIcon),
		" ",
		theme.Title.Render("Exit: "),
		theme.TitleMuted.Render(exitKind),
		"  ",
		theme.TitleMuted.Render(fmt.Sprintf("PID %d", ei.PID)),
	))

	// Exited at
	if !ei.ExitedAt.IsZero() {
		lines = append(lines, theme.TitleMuted.Render("Exited: "+ei.ExitedAt.Format("2006-01-02 15:04:05")))
	}

	// Error message
	if ei.Error != "" {
		lines = append(lines, theme.StatusDead.Render("Error: "+ei.Error))
	}

	// Stderr tail
	stderrLines := ei.StderrTail
	if len(stderrLines) > 6 {
		stderrLines = stderrLines[len(stderrLines)-6:]
	}
	if len(stderrLines) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.TitleMuted.Render("Last stderr:"))
		for _, line := range stderrLines {
			// Truncate long lines
			if len(line) > m.width-8 {
				line = line[:m.width-11] + "..."
			}
			lines = append(lines, theme.StatusDead.Render("! "+line))
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Error).
		Padding(0, 1).
		Width(m.width - 4).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m ServiceModel) loadInitialTail() ServiceModel {
	m.stdout = m.loadTailForStream(m.stdout)
	m.stderr = m.loadTailForStream(m.stderr)
	m = m.refreshViewportContent(true)
	return m
}

func (m ServiceModel) loadTailForStream(s logStreamState) logStreamState {
	s.lastErr = ""
	s.lines = nil
	s.carry = ""
	s.offset = 0
	s.seenFirst = true

	if s.path == "" {
		s.lastErr = "missing log path"
		return s
	}

	lines, offset, err := readTailLines(s.path, m.tailLines, 2<<20)
	if err != nil {
		s.lastErr = err.Error()
		return s
	}
	s.lines = lines
	s.offset = offset
	return s
}

func (m ServiceModel) tickReadAll() ServiceModel {
	m.stdout = m.readNewBytes(m.stdout)
	m.stderr = m.readNewBytes(m.stderr)
	return m
}

func (m ServiceModel) readNewBytes(s logStreamState) logStreamState {
	s.lastErr = ""
	if s.path == "" {
		s.lastErr = "missing log path"
		return s
	}

	f, err := os.Open(s.path)
	if err != nil {
		s.lastErr = err.Error()
		return s
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		s.lastErr = err.Error()
		return s
	}
	size := info.Size()
	if size < s.offset {
		s.offset = 0
		s.lines = nil
		s.carry = ""
	}

	if _, err := f.Seek(s.offset, io.SeekStart); err != nil {
		s.lastErr = err.Error()
		return s
	}

	const maxRead = 256 << 10
	buf, err := io.ReadAll(io.LimitReader(f, maxRead))
	if err != nil {
		s.lastErr = err.Error()
		return s
	}
	if len(buf) == 0 {
		return s
	}
	s.offset += int64(len(buf))

	text := s.carry + string(buf)
	parts := strings.Split(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		s.carry = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	} else {
		s.carry = ""
		if len(parts) > 0 && parts[len(parts)-1] == "" {
			parts = parts[:len(parts)-1]
		}
	}
	for _, line := range parts {
		s.lines = append(s.lines, line)
	}
	if m.maxLines > 0 && len(s.lines) > m.maxLines {
		s.lines = append([]string{}, s.lines[len(s.lines)-m.maxLines:]...)
	}
	return s
}

func (m ServiceModel) refreshViewportContent(gotoBottom bool) ServiceModel {
	s := m.activeState()
	content := ""
	if len(s.lines) == 0 {
		content = "(no log lines yet)\n"
	} else {
		lines := s.lines
		if m.filter != "" {
			filtered := make([]string, 0, len(lines))
			for _, line := range lines {
				if strings.Contains(line, m.filter) {
					filtered = append(filtered, line)
				}
			}
			lines = filtered
		}
		if len(lines) == 0 {
			content = "(no matching lines)\n"
		} else {
			content = strings.Join(lines, "\n") + "\n"
		}
	}
	m.vp.SetContent(content)
	if gotoBottom && m.follow {
		m.vp.GotoBottom()
	}
	return m
}

func readTailLines(path string, tailLines int, maxBytes int64) ([]string, int64, error) {
	if tailLines <= 0 {
		tailLines = 200
	}
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	size := info.Size()
	start := int64(0)
	if size > maxBytes {
		start = size - maxBytes
	}

	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, 0, err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, err
	}
	if start > 0 {
		if i := bytes.IndexByte(b, '\n'); i >= 0 && i+1 < len(b) {
			b = b[i+1:]
		}
	}

	lines := strings.Split(string(b), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > tailLines {
		lines = append([]string{}, lines[len(lines)-tailLines:]...)
	}

	return lines, size, nil
}
