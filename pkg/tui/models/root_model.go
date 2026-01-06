package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/devctl/pkg/tui"
)

type ViewID string

const (
	ViewDashboard ViewID = "dashboard"
	ViewEvents    ViewID = "events"
)

type RootModel struct {
	width  int
	height int

	active ViewID

	dashboard DashboardModel
	events    EventLogModel
}

func NewRootModel() RootModel {
	return RootModel{
		active:    ViewDashboard,
		dashboard: NewDashboardModel(),
		events:    NewEventLogModel(),
	}
}

func (m RootModel) Init() tea.Cmd { return nil }

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
		return m, nil
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.active == ViewDashboard {
				m.active = ViewEvents
			} else {
				m.active = ViewDashboard
			}
			return m, nil
		}
	case tui.StateSnapshotMsg:
		m.dashboard = m.dashboard.WithSnapshot(v.Snapshot)
		return m, nil
	case tui.EventLogAppendMsg:
		m.events = m.events.Append(v.Entry)
		return m, nil
	}
	return m, nil
}

func (m RootModel) View() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("devctl tui — %s  (tab switch, q quit)\n\n", m.active))
	switch m.active {
	case ViewEvents:
		b.WriteString(m.events.View())
	default:
		b.WriteString(m.dashboard.View())
	}
	return b.String()
}
