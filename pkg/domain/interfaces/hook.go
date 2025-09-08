package interfaces

import (
	"context"

	"github.com/m-mizutani/octap/pkg/domain/model"
)

// HookExecutor executes hooks based on workflow events
type HookExecutor interface {
	Execute(ctx context.Context, event model.WorkflowEvent) error
	// WaitForCompletion waits for all pending actions to complete.
	// This should be called only when the process is about to exit.
	WaitForCompletion()
}

// ActionExecutor executes a specific action
type ActionExecutor interface {
	Execute(ctx context.Context, action model.Action, event model.WorkflowEvent) error
}
