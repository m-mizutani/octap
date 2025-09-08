package interfaces

import (
	"context"

	"github.com/m-mizutani/octap/pkg/domain/model"
)

// HookExecutor executes hooks based on workflow events
type HookExecutor interface {
	Execute(ctx context.Context, event model.WorkflowEvent) error
}

// ActionExecutor executes a specific action
type ActionExecutor interface {
	Execute(ctx context.Context, action model.Action, event model.WorkflowEvent) error
}
