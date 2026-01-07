package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/devctl/pkg/tui"
)

type PipelineModel struct {
	width  int
	height int

	runStarted  *tui.PipelineRunStarted
	runFinished *tui.PipelineRunFinished

	phaseOrder []tui.PipelinePhase
	phases     map[tui.PipelinePhase]*pipelinePhaseState

	buildSteps       []tui.PipelineStepResult
	prepareSteps     []tui.PipelineStepResult
	buildArtifacts   map[string]string
	prepareArtifacts map[string]string
	validate         *tui.PipelineValidateResult
	launchPlan       *tui.PipelineLaunchPlan

	focus            pipelineFocus
	buildCursor      int
	buildShow        bool
	prepareCursor    int
	prepareShow      bool
	validationCursor int
	validationShow   bool
}

type pipelineFocus string

const (
	pipelineFocusBuild      pipelineFocus = "build"
	pipelineFocusPrepare    pipelineFocus = "prepare"
	pipelineFocusValidation pipelineFocus = "validation"
)

type pipelinePhaseState struct {
	startedAt  time.Time
	finishedAt time.Time
	ok         *bool
	durationMs int64
	errText    string
}

func NewPipelineModel() PipelineModel {
	return PipelineModel{
		phases: map[tui.PipelinePhase]*pipelinePhaseState{},
	}
}

func (m PipelineModel) WithSize(width, height int) PipelineModel {
	m.width, m.height = width, height
	return m
}

func (m PipelineModel) Update(msg tea.Msg) (PipelineModel, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "b":
			m.focus = pipelineFocusBuild
			m.buildShow = true
			return m, nil
		case "p":
			m.focus = pipelineFocusPrepare
			m.prepareShow = true
			return m, nil
		case "v":
			m.focus = pipelineFocusValidation
			m.validationShow = true
			return m, nil
		case "up", "k":
			m = m.moveCursor(-1)
			return m, nil
		case "down", "j":
			m = m.moveCursor(1)
			return m, nil
		case "enter":
			m = m.toggleDetails()
			return m, nil
		default:
			return m, nil
		}
	case tui.PipelineRunStartedMsg:
		run := v.Run
		m.runStarted = &run
		m.runFinished = nil
		m.buildSteps = nil
		m.prepareSteps = nil
		m.buildArtifacts = nil
		m.prepareArtifacts = nil
		m.validate = nil
		m.launchPlan = nil
		m.focus = pipelineFocusBuild
		m.buildCursor = 0
		m.buildShow = false
		m.prepareCursor = 0
		m.prepareShow = false
		m.validationCursor = 0
		m.validationShow = false

		m.phases = map[tui.PipelinePhase]*pipelinePhaseState{}
		if len(run.Phases) > 0 {
			m.phaseOrder = append([]tui.PipelinePhase{}, run.Phases...)
		} else {
			m.phaseOrder = []tui.PipelinePhase{
				tui.PipelinePhaseMutateConfig,
				tui.PipelinePhaseBuild,
				tui.PipelinePhasePrepare,
				tui.PipelinePhaseValidate,
				tui.PipelinePhaseLaunchPlan,
				tui.PipelinePhaseSupervise,
				tui.PipelinePhaseStateSave,
			}
		}
		return m, nil
	case tui.PipelineRunFinishedMsg:
		if m.runStarted == nil || m.runStarted.RunID != v.Run.RunID {
			return m, nil
		}
		run := v.Run
		m.runFinished = &run
		return m, nil
	case tui.PipelinePhaseStartedMsg:
		if m.runStarted == nil || m.runStarted.RunID != v.Event.RunID {
			return m, nil
		}
		ph := m.phase(v.Event.Phase)
		ph.startedAt = v.Event.At
		ph.finishedAt = time.Time{}
		ph.ok = nil
		ph.durationMs = 0
		ph.errText = ""
		return m, nil
	case tui.PipelinePhaseFinishedMsg:
		if m.runStarted == nil || m.runStarted.RunID != v.Event.RunID {
			return m, nil
		}
		ph := m.phase(v.Event.Phase)
		ph.finishedAt = v.Event.At
		ph.durationMs = v.Event.DurationMs
		ok := v.Event.Ok
		ph.ok = &ok
		ph.errText = v.Event.Error
		return m, nil
	case tui.PipelineBuildResultMsg:
		if m.runStarted == nil || m.runStarted.RunID != v.Result.RunID {
			return m, nil
		}
		m.buildSteps = append([]tui.PipelineStepResult{}, v.Result.Steps...)
		m.buildArtifacts = copyStringMap(v.Result.Artifacts)
		if m.focus == "" {
			m.focus = pipelineFocusBuild
		}
		return m, nil
	case tui.PipelinePrepareResultMsg:
		if m.runStarted == nil || m.runStarted.RunID != v.Result.RunID {
			return m, nil
		}
		m.prepareSteps = append([]tui.PipelineStepResult{}, v.Result.Steps...)
		m.prepareArtifacts = copyStringMap(v.Result.Artifacts)
		return m, nil
	case tui.PipelineValidateResultMsg:
		if m.runStarted == nil || m.runStarted.RunID != v.Result.RunID {
			return m, nil
		}
		res := v.Result
		m.validate = &res
		m.validationCursor = 0
		m.validationShow = len(res.Errors) > 0 || len(res.Warnings) > 0
		return m, nil
	case tui.PipelineLaunchPlanMsg:
		if m.runStarted == nil || m.runStarted.RunID != v.Plan.RunID {
			return m, nil
		}
		plan := v.Plan
		m.launchPlan = &plan
		return m, nil
	default:
		return m, nil
	}
}

func (m PipelineModel) View() string {
	if m.runStarted == nil {
		return "No pipeline run recorded yet.\n\nRun `u` (up) or `r` (restart) from the dashboard to see progress here.\n"
	}

	run := m.runStarted
	status := "running"
	if m.runFinished != nil {
		if m.runFinished.Ok {
			status = "ok"
		} else {
			status = "failed"
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Pipeline: %s  run=%s  (%s)\n", run.Kind, run.RunID, status))
	b.WriteString(fmt.Sprintf("Started: %s\n\n", run.At.Format("2006-01-02 15:04:05")))
	if m.focus != "" {
		b.WriteString(fmt.Sprintf("Focus: %s  (b build, p prepare, v validation; ↑/↓ select; enter details)\n\n", m.focus))
	}

	b.WriteString("Phases:\n")
	for _, p := range m.phaseOrder {
		st := m.phases[p]
		b.WriteString(fmt.Sprintf("- %s: %s\n", p, formatPhaseState(st)))
	}
	b.WriteString("\n")

	if len(m.buildSteps) > 0 {
		b.WriteString("Build steps:\n")
		for i, s := range m.buildSteps {
			cursor := "-"
			if m.focus == pipelineFocusBuild {
				cursor = " "
				if i == clampInt(m.buildCursor, 0, maxInt(0, len(m.buildSteps)-1)) {
					cursor = ">"
				}
			}
			b.WriteString(fmt.Sprintf("%s %s: %s\n", cursor, s.Name, formatStep(s)))
		}
		if m.focus == pipelineFocusBuild && m.buildShow {
			m.renderBuildDetails(&b)
		}
		b.WriteString("\n")
	}

	if len(m.prepareSteps) > 0 {
		b.WriteString("Prepare steps:\n")
		for i, s := range m.prepareSteps {
			cursor := "-"
			if m.focus == pipelineFocusPrepare {
				cursor = " "
				if i == clampInt(m.prepareCursor, 0, maxInt(0, len(m.prepareSteps)-1)) {
					cursor = ">"
				}
			}
			b.WriteString(fmt.Sprintf("%s %s: %s\n", cursor, s.Name, formatStep(s)))
		}
		if m.focus == pipelineFocusPrepare && m.prepareShow {
			m.renderPrepareDetails(&b)
		}
		b.WriteString("\n")
	}

	if m.validate != nil {
		v := m.validate
		if v.Valid {
			b.WriteString(fmt.Sprintf("Validate: ok (%d warnings)\n\n", len(v.Warnings)))
		} else {
			b.WriteString(fmt.Sprintf("Validate: failed (%d errors, %d warnings)\n", len(v.Errors), len(v.Warnings)))
			if len(v.Errors) > 0 {
				first := v.Errors[0]
				b.WriteString(fmt.Sprintf("First error: %s: %s\n\n", first.Code, first.Message))
			} else {
				b.WriteString("\n")
			}
		}

		issues := validationIssues(v)
		if len(issues) > 0 {
			b.WriteString("Validation issues:\n")
			if m.focus != pipelineFocusValidation {
				b.WriteString("(press v to focus)\n")
			}
			for i, is := range issues {
				cursor := "-"
				if m.focus == pipelineFocusValidation {
					cursor = " "
					if i == clampInt(m.validationCursor, 0, maxInt(0, len(issues)-1)) {
						cursor = ">"
					}
				}
				b.WriteString(fmt.Sprintf("%s %s %s: %s\n", cursor, is.kind, is.code, is.message))
			}
			b.WriteString("\n")

			if m.focus == pipelineFocusValidation && m.validationShow {
				sel := issues[clampInt(m.validationCursor, 0, maxInt(0, len(issues)-1))]
				b.WriteString(fmt.Sprintf("Details: %s %s\n", sel.kind, sel.code))
				if sel.details == nil || len(sel.details) == 0 {
					b.WriteString("(no details)\n\n")
				} else {
					j, err := json.MarshalIndent(sel.details, "", "  ")
					if err != nil {
						b.WriteString(fmt.Sprintf("(failed to render details: %v)\n\n", err))
					} else {
						lines := strings.Split(string(j), "\n")
						const maxLines = 12
						if len(lines) > maxLines {
							lines = append(lines[:maxLines], "  ...")
						}
						for _, line := range lines {
							b.WriteString(line)
							b.WriteString("\n")
						}
						b.WriteString("\n")
					}
				}
			}
		}
	}

	if m.launchPlan != nil {
		b.WriteString(fmt.Sprintf("Launch plan: %d services\n", len(m.launchPlan.Services)))
		if len(m.launchPlan.Services) > 0 {
			b.WriteString(fmt.Sprintf("Services: %s\n", strings.Join(m.launchPlan.Services, ", ")))
		}
		b.WriteString("\n")
	}

	if m.runFinished != nil {
		f := m.runFinished
		if f.DurationMs > 0 {
			b.WriteString(fmt.Sprintf("Total: %s\n", formatDurationMs(f.DurationMs)))
		}
		if !f.Ok && f.Error != "" {
			b.WriteString(fmt.Sprintf("Error: %s\n", f.Error))
		}
	}

	return b.String()
}

func (m PipelineModel) moveCursor(delta int) PipelineModel {
	switch m.focus {
	case pipelineFocusPrepare:
		m.prepareCursor = clampInt(m.prepareCursor+delta, 0, maxInt(0, len(m.prepareSteps)-1))
	case pipelineFocusValidation:
		v := m.validate
		issues := validationIssues(v)
		m.validationCursor = clampInt(m.validationCursor+delta, 0, maxInt(0, len(issues)-1))
	default:
		m.buildCursor = clampInt(m.buildCursor+delta, 0, maxInt(0, len(m.buildSteps)-1))
	}
	return m
}

func (m PipelineModel) toggleDetails() PipelineModel {
	switch m.focus {
	case pipelineFocusPrepare:
		m.prepareShow = !m.prepareShow
	case pipelineFocusValidation:
		m.validationShow = !m.validationShow
	default:
		m.buildShow = !m.buildShow
	}
	return m
}

func (m PipelineModel) renderBuildDetails(b *strings.Builder) {
	if len(m.buildSteps) == 0 {
		return
	}
	idx := clampInt(m.buildCursor, 0, len(m.buildSteps)-1)
	sel := m.buildSteps[idx]
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Details: build step %q\n", sel.Name))
	b.WriteString(fmt.Sprintf("- ok: %v\n", sel.Ok))
	if sel.DurationMs > 0 {
		b.WriteString(fmt.Sprintf("- duration: %s\n", formatDurationMs(sel.DurationMs)))
	}
	if len(m.buildArtifacts) > 0 {
		b.WriteString(fmt.Sprintf("- artifacts: %d\n", len(m.buildArtifacts)))
	}
}

func (m PipelineModel) renderPrepareDetails(b *strings.Builder) {
	if len(m.prepareSteps) == 0 {
		return
	}
	idx := clampInt(m.prepareCursor, 0, len(m.prepareSteps)-1)
	sel := m.prepareSteps[idx]
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Details: prepare step %q\n", sel.Name))
	b.WriteString(fmt.Sprintf("- ok: %v\n", sel.Ok))
	if sel.DurationMs > 0 {
		b.WriteString(fmt.Sprintf("- duration: %s\n", formatDurationMs(sel.DurationMs)))
	}
	if len(m.prepareArtifacts) > 0 {
		b.WriteString(fmt.Sprintf("- artifacts: %d\n", len(m.prepareArtifacts)))
	}
}

type validationIssue struct {
	kind    string
	code    string
	message string
	details map[string]any
}

func validationIssues(v *tui.PipelineValidateResult) []validationIssue {
	if v == nil {
		return nil
	}
	out := make([]validationIssue, 0, len(v.Errors)+len(v.Warnings))
	for _, e := range v.Errors {
		out = append(out, validationIssue{
			kind:    "error",
			code:    e.Code,
			message: e.Message,
			details: e.Details,
		})
	}
	for _, e := range v.Warnings {
		out = append(out, validationIssue{
			kind:    "warn",
			code:    e.Code,
			message: e.Message,
			details: e.Details,
		})
	}
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (m PipelineModel) phase(p tui.PipelinePhase) *pipelinePhaseState {
	if m.phases == nil {
		m.phases = map[tui.PipelinePhase]*pipelinePhaseState{}
	}
	st := m.phases[p]
	if st == nil {
		st = &pipelinePhaseState{}
		m.phases[p] = st
	}
	return st
}

func formatPhaseState(st *pipelinePhaseState) string {
	if st == nil {
		return "pending"
	}
	if st.ok == nil && !st.startedAt.IsZero() {
		return "running"
	}
	if st.ok == nil {
		return "pending"
	}
	state := "ok"
	if !*st.ok {
		state = "failed"
	}
	if st.durationMs > 0 {
		state = fmt.Sprintf("%s (%s)", state, formatDurationMs(st.durationMs))
	}
	if st.errText != "" && !*st.ok {
		state = fmt.Sprintf("%s: %s", state, st.errText)
	}
	return state
}

func formatStep(s tui.PipelineStepResult) string {
	state := "ok"
	if !s.Ok {
		state = "failed"
	}
	if s.DurationMs > 0 {
		state = fmt.Sprintf("%s (%s)", state, formatDurationMs(s.DurationMs))
	}
	return state
}

func formatDurationMs(ms int64) string {
	if ms <= 0 {
		return "0s"
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	sec := float64(d) / float64(time.Second)
	if sec < 10 {
		return fmt.Sprintf("%.1fs", sec)
	}
	return fmt.Sprintf("%.0fs", sec)
}
