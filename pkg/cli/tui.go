package cli

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99")).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.Color("240"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

type TUIModel struct {
	repoName   string
	commitSHA  string
	interval   time.Duration
	lastUpdate time.Time
	runs       []*model.WorkflowRun
	waiting    bool
	width      int
	height     int
}

type TickMsg time.Time
type UpdateDataMsg struct {
	Runs       []*model.WorkflowRun
	LastUpdate time.Time
	Waiting    bool
}

func NewTUIModel(repoName, commitSHA string, interval time.Duration) *TUIModel {
	return &TUIModel{
		repoName:  repoName,
		commitSHA: commitSHA,
		interval:  interval,
		waiting:   true,
	}
}

func (m *TUIModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
	)
}

func (m *TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case TickMsg:
		return m, tickCmd()

	case UpdateDataMsg:
		m.runs = msg.Runs
		m.lastUpdate = msg.LastUpdate
		m.waiting = msg.Waiting
		return m, nil
	}

	return m, nil
}

func (m *TUIModel) View() string {
	if m.width == 0 {
		return ""
	}

	header := headerStyle.Render("🔄 Monitoring GitHub Actions")
	info := fmt.Sprintf("Repository: %s | Commit: %s | Interval: %s",
		m.repoName, m.commitSHA[:8], m.interval)

	if m.waiting {
		waitMsg := "⏳ Waiting for workflows to start..."
		if !m.lastUpdate.IsZero() {
			waitMsg += " (last checked: " + m.lastUpdate.Format("15:04:05") + ")"
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			info,
			"",
			waitMsg,
			"",
			statusStyle.Render(fmt.Sprintf("Now: %s | Press 'q' to quit",
				time.Now().Format("15:04:05"))))
	}

	// Create table
	table := m.renderTable()

	now := time.Now()
	nextCheck := time.Until(m.lastUpdate.Add(m.interval))
	statusInfo := fmt.Sprintf("Now: %s | Last check: %s",
		now.Format("15:04:05"),
		m.lastUpdate.Format("15:04:05"))

	if nextCheck > 0 {
		statusInfo += fmt.Sprintf(" | Next in: %s", nextCheck.Round(time.Second))
	} else {
		statusInfo += " | Checking..."
	}
	statusInfo += " | Press 'q' to quit"

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		info,
		"",
		table,
		"",
		statusStyle.Render(statusInfo))
}

func (m *TUIModel) renderTable() string {
	if len(m.runs) == 0 {
		return "No workflows found"
	}

	// Deduplicate runs by name (keep the latest one)
	runMap := make(map[string]*model.WorkflowRun)
	for _, run := range m.runs {
		existing, exists := runMap[run.Name]
		if !exists || run.UpdatedAt.After(existing.UpdatedAt) {
			runMap[run.Name] = run
		}
	}

	// Create table rows
	var rows []string

	// Header
	headerRow := fmt.Sprintf("%-4s %-20s %-15s %-12s",
		"", "Workflow", "Status", "Time")
	rows = append(rows, tableHeaderStyle.Render(headerRow))

	for _, run := range runMap {
		icon := getStatusIcon(run.Status, run.Conclusion)
		name := run.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		status := getStatusText(run.Status, run.Conclusion)
		timeInfo := getTimeInfo(run)

		row := fmt.Sprintf("%-4s %-20s %-15s %-12s",
			icon, name, status, timeInfo)

		// Style row based on status
		style := lipgloss.NewStyle()
		if run.Status == model.WorkflowStatusCompleted {
			if run.Conclusion == model.WorkflowConclusionSuccess {
				style = style.Foreground(lipgloss.Color("10"))
			} else if run.Conclusion == model.WorkflowConclusionFailure {
				style = style.Foreground(lipgloss.Color("9"))
			}
		} else {
			style = style.Foreground(lipgloss.Color("11"))
		}

		rows = append(rows, style.Render(row))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func getStatusIcon(status model.WorkflowStatus, conclusion model.WorkflowConclusion) string {
	if status == model.WorkflowStatusCompleted {
		switch conclusion {
		case model.WorkflowConclusionSuccess:
			return "✅"
		case model.WorkflowConclusionFailure:
			return "❌"
		case model.WorkflowConclusionCancelled:
			return "⚪"
		case model.WorkflowConclusionSkipped:
			return "⏭️"
		default:
			return "❓"
		}
	}

	switch status {
	case model.WorkflowStatusQueued:
		return "⏳"
	case model.WorkflowStatusInProgress:
		return "🔄"
	default:
		return "❓"
	}
}

func getStatusText(status model.WorkflowStatus, conclusion model.WorkflowConclusion) string {
	if status == model.WorkflowStatusCompleted {
		return "[" + string(conclusion) + "]"
	}
	return "[" + string(status) + "]"
}

func getTimeInfo(run *model.WorkflowRun) string {
	if run.Status == model.WorkflowStatusCompleted {
		duration := time.Since(run.UpdatedAt)
		if duration < time.Minute {
			return fmt.Sprintf("%ds ago", int(duration.Seconds()))
		}
		if duration < time.Hour {
			return fmt.Sprintf("%dm ago", int(duration.Minutes()))
		}
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	}

	if run.Status == model.WorkflowStatusInProgress {
		return "Running..."
	}

	return "Waiting..."
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// TUIDisplay implements interfaces.Display using Bubble Tea
type TUIDisplay struct {
	program *tea.Program
	model   *TUIModel
}

func NewTUIDisplay(repoName, commitSHA string, interval time.Duration) interfaces.Display {
	model := NewTUIModel(repoName, commitSHA, interval)
	program := tea.NewProgram(model)

	return &TUIDisplay{
		program: program,
		model:   model,
	}
}

func (d *TUIDisplay) Clear() {
	// Not needed with Bubble Tea
}

func (d *TUIDisplay) Update(runs []*model.WorkflowRun, lastUpdate time.Time, interval time.Duration) {
	d.program.Send(UpdateDataMsg{
		Runs:       runs,
		LastUpdate: lastUpdate,
		Waiting:    len(runs) == 0,
	})
}

func (d *TUIDisplay) ShowWaiting(commitSHA, repoName string) {
	d.program.Send(UpdateDataMsg{
		Runs:       nil,
		LastUpdate: time.Now(),
		Waiting:    true,
	})
}

func (d *TUIDisplay) Run() error {
	_, err := d.program.Run()
	return err
}
