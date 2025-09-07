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
	// Do nothing - we'll use inline updates instead
}

func (d *DisplayManager) Update(runs []*model.WorkflowRun, lastUpdate time.Time, interval time.Duration) {
	// Deduplicate runs by name (keep the latest one)
	runMap := make(map[string]*model.WorkflowRun)
	for _, run := range runs {
		existing, exists := runMap[run.Name]
		if !exists || run.UpdatedAt.After(existing.UpdatedAt) {
			runMap[run.Name] = run
		}
	}

	// Print each unique workflow status
	for _, run := range runMap {
		statusIcon := d.getStatusIcon(run.Status, run.Conclusion)
		statusText := d.getStatusText(run.Status, run.Conclusion)
		timeInfo := d.getTimeInfo(run)

		fmt.Printf("%s %s %s %s\n",
			statusIcon, run.Name, statusText, timeInfo)
	}
}


func (d *DisplayManager) ShowWaiting(commitSHA, repoName string) {
	fmt.Printf("⏳ Waiting for workflows to start...\n")
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
