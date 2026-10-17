package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
)

// Poll cadences. Variables so tests can shorten them.
var (
	idleRefresh   = 10 * time.Second // passive refresh on the detail screen
	activeRefresh = 3 * time.Second  // while waiting for a restart/stop
)

// rowItem is a generic one-line list row used by the settings, backups and
// files screens.
type rowItem struct {
	title, desc string
	data        any
}

func (r rowItem) Title() string       { return r.title }
func (r rowItem) Description() string { return r.desc }
func (r rowItem) FilterValue() string { return r.title }

// waitState tracks a restart/stop we're waiting to see complete.
type waitState struct {
	verb  string // "restart" / "stop"
	watch *api.StatusWatch
	start time.Time
}

func (w *waitState) describe(now time.Time) string {
	target := "to be running again"
	if w.verb == "stop" {
		target = "to stop"
	}
	return fmt.Sprintf("Waiting for the server %s… (%ds)", target, int(now.Sub(w.start).Seconds()))
}

// startWaiting begins watching for a restart or stop to complete, driving
// fast polls until it does (or times out).
func (m *model) startWaiting(verb string, now time.Time) tea.Cmd {
	opts := api.WaitOptions{Want: func(s string) bool { return s == api.StatusStarted }}
	if verb == "restart" {
		opts.LeaveFirst = api.StatusStarted
		opts.GracePeriod = 20 * time.Second
	} else {
		opts.Want = func(s string) bool { return s == api.StatusStopped }
	}
	m.waiting = &waitState{verb: verb, watch: api.NewStatusWatch(opts, now), start: now}
	return m.startChain(0)
}

// startChain begins a fresh poll chain (invalidating any older one) whose
// first poll fires after delay. Called whenever the detail screen becomes
// active or a wait begins.
func (m *model) startChain(delay time.Duration) tea.Cmd {
	m.gen++
	gen := m.gen
	if delay <= 0 {
		return pollStatus(m.client, m.selectedSvc.ID, gen)
	}
	return tea.Tick(delay, func(time.Time) tea.Msg { return tickMsg{gen: gen} })
}

// enterDetail switches to the server detail screen and (re)starts the
// passive refresh chain.
func (m *model) enterDetail() tea.Cmd {
	m.screen = screenServerDetail
	return m.startChain(idleRefresh)
}

// confirmState is a modal asking the user to confirm an action, optionally
// collecting a free-text message.
type confirmState struct {
	prompt    string
	withInput bool
	input     textinput.Model
	run       func(m model, text string) (model, tea.Cmd)
}

func newConfirm(prompt string, run func(m model, text string) (model, tea.Cmd)) *confirmState {
	return &confirmState{prompt: prompt, run: run}
}

func newConfirmWithMessage(prompt string, run func(m model, text string) (model, tea.Cmd)) *confirmState {
	ti := textinput.New()
	ti.Prompt = "message to players (optional): "
	ti.CharLimit = 200
	ti.Focus()
	return &confirmState{prompt: prompt, withInput: true, input: ti, run: run}
}

// handleConfirmKey processes a key while a confirmation modal is open.
func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	c := m.confirm
	if c.withInput {
		switch msg.Type {
		case tea.KeyEnter:
			m.confirm = nil
			nm, cmd := c.run(m, strings.TrimSpace(c.input.Value()))
			return nm, cmd
		case tea.KeyEsc:
			m.confirm = nil
			return m, nil
		}
		var cmd tea.Cmd
		c.input, cmd = c.input.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "y", "Y", "enter":
		m.confirm = nil
		nm, cmd := c.run(m, "")
		return nm, cmd
	case "n", "N", "esc":
		m.confirm = nil
	}
	return m, nil
}

func (m model) viewConfirm() string {
	c := m.confirm
	var b strings.Builder
	b.WriteString(m.styles.Warning.Render(c.prompt))
	b.WriteString("\n\n")
	if c.withInput {
		b.WriteString(c.input.View())
		b.WriteString("\n\n")
		b.WriteString(m.styles.Help.Render("enter: confirm  •  esc: cancel"))
	} else {
		b.WriteString(m.styles.Help.Render("y: yes  •  n: cancel"))
	}
	return m.styles.Border.Render(b.String())
}
