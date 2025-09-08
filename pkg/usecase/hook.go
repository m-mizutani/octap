package usecase

import (
	"context"
	"log/slog"
	"sync"

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

	logger.Debug("hookExecutor.Execute called",
		slog.String("event_type", string(event.Type)),
		slog.String("repository", event.Repository),
		slog.String("workflow", event.Workflow),
	)

	actions := h.getActionsForEvent(event.Type)
	logger.Debug("Got actions for event",
		slog.String("event_type", string(event.Type)),
		slog.Int("action_count", len(actions)),
	)

	// Use WaitGroup to ensure all hooks complete
	var wg sync.WaitGroup

	// For complete events, we need to wait for all actions to finish
	shouldWait := event.Type == model.HookCompleteSuccess || event.Type == model.HookCompleteFailure

	for i, action := range actions {
		logger.Debug("Executing action",
			slog.Int("index", i),
			slog.String("type", action.Type),
			slog.String("event", string(event.Type)),
			slog.Bool("will_wait", shouldWait),
		)

		wg.Add(1)
		// Execute all actions asynchronously with WaitGroup control
		go func(a model.Action, idx int) {
			defer wg.Done()

			logger.Debug("Starting action execution",
				slog.Int("index", idx),
				slog.String("type", a.Type),
			)

			if err := h.executeAction(ctx, a, event); err != nil {
				logger.Warn("Failed to execute hook action",
					slog.Int("index", idx),
					slog.String("type", a.Type),
					slog.String("event", string(event.Type)),
					slog.String("error", err.Error()),
				)
			} else {
				logger.Debug("Hook action executed successfully",
					slog.Int("index", idx),
					slog.String("type", a.Type),
					slog.String("event", string(event.Type)),
				)
			}
		}(action, i)
	}

	// Wait for completion only for complete events
	if shouldWait {
		logger.Debug("Waiting for all complete event hooks to finish")
		wg.Wait()
		logger.Debug("All complete event hooks finished")
	} else {
		// For individual workflow events, we don't wait
		// but we still need to ensure goroutines don't get killed immediately
		go func() {
			wg.Wait()
			logger.Debug("All individual event hooks finished in background")
		}()
	}

	return nil
}

// getActionsForEvent returns actions configured for the given event type
func (h *hookExecutor) getActionsForEvent(eventType model.HookEvent) []model.Action {
	if h.config == nil {
		logger := ctxlog.From(context.Background())
		logger.Debug("No config available for hooks")
		return nil
	}

	logger := ctxlog.From(context.Background())
	var actions []model.Action

	switch eventType {
	case model.HookCheckSuccess:
		actions = h.config.Hooks.CheckSuccess
		logger.Debug("Getting CheckSuccess actions", slog.Int("count", len(actions)))
	case model.HookCheckFailure:
		actions = h.config.Hooks.CheckFailure
		logger.Debug("Getting CheckFailure actions", slog.Int("count", len(actions)))
	case model.HookCompleteSuccess:
		actions = h.config.Hooks.CompleteSuccess
		logger.Debug("Getting CompleteSuccess actions", slog.Int("count", len(actions)))
	case model.HookCompleteFailure:
		actions = h.config.Hooks.CompleteFailure
		logger.Debug("Getting CompleteFailure actions", slog.Int("count", len(actions)))
	default:
		logger.Debug("Unknown event type", slog.String("event_type", string(eventType)))
		return nil
	}

	return actions
}

// executeAction executes a single action
func (h *hookExecutor) executeAction(ctx context.Context, action model.Action, event model.WorkflowEvent) error {
	logger := ctxlog.From(ctx)

	executor, ok := h.actions[action.Type]
	if !ok {
		logger.Warn("Unknown action type",
			slog.String("type", action.Type),
		)
		return nil
	}

	logger.Debug("Calling action executor",
		slog.String("action_type", action.Type),
		slog.Any("action_data", action.Data),
	)

	err := executor.Execute(ctx, action, event)
	if err != nil {
		logger.Debug("Action executor returned error",
			slog.String("action_type", action.Type),
			slog.String("error", err.Error()),
		)
	} else {
		logger.Debug("Action executor completed",
			slog.String("action_type", action.Type),
		)
	}
	return err
}
