package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type DisplayManager struct {
	repoName     string
	commitSHA    string
	lastRunNames []string
	initialized  bool
}

func NewDisplayManager(repoName, commitSHA string) interfaces.Display {
	return &DisplayManager{
		repoName:  repoName,
		commitSHA: commitSHA,
	}
}

func (d *DisplayManager) Clear() {
	// Do nothing - we'll use inline updates instead
}

func (d *DisplayManager) Update(runs []*model.WorkflowRun, lastUpdate time.Time, interval time.Duration) {
	if len(runs) == 0 {
		if !d.initialized {
			fmt.Printf("⏳ No workflow runs found yet... (last checked: %s)\n", 
				lastUpdate.Format("15:04:05"))
			d.initialized = true
		}
		return
	}

	// Create current run names slice
	currentRunNames := make([]string, len(runs))
	for i, run := range runs {
		currentRunNames[i] = run.Name
	}

	// If this is first time or workflow list changed, print headers
	if !d.initialized || !d.sameWorkflows(currentRunNames) {
		if d.initialized {
			// Move cursor up to overwrite previous lines
			fmt.Printf("\033[%dA", len(d.lastRunNames))
		}
		d.lastRunNames = currentRunNames
		d.initialized = true
	} else {
		// Move cursor up to overwrite previous lines
		fmt.Printf("\033[%dA", len(runs))
	}

	// Show workflow status in one line per workflow
	for _, run := range runs {
		statusIcon := d.getStatusIcon(run.Status, run.Conclusion)
		statusText := d.getStatusText(run.Status, run.Conclusion)
		timeInfo := d.getTimeInfo(run)

		// Clear line and print status
		fmt.Printf("\033[2K%s %s %s %s\n",
			statusIcon, run.Name, statusText, timeInfo)
	}
}

func (d *DisplayManager) sameWorkflows(current []string) bool {
	if len(current) != len(d.lastRunNames) {
		return false
	}
	for i, name := range current {
		if name != d.lastRunNames[i] {
			return false
		}
	}
	return true
}

func (d *DisplayManager) ShowWaiting(commitSHA, repoName string) {
	fmt.Printf("⏳ Waiting for workflows to start... (repo: %s, commit: %s)\n", 
		repoName, commitSHA[:8])
}

func (d *DisplayManager) getStatusIcon(status model.WorkflowStatus, conclusion model.WorkflowConclusion) string {
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

func (d *DisplayManager) getStatusText(status model.WorkflowStatus, conclusion model.WorkflowConclusion) string {
	if status == model.WorkflowStatusCompleted {
		return fmt.Sprintf("[%s]", strings.ToLower(string(conclusion)))
	}
	return fmt.Sprintf("[%s]", strings.ReplaceAll(string(status), "_", " "))
}

func (d *DisplayManager) getTimeInfo(run *model.WorkflowRun) string {
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
