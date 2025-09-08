package model

// HookEvent represents a type of workflow event
type HookEvent string

const (
	HookCheckSuccess    HookEvent = "check_success"
	HookCheckFailure    HookEvent = "check_failure"
	HookCompleteSuccess HookEvent = "complete_success"
	HookCompleteFailure HookEvent = "complete_failure"
)

// WorkflowEvent contains information about a workflow event
type WorkflowEvent struct {
	Type       HookEvent
	Repository string
	Workflow   string
	RunID      int64
	URL        string
}
