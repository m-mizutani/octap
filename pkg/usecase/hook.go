package usecase

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type hookExecutor struct {
	config  *model.Config
	actions map[string]interfaces.ActionExecutor
}

// NewHookExecutor creates a new HookExecutor instance
func NewHookExecutor(config *model.Config) interfaces.HookExecutor {
	return &hookExecutor{
		config: config,
		actions: map[string]interfaces.ActionExecutor{
			"sound": NewSoundAction(),
		},
	}
}

// Execute runs hooks for the given event
func (h *hookExecutor) Execute(ctx context.Context, event model.WorkflowEvent) error {
	logger := ctxlog.From(ctx)

	actions := h.getActionsForEvent(event.Type)
	for _, action := range actions {
		// Execute action asynchronously
		go func(a model.Action) {
			if err := h.executeAction(ctx, a, event); err != nil {
				logger.Warn("Failed to execute hook action",
					slog.String("type", a.Type),
					slog.String("event", string(event.Type)),
					slog.String("error", err.Error()),
				)
			}
		}(action)
	}

	return nil
}

// getActionsForEvent returns actions configured for the given event type
func (h *hookExecutor) getActionsForEvent(eventType model.HookEvent) []model.Action {
	if h.config == nil {
		return nil
	}

	switch eventType {
	case model.HookCheckSuccess:
		return h.config.Hooks.CheckSuccess
	case model.HookCheckFailure:
		return h.config.Hooks.CheckFailure
	case model.HookCompleteSuccess:
		return h.config.Hooks.CompleteSuccess
	case model.HookCompleteFailure:
		return h.config.Hooks.CompleteFailure
	default:
		return nil
	}
}

// executeAction executes a single action
func (h *hookExecutor) executeAction(ctx context.Context, action model.Action, event model.WorkflowEvent) error {
	executor, ok := h.actions[action.Type]
	if !ok {
		logger := ctxlog.From(ctx)
		logger.Warn("Unknown action type",
			slog.String("type", action.Type),
		)
		return nil
	}

	return executor.Execute(ctx, action, event)
}
