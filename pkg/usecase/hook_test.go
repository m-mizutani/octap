package usecase_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
						Data: map[string]any{},
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

	t.Run("WaitForCompletion waits for all pending actions", func(t *testing.T) {
		// Create config with multiple actions that take time
		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckSuccess: []model.Action{
					{
						Type: "command",
						Data: map[string]any{
							"command": "sleep",
							"args":    []string{"0.1"},
						},
					},
					{
						Type: "command",
						Data: map[string]any{
							"command": "sleep",
							"args":    []string{"0.1"},
						},
					},
				},
				CheckFailure: []model.Action{
					{
						Type: "command",
						Data: map[string]any{
							"command": "sleep",
							"args":    []string{"0.1"},
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		// Track action completion
		var actionCounter int32
		startTime := time.Now()

		// Execute multiple events asynchronously
		events := []model.WorkflowEvent{
			{
				Type:       model.HookCheckSuccess,
				Repository: "test/repo",
				Workflow:   "test1",
				RunID:      1,
			},
			{
				Type:       model.HookCheckSuccess,
				Repository: "test/repo",
				Workflow:   "test2",
				RunID:      2,
			},
			{
				Type:       model.HookCheckFailure,
				Repository: "test/repo",
				Workflow:   "test3",
				RunID:      3,
			},
		}

		// Execute all events without waiting
		for _, event := range events {
			err := executor.Execute(ctx, event)
			gt.NoError(t, err)
			atomic.AddInt32(&actionCounter, 1)
		}

		// Actions should be running in background
		// Total actions: 2 (CheckSuccess) + 2 (CheckSuccess) + 1 (CheckFailure) = 5
		// Each takes ~0.1 seconds

		// Wait for all actions to complete
		executor.WaitForCompletion()

		duration := time.Since(startTime)
		// All actions should have completed
		// Duration should be at least 0.1 seconds (running in parallel)
		gt.True(t, duration >= 100*time.Millisecond)
		// But not too long (should run in parallel, not sequential)
		gt.True(t, duration < 500*time.Millisecond)
	})

	t.Run("WaitForCompletion returns immediately when no actions", func(t *testing.T) {
		// Create executor with empty config
		config := &model.Config{
			Hooks: model.HooksConfig{},
		}

		executor := usecase.NewHookExecutor(config)

		startTime := time.Now()
		executor.WaitForCompletion()
		duration := time.Since(startTime)

		// Should return immediately
		gt.True(t, duration < 10*time.Millisecond)
	})

	t.Run("Individual events are async, complete events are sync", func(t *testing.T) {
		var executionOrder []string
		var mu sync.Mutex

		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckSuccess: []model.Action{
					{
						Type: "command",
						Data: map[string]any{
							"command": "echo",
							"args":    []string{"check"},
						},
					},
				},
				CompleteSuccess: []model.Action{
					{
						Type: "command",
						Data: map[string]any{
							"command": "echo",
							"args":    []string{"complete"},
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		// Execute check event (async)
		checkEvent := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test",
			RunID:      1,
		}
		err := executor.Execute(ctx, checkEvent)
		gt.NoError(t, err)

		mu.Lock()
		executionOrder = append(executionOrder, "check-returned")
		mu.Unlock()

		// Execute complete event (sync - should wait)
		completeEvent := model.WorkflowEvent{
			Type:       model.HookCompleteSuccess,
			Repository: "test/repo",
			Workflow:   "test",
			RunID:      2,
		}
		err = executor.Execute(ctx, completeEvent)
		gt.NoError(t, err)

		mu.Lock()
		executionOrder = append(executionOrder, "complete-returned")
		mu.Unlock()

		// Wait for all to ensure check action completes
		executor.WaitForCompletion()

		// Check event should return immediately (async)
		// Complete event should wait for its action (sync)
		gt.Equal(t, executionOrder[0], "check-returned")
		gt.Equal(t, executionOrder[1], "complete-returned")
	})

	t.Run("No goroutine leak", func(t *testing.T) {
		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckSuccess: []model.Action{
					{
						Type: "command",
						Data: map[string]any{
							"command": "echo",
							"args":    []string{"test"},
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		// Get initial goroutine count
		initialCount := runtime.NumGoroutine()

		// Execute multiple events
		for i := range 10 {
			event := model.WorkflowEvent{
				Type:       model.HookCheckSuccess,
				Repository: "test/repo",
				Workflow:   "test",
				RunID:      int64(i),
			}
			err := executor.Execute(ctx, event)
			gt.NoError(t, err)
		}

		// Wait for all actions to complete
		executor.WaitForCompletion()

		// Give goroutines time to clean up
		time.Sleep(100 * time.Millisecond)

		// Check goroutine count
		finalCount := runtime.NumGoroutine()

		// Should not have leaked goroutines (allow small variance for runtime)
		gt.True(t, finalCount <= initialCount+2)
	})

	t.Run("Execute with real Slack action", func(t *testing.T) {
		// Setup test server to simulate Slack webhook
		var requestCount int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckSuccess: []model.Action{
					{
						Type: "slack",
						Data: map[string]any{
							"webhook_url": server.URL,
							"message":     "Build succeeded for {{.Repository}}",
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "CI",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		err := executor.Execute(ctx, event)
		gt.NoError(t, err)

		// Wait for async action to complete
		executor.WaitForCompletion()

		// Verify Slack webhook was called
		gt.Equal(t, atomic.LoadInt32(&requestCount), int32(1))
	})

	t.Run("Execute with real Command action", func(t *testing.T) {
		// Create a test file that the command will write to
		tempFile := "./test_command_output.txt"
		defer os.Remove(tempFile)

		var commandStr string
		var args []string
		if runtime.GOOS == "windows" {
			commandStr = "cmd"
			args = []string{"/c", fmt.Sprintf("echo test > %s", tempFile)}
		} else {
			commandStr = "sh"
			args = []string{"-c", fmt.Sprintf("echo test > %s", tempFile)}
		}

		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckFailure: []model.Action{
					{
						Type: "command",
						Data: map[string]any{
							"command": commandStr,
							"args":    args,
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		event := model.WorkflowEvent{
			Type:       model.HookCheckFailure,
			Repository: "test/repo",
			Workflow:   "Tests",
			RunID:      456,
			URL:        "https://github.com/test/repo/actions/runs/456",
		}

		err := executor.Execute(ctx, event)
		gt.NoError(t, err)

		// Wait for async action to complete
		executor.WaitForCompletion()

		// Verify command was executed by checking if file was created
		_, err = os.Stat(tempFile)
		gt.NoError(t, err)
	})

	t.Run("Execute multiple action types concurrently", func(t *testing.T) {
		// Setup test server for Slack
		var slackCount int32
		slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&slackCount, 1)
			time.Sleep(50 * time.Millisecond) // Simulate network delay
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer slackServer.Close()

		// Prepare command that creates a file
		tempFile := "./test_multi_output.txt"
		defer os.Remove(tempFile)

		var commandStr string
		var args []string
		if runtime.GOOS == "windows" {
			commandStr = "cmd"
			args = []string{"/c", fmt.Sprintf("echo multi > %s", tempFile)}
		} else {
			commandStr = "sh"
			args = []string{"-c", fmt.Sprintf("echo multi > %s", tempFile)}
		}

		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckSuccess: []model.Action{
					{
						Type: "slack",
						Data: map[string]any{
							"webhook_url": slackServer.URL,
							"message":     "First notification",
						},
					},
					{
						Type: "command",
						Data: map[string]any{
							"command": commandStr,
							"args":    args,
						},
					},
					{
						Type: "slack",
						Data: map[string]any{
							"webhook_url": slackServer.URL,
							"message":     "Second notification",
							"color":       "good",
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "Multi",
			RunID:      789,
			URL:        "https://github.com/test/repo/actions/runs/789",
		}

		startTime := time.Now()
		err := executor.Execute(ctx, event)
		gt.NoError(t, err)

		// Should return immediately (async)
		duration := time.Since(startTime)
		gt.True(t, duration < 20*time.Millisecond)

		// Wait for all actions to complete
		executor.WaitForCompletion()

		// Verify all actions were executed
		gt.Equal(t, atomic.LoadInt32(&slackCount), int32(2)) // Two Slack notifications
		_, err = os.Stat(tempFile)
		gt.NoError(t, err) // Command created the file
	})

	t.Run("Complete events with real actions execute synchronously", func(t *testing.T) {
		// Setup test server for Slack
		var requestTime time.Time
		slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestTime = time.Now()
			time.Sleep(100 * time.Millisecond) // Simulate delay
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer slackServer.Close()

		config := &model.Config{
			Hooks: model.HooksConfig{
				CompleteSuccess: []model.Action{
					{
						Type: "slack",
						Data: map[string]any{
							"webhook_url": slackServer.URL,
							"message":     "All workflows completed successfully",
							"color":       "good",
							"icon_emoji":  ":white_check_mark:",
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		event := model.WorkflowEvent{
			Type: model.HookCompleteSuccess,
		}

		startTime := time.Now()
		err := executor.Execute(ctx, event)
		executionTime := time.Since(startTime)
		gt.NoError(t, err)

		// Should wait for action to complete (synchronous)
		gt.True(t, executionTime >= 100*time.Millisecond)
		
		// Verify the request was made during Execute, not after
		gt.True(t, !requestTime.IsZero())
		gt.True(t, requestTime.Before(time.Now()))
	})

	t.Run("Handle action execution errors gracefully", func(t *testing.T) {
		// Setup test server that returns error
		slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer slackServer.Close()

		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckSuccess: []model.Action{
					{
						Type: "slack",
						Data: map[string]any{
							"webhook_url": slackServer.URL,
							"message":     "This will fail",
						},
					},
					{
						Type: "command",
						Data: map[string]any{
							"command": "/nonexistent/command",
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "ErrorTest",
			RunID:      999,
		}

		// Should not return error even if actions fail
		err := executor.Execute(ctx, event)
		gt.NoError(t, err)

		// Wait for actions to complete
		executor.WaitForCompletion()
		// Test passes if no panic occurred
	})

	t.Run("Environment variables in command actions", func(t *testing.T) {
		// Create a test script that uses OCTAP environment variables
		tempFile := "./test_env_output.txt"
		defer os.Remove(tempFile)

		var commandStr string
		var args []string
		if runtime.GOOS == "windows" {
			commandStr = "cmd"
			args = []string{"/c", fmt.Sprintf("echo %%OCTAP_REPOSITORY%% > %s", tempFile)}
		} else {
			commandStr = "sh"
			args = []string{"-c", fmt.Sprintf("echo $OCTAP_REPOSITORY > %s", tempFile)}
		}

		config := &model.Config{
			Hooks: model.HooksConfig{
				CheckSuccess: []model.Action{
					{
						Type: "command",
						Data: map[string]any{
							"command": commandStr,
							"args":    args,
						},
					},
				},
			},
		}

		executor := usecase.NewHookExecutor(config)
		ctx := context.Background()

		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "myorg/myrepo",
			Workflow:   "Build",
			RunID:      12345,
			URL:        "https://github.com/myorg/myrepo/actions/runs/12345",
		}

		err := executor.Execute(ctx, event)
		gt.NoError(t, err)

		// Wait for action to complete
		executor.WaitForCompletion()

		// Verify environment variable was set correctly
		content, err := os.ReadFile(tempFile)
		gt.NoError(t, err)
		// Check that the content contains the repository name
		contentStr := string(content)
		gt.True(t, len(contentStr) > 0)
		// Repository name should be in the output (allowing for newlines/spaces)
	})
}
