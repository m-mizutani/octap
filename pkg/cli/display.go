package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

// DisplayManager provides progress-based status display
type DisplayManager struct {
	repoName       string
	commitSHA      string
	initialRuns    map[string]*model.WorkflowRun
	currentRuns    map[string]*model.WorkflowRun
	totalCount     int
	completedCount int
	firstDisplay   bool
	lastCheckTime  time.Time
}

func NewDisplayManager(repoName, commitSHA string) interfaces.ExtendedDisplay {
	return &DisplayManager{
		repoName:     repoName,
		commitSHA:    commitSHA,
		initialRuns:  make(map[string]*model.WorkflowRun),
		currentRuns:  make(map[string]*model.WorkflowRun),
		firstDisplay: true,
	}
}

func (d *DisplayManager) Clear() {
	// Not needed for this display
}

func (d *DisplayManager) Update(runs []*model.WorkflowRun, lastUpdate time.Time, interval time.Duration) {
	d.lastCheckTime = lastUpdate

	// Deduplicate runs by name (keep the latest one)
	newRuns := make(map[string]*model.WorkflowRun)
	for _, run := range runs {
		existing, exists := newRuns[run.Name]
		if !exists || run.UpdatedAt.After(existing.UpdatedAt) {
			newRuns[run.Name] = run
		}
	}

	// First display - show all workflows
	if d.firstDisplay {
		d.firstDisplay = false
		d.initialRuns = newRuns
		d.currentRuns = newRuns
		d.totalCount = len(newRuns)

		if len(newRuns) == 0 {
			fmt.Printf("⏳ Waiting for workflows to start for commit %s...\n", d.commitSHA[:8])
			return
		}

		// Count initial completed
		d.completedCount = 0
		for _, run := range newRuns {
			if run.Status == model.WorkflowStatusCompleted {
				d.completedCount++
			}
		}

		fmt.Println("\n📋 Workflow Status:")
		fmt.Println(strings.Repeat("─", 50))

		for _, run := range newRuns {
			d.printWorkflowLine(run)
		}
		fmt.Println(strings.Repeat("─", 50))

		// Show initial progress
		progressBar := d.getProgressBar()
		fmt.Printf("%s %s\n", progressBar, getProgressText(d.completedCount, d.totalCount))

		return
	}

	// Check for status changes
	hasChanges := false
	var changedRuns []*model.WorkflowRun
	newCompletedCount := 0

	for name, run := range newRuns {
		if run.Status == model.WorkflowStatusCompleted {
			newCompletedCount++
		}

		oldRun, exists := d.currentRuns[name]
		if !exists || oldRun.Status != run.Status || oldRun.Conclusion != run.Conclusion {
			hasChanges = true
			changedRuns = append(changedRuns, run)
		}
	}

	// Update current state
	d.currentRuns = newRuns
	d.completedCount = newCompletedCount

	// Update totalCount if new workflows appeared
	if len(newRuns) > d.totalCount {
		d.totalCount = len(newRuns)
	}

	// If there are changes, show them
	if hasChanges {
		// Clear the countdown line
		fmt.Print("\r\033[K")

		// Show progress and changes
		timestamp := time.Now().Format("15:04:05")
		progressBar := d.getProgressBar()
		fmt.Printf("\n%s %s [%s]\n", progressBar, getProgressText(d.completedCount, d.totalCount), timestamp)

		for _, run := range changedRuns {
			fmt.Printf("  └─ ")
			d.printWorkflowLine(run)
		}
	}
}

func (d *DisplayManager) ShowWaiting(commitSHA, repoName string) {
	// Not used in this implementation
}

func (d *DisplayManager) ShowCountdown(remaining time.Duration) {
	// Show countdown on the same line
	fmt.Printf("\r\033[K⏱️  Next check in: %s", formatDuration(remaining))
}

func (d *DisplayManager) ShowFinalSummary() {
	fmt.Print("\r\033[K") // Clear countdown line

	fmt.Println("\n" + strings.Repeat("═", 50))
	fmt.Println("✨ All workflows completed!")
	fmt.Println(strings.Repeat("═", 50))

	successCount := 0
	failureCount := 0
	otherCount := 0

	for _, run := range d.currentRuns {
		if run.Status == model.WorkflowStatusCompleted {
			switch run.Conclusion {
			case model.WorkflowConclusionSuccess:
				successCount++
			case model.WorkflowConclusionFailure:
				failureCount++
			default:
				otherCount++
			}
		}
	}

	fmt.Printf("📊 Results: ")
	if successCount > 0 {
		color.New(color.FgGreen).Printf("✅ %d success ", successCount)
	}
	if failureCount > 0 {
		color.New(color.FgRed).Printf("❌ %d failed ", failureCount)
	}
	if otherCount > 0 {
		color.New(color.FgYellow).Printf("⚠️  %d other", otherCount)
	}
	fmt.Println()
}

func (d *DisplayManager) printWorkflowLine(run *model.WorkflowRun) {
	icon := getWorkflowIcon(run.Status, run.Conclusion)
	statusText := getWorkflowStatusText(run.Status, run.Conclusion)

	var statusColor *color.Color
	switch run.Status {
	case model.WorkflowStatusCompleted:
		switch run.Conclusion {
		case model.WorkflowConclusionSuccess:
			statusColor = color.New(color.FgGreen)
		case model.WorkflowConclusionFailure:
			statusColor = color.New(color.FgRed)
		default:
			statusColor = color.New(color.FgYellow)
		}
	case model.WorkflowStatusInProgress:
		statusColor = color.New(color.FgCyan)
	default:
		statusColor = color.New(color.FgWhite)
	}

	fmt.Printf("%s ", icon)
	statusColor.Printf("%-20s %s", run.Name, statusText)

	// Show URL for failed workflows
	if run.Status == model.WorkflowStatusCompleted && run.Conclusion == model.WorkflowConclusionFailure {
		fmt.Printf(" 🔗 %s", run.URL)
	}
	fmt.Println()
}

func (d *DisplayManager) getProgressBar() string {
	if d.totalCount == 0 {
		return "⏳"
	}

	percentage := float64(d.completedCount) / float64(d.totalCount)

	// Determine icon based on completion status
	allSuccess := true
	hasFailure := false
	for _, run := range d.currentRuns {
		if run.Status == model.WorkflowStatusCompleted {
			if run.Conclusion != model.WorkflowConclusionSuccess {
				allSuccess = false
				if run.Conclusion == model.WorkflowConclusionFailure {
					hasFailure = true
				}
			}
		}
	}

	if percentage == 1.0 {
		if allSuccess {
			return "✅"
		} else if hasFailure {
			return "❌"
		} else {
			return "⚠️"
		}
	}

	return "🔄"
}

func getWorkflowIcon(status model.WorkflowStatus, conclusion model.WorkflowConclusion) string {
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
	case model.WorkflowStatusInProgress:
		return "🔄"
	case model.WorkflowStatusQueued:
		return "⏳"
	default:
		return "❓"
	}
}

func getWorkflowStatusText(status model.WorkflowStatus, conclusion model.WorkflowConclusion) string {
	if status == model.WorkflowStatusCompleted {
		return fmt.Sprintf("[%s]", conclusion)
	}
	return fmt.Sprintf("[%s]", status)
}

func getProgressText(completed, total int) string {
	return fmt.Sprintf("%d/%d completed", completed, total)
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 0 {
		return "0s"
	}
	return fmt.Sprintf("%ds", seconds)
}
