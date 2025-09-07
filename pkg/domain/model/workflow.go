package model

import "time"

type WorkflowStatus string

const (
	WorkflowStatusQueued     WorkflowStatus = "queued"
	WorkflowStatusInProgress WorkflowStatus = "in_progress"
	WorkflowStatusCompleted  WorkflowStatus = "completed"
)

type WorkflowConclusion string

const (
	WorkflowConclusionSuccess   WorkflowConclusion = "success"
	WorkflowConclusionFailure   WorkflowConclusion = "failure"
	WorkflowConclusionCancelled WorkflowConclusion = "cancelled"
	WorkflowConclusionSkipped   WorkflowConclusion = "skipped"
	WorkflowConclusionTimedOut  WorkflowConclusion = "timed_out"
)

type WorkflowRun struct {
	ID         int64
	Name       string
	Status     WorkflowStatus
	Conclusion WorkflowConclusion
	URL        string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Summary struct {
	TotalRuns    int
	SuccessCount int
	FailureCount int
	OtherCount   int
	Duration     time.Duration
}
