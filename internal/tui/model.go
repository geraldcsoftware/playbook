package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
	"github.com/geraldcsoftware/playbook/pkg/ssh"
)

type Action int

const (
	ActionNone Action = iota
	ActionRun
	ActionViewHosts
	ActionDoctor
	ActionQuit
)

type menuItem struct {
	label  string
	action Action
}

type Model struct {
	playbook      playbook.Playbook
	resolvedHosts []ssh.ResolvedHost
	resolveErrors []string
	menuItems     []menuItem
	cursor        int
	chosen        Action
}

func NewModel(pb playbook.Playbook, hosts []ssh.ResolvedHost, errors []string) Model {
	return Model{
		playbook:      pb,
		resolvedHosts: hosts,
		resolveErrors: errors,
		menuItems: []menuItem{
			{"Run playbook", ActionRun},
			{"View hosts", ActionViewHosts},
			{"Run doctor", ActionDoctor},
			{"Quit", ActionQuit},
		},
		chosen: ActionNone,
	}
}

func (m Model) ChosenAction() Action { return m.chosen }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.chosen = ActionQuit
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.menuItems[m.cursor].action
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	s := "\n"
	s += titleStyle.Render("  playbook") + "\n\n"

	s += labelStyle.Render("  Playbook:  ") + valueStyle.Render(m.playbook.Name) + "\n"
	s += labelStyle.Render("  File:      ") + valueStyle.Render(m.playbook.File) + "\n"

	if len(m.resolvedHosts) > 0 {
		s += labelStyle.Render("  Hosts:     ") + valueStyle.Render(m.resolvedHosts[0].Hostname) + "\n"
		for _, h := range m.resolvedHosts[1:] {
			s += labelStyle.Render("             ") + valueStyle.Render(h.Hostname) + "\n"
		}
	}
	for _, e := range m.resolveErrors {
		s += "  " + errorStyle.Render("! "+e) + "\n"
	}

	s += "\n"

	for i, item := range m.menuItems {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		s += "  " + style.Render(cursor+item.label) + "\n"
	}

	s += "\n"
	s += "  " + helpStyle.Render("up/down navigate  enter select  q quit") + "\n\n"

	return s
}

func Run(pb playbook.Playbook, hosts []ssh.ResolvedHost, errors []string) (Action, error) {
	m := NewModel(pb, hosts, errors)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ActionNone, fmt.Errorf("TUI error: %w", err)
	}
	return finalModel.(Model).ChosenAction(), nil
}
