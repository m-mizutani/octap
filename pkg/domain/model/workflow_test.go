package model_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

func TestWorkflowRun(t *testing.T) {
	t.Run("WorkflowRun fields", func(t *testing.T) {
		now := time.Now()
		run := &model.WorkflowRun{
			ID:         12345,
			Name:       "Build and Test",
			Status:     model.WorkflowStatusCompleted,
			Conclusion: model.WorkflowConclusionSuccess,
			URL:        "https://github.com/owner/repo/actions/runs/12345",
			CreatedAt:  now,
			UpdatedAt:  now.Add(5 * time.Minute),
		}

		gt.Equal(t, run.ID, int64(12345))
		gt.Equal(t, run.Name, "Build and Test")
		gt.Equal(t, run.Status, model.WorkflowStatusCompleted)
		gt.Equal(t, run.Conclusion, model.WorkflowConclusionSuccess)
	})
}

func TestSummary(t *testing.T) {
	t.Run("Summary calculation", func(t *testing.T) {
		summary := &model.Summary{
			TotalRuns:    10,
			SuccessCount: 7,
			FailureCount: 2,
			OtherCount:   1,
			Duration:     30 * time.Minute,
		}

		gt.Equal(t, summary.TotalRuns, 10)
		gt.Equal(t, summary.SuccessCount, 7)
		gt.Equal(t, summary.FailureCount, 2)
		gt.Equal(t, summary.OtherCount, 1)
		gt.Equal(t, summary.Duration, 30*time.Minute)
	})
}
