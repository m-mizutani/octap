package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

// InlineDisplayManager provides compact, non-fullscreen status display
type InlineDisplayManager struct {
	repoName     string
	commitSHA    string
	lastRuns     map[string]*model.WorkflowRun
	spinner      *spinner.Spinner
	hasWorkflows bool
	firstCheck   bool // Track if we've done the first check yet
}

func NewInlineDisplayManager(repoName, commitSHA string) interfaces.Display {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "⏳ "
	s.Suffix = " Waiting for workflows..."

	return &InlineDisplayManager{
		repoName:   repoName,
		commitSHA:  commitSHA,
		lastRuns:   make(map[string]*model.WorkflowRun),
		spinner:    s,
		firstCheck: true, // Start with first check true
	}
}

func (d *InlineDisplayManager) Clear() {
	// Not needed for inline display
}

func (d *InlineDisplayManager) Update(runs []*model.WorkflowRun, lastUpdate time.Time, interval time.Duration) {
	// Mark that we've done at least one check
	defer func() { d.firstCheck = false }()

	if len(runs) == 0 {
		// Only start spinner if this is NOT the first check and we haven't found workflows yet
		if !d.firstCheck && !d.hasWorkflows && !d.spinner.Active() {
			d.spinner.Start()
		}
		return
	}

	// Stop spinner if workflows found
	if d.spinner.Active() {
		d.spinner.Stop()
		fmt.Print("\r\033[K") // Clear spinner line
	}
	d.hasWorkflows = true

	// Deduplicate runs by name (keep the latest one)
	currentRuns := make(map[string]*model.WorkflowRun)
	for _, run := range runs {
		existing, exists := currentRuns[run.Name]
		if !exists || run.UpdatedAt.After(existing.UpdatedAt) {
			currentRuns[run.Name] = run
		}
	}

	// Check for status changes and print updates
	for name, run := range currentRuns {
		lastRun, existed := d.lastRuns[name]

		// Print only if status changed or it's new
		if !existed || lastRun.Status != run.Status || lastRun.Conclusion != run.Conclusion {
			d.printWorkflowStatus(run)
		}
	}

	// Update stored runs
	d.lastRuns = currentRuns

	// Print summary status line
	d.printStatusSummary(currentRuns, lastUpdate, interval)
}

func (d *InlineDisplayManager) ShowWaiting(commitSHA, repoName string) {
	// Only show waiting message if we haven't found any workflows yet and not on first check
	if !d.firstCheck && !d.hasWorkflows && !d.spinner.Active() {
		d.spinner.Start()
	}
}

func (d *InlineDisplayManager) printWorkflowStatus(run *model.WorkflowRun) {
	var statusColor *color.Color
	var icon string

	switch run.Status {
	case model.WorkflowStatusCompleted:
		switch run.Conclusion {
		case model.WorkflowConclusionSuccess:
			statusColor = color.New(color.FgGreen)
			icon = "✅"
		case model.WorkflowConclusionFailure:
			statusColor = color.New(color.FgRed)
			icon = "❌"
		case model.WorkflowConclusionCancelled:
			statusColor = color.New(color.FgYellow)
			icon = "⚪"
		case model.WorkflowConclusionSkipped:
			statusColor = color.New(color.FgBlue)
			icon = "⏭️"
		default:
			statusColor = color.New(color.FgMagenta)
			icon = "❓"
		}
	case model.WorkflowStatusInProgress:
		statusColor = color.New(color.FgYellow)
		icon = "🔄"
	case model.WorkflowStatusQueued:
		statusColor = color.New(color.FgCyan)
		icon = "⏳"
	default:
		statusColor = color.New(color.FgWhite)
		icon = "❓"
	}

	timeInfo := d.getTimeInfo(run)
	statusText := d.getStatusText(run.Status, run.Conclusion)

	statusColor.Printf("%s %s %s %s\n",
		icon, run.Name, statusText, timeInfo)
}

func (d *InlineDisplayManager) printStatusSummary(runs map[string]*model.WorkflowRun, lastUpdate time.Time, interval time.Duration) {
	if len(runs) == 0 {
		return
	}

	var running, completed, failed int
	for _, run := range runs {
		switch run.Status {
		case model.WorkflowStatusCompleted:
			if run.Conclusion == model.WorkflowConclusionSuccess {
				completed++
			} else if run.Conclusion == model.WorkflowConclusionFailure {
				failed++
			} else {
				// Other conclusions like cancelled, skipped, etc. are not counted as failed
				completed++
			}
		case model.WorkflowStatusInProgress, model.WorkflowStatusQueued:
			running++
		}
	}

	// Create compact status line
	var parts []string
	if completed > 0 {
		parts = append(parts, color.New(color.FgGreen).Sprintf("✅ %d completed", completed))
	}
	if failed > 0 {
		parts = append(parts, color.New(color.FgRed).Sprintf("❌ %d failed", failed))
	}
	if running > 0 {
		parts = append(parts, color.New(color.FgYellow).Sprintf("🔄 %d running", running))
	}

	nextCheck := time.Until(lastUpdate.Add(interval))
	timeInfo := fmt.Sprintf("next: %s", nextCheck.Round(time.Second))
	if nextCheck <= 0 {
		timeInfo = "checking..."
	}

	statusLine := fmt.Sprintf("[%s | %s]",
		strings.Join(parts, " "),
		color.New(color.FgCyan).Sprint(timeInfo))

	// Print status line with carriage return to allow overwrite
	fmt.Printf("\r\033[K%s", statusLine)
}

func (d *InlineDisplayManager) getStatusText(status model.WorkflowStatus, conclusion model.WorkflowConclusion) string {
	if status == model.WorkflowStatusCompleted {
		return "[" + string(conclusion) + "]"
	}
	return "[" + string(status) + "]"
}

func (d *InlineDisplayManager) getTimeInfo(run *model.WorkflowRun) string {
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
		return "running..."
	}

	return "queued..."
}

// Stop spinner and clean up display
func (d *InlineDisplayManager) Stop() {
	if d.spinner.Active() {
		d.spinner.Stop()
		fmt.Print("\r\033[K") // Clear current line
	}
}
