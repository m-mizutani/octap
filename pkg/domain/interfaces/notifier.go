package interfaces

import (
	"context"

	"github.com/m-mizutani/octap/pkg/domain/model"
)

type Notifier interface {
	NotifySuccess(ctx context.Context, workflow *model.WorkflowRun) error
	NotifyFailure(ctx context.Context, workflow *model.WorkflowRun) error
	NotifyComplete(ctx context.Context, summary *model.Summary) error
	SetConfig(config *model.Config)
	// WaitForPendingActions waits for all pending hook actions to complete.
	// This should be called only when the process is about to exit.
	WaitForPendingActions()
}
