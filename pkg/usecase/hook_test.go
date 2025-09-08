package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/domain/model"
	"github.com/m-mizutani/octap/pkg/usecase"
)

func TestHookExecutor(t *testing.T) {
	t.Run("Execute with nil config does not panic", func(t *testing.T) {
		executor := usecase.NewHookExecutor(nil)
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
		}

		err := executor.Execute(context.Background(), event)
		gt.NoError(t, err)
	})

	t.Run("Execute with empty config does not panic", func(t *testing.T) {
		config := &model.Config{}
		executor := usecase.NewHookExecutor(config)
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
		}

		err := executor.Execute(context.Background(), event)
		gt.NoError(t, err)
	})

	t.Run("Execute handles unknown action type gracefully", func(t *testing.T) {
		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckSuccess: []model.Action{
					{
						Type: "unknown",
						Data: map[string]interface{}{},
					},
				},
			},
		}
		executor := usecase.NewHookExecutor(config)
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
		}

		err := executor.Execute(context.Background(), event)
		gt.NoError(t, err)
	})
}
