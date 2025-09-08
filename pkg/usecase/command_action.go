package usecase

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type commandAction struct{}

// NewCommandAction creates a new CommandAction instance
func NewCommandAction() interfaces.ActionExecutor {
	return &commandAction{}
}

// Execute runs a command
func (c *commandAction) Execute(ctx context.Context, action model.Action, event model.WorkflowEvent) error {
	logger := ctxlog.From(ctx)

	logger.Debug("commandAction.Execute called",
		slog.String("event_type", string(event.Type)),
		slog.Any("action_data", action.Data),
	)

	// Convert to typed action
	cmdAction, err := action.ToCommandAction()
	if err != nil {
		logger.Error("Failed to parse command action",
			slog.String("error", err.Error()),
		)
		return goerr.Wrap(err, "failed to parse command action")
	}

	// Prepare environment variables
	env := c.prepareEnv(event)
	if len(cmdAction.Env) > 0 {
		env = append(env, cmdAction.Env...)
	}

	// Set default timeout if not specified
	timeout := cmdAction.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Execute command
	err = c.executeCommand(ctx, cmdAction, env, timeout)
	if err != nil {
		logger.Error("Command execution failed",
			slog.String("command", cmdAction.Command),
			slog.Any("args", cmdAction.Args),
			slog.Duration("timeout", timeout),
			slog.String("error", err.Error()),
		)
		return goerr.Wrap(err, "command execution failed")
	}

	logger.Debug("Command executed successfully",
		slog.String("command", cmdAction.Command),
		slog.Any("args", cmdAction.Args),
	)
	return nil
}

// prepareEnv prepares environment variables for command execution
func (c *commandAction) prepareEnv(event model.WorkflowEvent) []string {
	// Start with current environment
	env := os.Environ()

	// Add octap-specific environment variables
	octapEnv := map[string]string{
		"OCTAP_EVENT_TYPE": string(event.Type),
		"OCTAP_REPOSITORY": event.Repository,
		"OCTAP_WORKFLOW":   event.Workflow,
		"OCTAP_RUN_ID":     fmt.Sprintf("%d", event.RunID),
		"OCTAP_RUN_URL":    event.URL,
	}

	for key, value := range octapEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// executeCommand executes the command with timeout
func (c *commandAction) executeCommand(ctx context.Context, cmdAction *model.CommandAction, env []string, timeout time.Duration) error {
	logger := ctxlog.From(ctx)

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Expand environment variables in command and args
	command := expandPath(cmdAction.Command)
	args := make([]string, len(cmdAction.Args))
	for i, arg := range cmdAction.Args {
		args[i] = os.ExpandEnv(arg)
	}

	// Create command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// On Windows, use cmd.exe or PowerShell
		if strings.HasSuffix(strings.ToLower(command), ".ps1") {
			// PowerShell script
			psArgs := append([]string{"-ExecutionPolicy", "Bypass", "-File", command}, args...)
			cmd = exec.CommandContext(cmdCtx, "powershell", psArgs...) // #nosec G204 - command is from config file
		} else {
			// Regular command or batch file
			cmd = exec.CommandContext(cmdCtx, command, args...) // #nosec G204 - command is from config file
		}
	} else {
		// Unix-like systems
		cmd = exec.CommandContext(cmdCtx, command, args...) // #nosec G204 - command is from config file
	}

	// Set environment
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log command execution
	logger.Debug("Executing command",
		slog.String("command", command),
		slog.Any("args", args),
		slog.String("os", runtime.GOOS),
		slog.Duration("timeout", timeout),
	)

	// Run command
	err := cmd.Run()

	// Log output
	if stdout.Len() > 0 {
		logger.Debug("Command stdout",
			slog.String("command", command),
			slog.String("stdout", stdout.String()),
		)
	}
	if stderr.Len() > 0 {
		logger.Debug("Command stderr",
			slog.String("command", command),
			slog.String("stderr", stderr.String()),
		)
	}

	if err != nil {
		// Check if it was a timeout
		if cmdCtx.Err() == context.DeadlineExceeded {
			return goerr.New(fmt.Sprintf("command timed out after %s", timeout))
		}
		// Include stderr in error message
		errMsg := fmt.Sprintf("command failed: %v", err)
		if stderr.Len() > 0 {
			errMsg += fmt.Sprintf(", stderr: %s", stderr.String())
		}
		return goerr.New(errMsg)
	}

	return nil
}
