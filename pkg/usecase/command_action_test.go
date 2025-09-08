package usecase_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/domain/model"
	"github.com/m-mizutani/octap/pkg/usecase"
)

func TestCommandAction(t *testing.T) {
	t.Run("Execute simple command", func(t *testing.T) {
		// Create action based on OS
		var action model.Action
		if runtime.GOOS == "windows" {
			action = model.Action{
				Type: "command",
				Data: map[string]interface{}{
					"command": "cmd",
					"args":    []string{"/c", "echo", "test"},
				},
			}
		} else {
			action = model.Action{
				Type: "command",
				Data: map[string]interface{}{
					"command": "echo",
					"args":    []string{"test"},
				},
			}
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		// Execute
		cmdAction := usecase.NewCommandAction()
		err := cmdAction.Execute(context.Background(), action, event)
		gt.NoError(t, err)
	})

	t.Run("Command with environment variables", func(t *testing.T) {
		// Create temp script file
		scriptContent := ""
		scriptPath := ""

		if runtime.GOOS == "windows" {
			scriptPath = "test_env.bat"
			scriptContent = "@echo %OCTAP_REPOSITORY%"
		} else {
			scriptPath = "./test_env.sh"
			scriptContent = "#!/bin/sh\necho $OCTAP_REPOSITORY"
		}

		err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
		gt.NoError(t, err)
		defer os.Remove(scriptPath)

		// Create action
		action := model.Action{
			Type: "command",
			Data: map[string]interface{}{
				"command": scriptPath,
			},
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		// Execute
		cmdAction := usecase.NewCommandAction()
		err = cmdAction.Execute(context.Background(), action, event)
		gt.NoError(t, err)
	})

	t.Run("Command with timeout", func(t *testing.T) {
		// Create action with very short timeout
		var action model.Action
		if runtime.GOOS == "windows" {
			action = model.Action{
				Type: "command",
				Data: map[string]interface{}{
					"command": "cmd",
					"args":    []string{"/c", "timeout", "/t", "10"},
					"timeout": "100ms",
				},
			}
		} else {
			action = model.Action{
				Type: "command",
				Data: map[string]interface{}{
					"command": "sleep",
					"args":    []string{"10"},
					"timeout": "100ms",
				},
			}
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		// Execute - should timeout
		cmdAction := usecase.NewCommandAction()
		start := time.Now()
		err := cmdAction.Execute(context.Background(), action, event)
		duration := time.Since(start)

		gt.Error(t, err)
		// Should timeout quickly (less than 1 second)
		gt.True(t, duration < 1*time.Second)
	})

	t.Run("Command with custom environment", func(t *testing.T) {
		// Create temp script file
		scriptContent := ""
		scriptPath := ""

		if runtime.GOOS == "windows" {
			scriptPath = "test_custom_env.bat"
			scriptContent = "@echo %CUSTOM_VAR%"
		} else {
			scriptPath = "./test_custom_env.sh"
			scriptContent = "#!/bin/sh\necho $CUSTOM_VAR"
		}

		err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
		gt.NoError(t, err)
		defer os.Remove(scriptPath)

		// Create action with custom env
		action := model.Action{
			Type: "command",
			Data: map[string]interface{}{
				"command": scriptPath,
				"env":     []string{"CUSTOM_VAR=custom_value"},
			},
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		// Execute
		cmdAction := usecase.NewCommandAction()
		err = cmdAction.Execute(context.Background(), action, event)
		gt.NoError(t, err)
	})

	t.Run("Invalid command", func(t *testing.T) {
		// Create action with non-existent command
		action := model.Action{
			Type: "command",
			Data: map[string]interface{}{
				"command": "/non/existent/command",
			},
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		// Execute - should return error
		cmdAction := usecase.NewCommandAction()
		err := cmdAction.Execute(context.Background(), action, event)
		gt.Error(t, err)
	})

	t.Run("Invalid action data", func(t *testing.T) {
		testCases := []struct {
			name string
			data map[string]interface{}
		}{
			{
				name: "missing command",
				data: map[string]interface{}{},
			},
			{
				name: "empty command",
				data: map[string]interface{}{
					"command": "",
				},
			},
			{
				name: "invalid args type",
				data: map[string]interface{}{
					"command": "echo",
					"args":    "not an array",
				},
			},
			{
				name: "invalid timeout format",
				data: map[string]interface{}{
					"command": "echo",
					"timeout": "invalid",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				action := model.Action{
					Type: "command",
					Data: tc.data,
				}

				event := model.WorkflowEvent{
					Type:       model.HookCheckSuccess,
					Repository: "test/repo",
					Workflow:   "test",
					RunID:      123,
				}

				cmdAction := usecase.NewCommandAction()
				err := cmdAction.Execute(context.Background(), action, event)
				gt.Error(t, err)
			})
		}
	})

	t.Run("Environment variable expansion in args", func(t *testing.T) {
		// Set test environment variable
		os.Setenv("TEST_VAR", "expanded_value")
		defer os.Unsetenv("TEST_VAR")

		// Create action
		var action model.Action
		if runtime.GOOS == "windows" {
			action = model.Action{
				Type: "command",
				Data: map[string]interface{}{
					"command": "cmd",
					"args":    []string{"/c", "echo", "$TEST_VAR"},
				},
			}
		} else {
			action = model.Action{
				Type: "command",
				Data: map[string]interface{}{
					"command": "echo",
					"args":    []string{"$TEST_VAR"},
				},
			}
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		// Execute
		cmdAction := usecase.NewCommandAction()
		err := cmdAction.Execute(context.Background(), action, event)
		gt.NoError(t, err)
	})
}

func TestCommandActionEnvironmentVariables(t *testing.T) {
	// Create temp script that outputs all OCTAP_ environment variables
	scriptContent := ""
	scriptPath := ""

	if runtime.GOOS == "windows" {
		scriptPath = "test_octap_env.bat"
		scriptContent = `@echo off
echo OCTAP_EVENT_TYPE=%OCTAP_EVENT_TYPE%
echo OCTAP_REPOSITORY=%OCTAP_REPOSITORY%
echo OCTAP_WORKFLOW=%OCTAP_WORKFLOW%
echo OCTAP_RUN_ID=%OCTAP_RUN_ID%
echo OCTAP_RUN_URL=%OCTAP_RUN_URL%`
	} else {
		scriptPath = "./test_octap_env.sh"
		scriptContent = `#!/bin/sh
echo "OCTAP_EVENT_TYPE=$OCTAP_EVENT_TYPE"
echo "OCTAP_REPOSITORY=$OCTAP_REPOSITORY"
echo "OCTAP_WORKFLOW=$OCTAP_WORKFLOW"
echo "OCTAP_RUN_ID=$OCTAP_RUN_ID"
echo "OCTAP_RUN_URL=$OCTAP_RUN_URL"`
	}

	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	gt.NoError(t, err)
	defer os.Remove(scriptPath)

	// Create action
	action := model.Action{
		Type: "command",
		Data: map[string]interface{}{
			"command": scriptPath,
		},
	}

	// Create event
	event := model.WorkflowEvent{
		Type:       model.HookCheckFailure,
		Repository: "owner/repo-name",
		Workflow:   "CI Pipeline",
		RunID:      987654321,
		URL:        "https://github.com/owner/repo-name/actions/runs/987654321",
	}

	// Execute
	cmdAction := usecase.NewCommandAction()
	err = cmdAction.Execute(context.Background(), action, event)
	gt.NoError(t, err)

	// The test verifies that the command executes successfully with all environment variables set
	// The actual output verification would require capturing stdout, which is logged
}

func TestCommandActionIntegration(t *testing.T) {
	t.Run("Chain multiple commands", func(t *testing.T) {
		// Create temp directory for test
		tempDir := "./test_output"
		err := os.MkdirAll(tempDir, 0755)
		gt.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Define test file path
		testFile := fmt.Sprintf("%s/test.txt", tempDir)

		// Create first action - write to file
		var action1 model.Action
		if runtime.GOOS == "windows" {
			action1 = model.Action{
				Type: "command",
				Data: map[string]interface{}{
					"command": "cmd",
					"args":    []string{"/c", fmt.Sprintf("echo test content > %s", testFile)},
				},
			}
		} else {
			action1 = model.Action{
				Type: "command",
				Data: map[string]interface{}{
					"command": "sh",
					"args":    []string{"-c", fmt.Sprintf("echo 'test content' > %s", testFile)},
				},
			}
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCompleteSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		// Execute first command
		cmdAction := usecase.NewCommandAction()
		err = cmdAction.Execute(context.Background(), action1, event)
		gt.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(testFile)
		gt.NoError(t, err)
	})
}
