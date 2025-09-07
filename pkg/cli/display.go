package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type DisplayManager struct {
	repoName  string
	commitSHA string
}

func NewDisplayManager(repoName, commitSHA string) interfaces.Display {
	return &DisplayManager{
		repoName:  repoName,
		commitSHA: commitSHA,
	}
}

func (d *DisplayManager) Clear() {
	fmt.Print("\033[H\033[2J")
}

func (d *DisplayManager) Update(runs []*model.WorkflowRun, lastUpdate time.Time, interval time.Duration) {
	d.Clear()

	fmt.Printf("🔄 Monitoring GitHub Actions for commit: %s\n", d.commitSHA[:8])
	fmt.Printf("Repository: %s\n", d.repoName)
	fmt.Printf("Interval: %s\n\n", interval)

	if len(runs) == 0 {
		fmt.Printf("⏳ No workflow runs found yet...\n")
	} else {
		fmt.Printf("╭─ Workflow Runs ────────────────────────────────────╮\n")
		for _, run := range runs {
			statusIcon := d.getStatusIcon(run.Status, run.Conclusion)
			statusText := d.getStatusText(run.Status, run.Conclusion)
			timeInfo := d.getTimeInfo(run)

			name := run.Name
			if len(name) > 20 {
				name = name[:20] + "..."
			}

			fmt.Printf("│ %s %-23s %-13s %-10s │\n",
				statusIcon, name, statusText, timeInfo)
		}
		fmt.Printf("╰────────────────────────────────────────────────────╯\n")
	}

	fmt.Printf("\nLast updated: %s\n", lastUpdate.Format("2006-01-02 15:04:05"))
	nextCheck := time.Until(lastUpdate.Add(interval))
	if nextCheck > 0 {
		fmt.Printf("Next check in: %s\n", nextCheck.Round(time.Second))
	}
}

func (d *DisplayManager) ShowWaiting(commitSHA, repoName string) {
	d.Clear()
	fmt.Printf("⏳ Waiting for workflows to start...\n")
	fmt.Printf("Repository: %s\n", repoName)
	fmt.Printf("Commit: %s\n", commitSHA[:8])
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
