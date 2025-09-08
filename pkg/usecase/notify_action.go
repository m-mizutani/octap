package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type notifyAction struct{}

// NewNotifyAction creates a new NotifyAction instance
func NewNotifyAction() interfaces.ActionExecutor {
	return &notifyAction{}
}

// Execute sends a desktop notification
func (n *notifyAction) Execute(ctx context.Context, action model.Action, event model.WorkflowEvent) error {
	logger := ctxlog.From(ctx)

	// Convert to typed action
	notifyAction, err := action.ToNotifyAction()
	if err != nil {
		return goerr.Wrap(err, "failed to parse notify action")
	}

	// Apply template variables
	title := n.expandTemplate(notifyAction.Title, event)
	message := n.expandTemplate(notifyAction.Message, event)

	playSound := true
	if notifyAction.Sound != nil {
		playSound = *notifyAction.Sound
	}

	// Send notification based on OS
	switch runtime.GOOS {
	case "darwin":
		return n.notifyMacOS(ctx, title, message, playSound)
	case "linux":
		return n.notifyLinux(ctx, title, message)
	case "windows":
		return n.notifyWindows(ctx, title, message)
	default:
		logger.Warn("notifications not supported on this OS",
			slog.String("os", runtime.GOOS),
		)
		return nil
	}
}

func (n *notifyAction) notifyMacOS(ctx context.Context, title, message string, playSound bool) error {
	logger := ctxlog.From(ctx)

	script := fmt.Sprintf(`display notification "%s" with title "%s"`,
		escapeAppleScript(message), escapeAppleScript(title))

	if playSound {
		script += " sound name \"Glass\""
	}

	cmd := exec.Command("osascript", "-e", script) // #nosec G204 - script is built from config data
	if err := cmd.Run(); err != nil {
		logger.Error("notification failed on macOS",
			slog.String("command", "osascript"),
			slog.String("script", script),
			slog.String("title", title),
			slog.String("message", message),
			slog.Bool("playSound", playSound),
			slog.String("error", err.Error()),
			slog.String("os", "darwin"),
		)
		return goerr.Wrap(err, "failed to send notification on macOS")
	}
	logger.Debug("notification sent successfully",
		slog.String("title", title),
		slog.String("message", message),
	)
	return nil
}

func (n *notifyAction) notifyLinux(ctx context.Context, title, message string) error {
	logger := ctxlog.From(ctx)

	cmd := exec.Command("notify-send", title, message)
	if err := cmd.Run(); err != nil {
		logger.Error("notification failed on Linux",
			slog.String("command", "notify-send"),
			slog.String("title", title),
			slog.String("message", message),
			slog.String("error", err.Error()),
			slog.String("os", "linux"),
		)
		return goerr.Wrap(err, "failed to send notification on Linux (notify-send not available or failed)")
	}
	logger.Debug("notification sent successfully",
		slog.String("title", title),
		slog.String("message", message),
	)
	return nil
}

func (n *notifyAction) notifyWindows(ctx context.Context, title, message string) error {
	logger := ctxlog.From(ctx)

	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$notification = New-Object System.Windows.Forms.NotifyIcon
$notification.Icon = [System.Drawing.SystemIcons]::Information
$notification.BalloonTipIcon = 'Info'
$notification.BalloonTipTitle = '%s'
$notification.BalloonTipText = '%s'
$notification.Visible = $true
$notification.ShowBalloonTip(10000)
`, escapePS(title), escapePS(message))

	cmd := exec.Command("powershell", "-Command", script) // #nosec G204 - script is built from config data
	if err := cmd.Run(); err != nil {
		logger.Error("notification failed on Windows",
			slog.String("command", "powershell"),
			slog.String("title", title),
			slog.String("message", message),
			slog.String("error", err.Error()),
			slog.String("os", "windows"),
		)
		return goerr.Wrap(err, "failed to send notification on Windows")
	}
	logger.Debug("notification sent successfully",
		slog.String("title", title),
		slog.String("message", message),
	)
	return nil
}

// expandTemplate replaces template variables with event data
func (n *notifyAction) expandTemplate(text string, event model.WorkflowEvent) string {
	replacer := strings.NewReplacer(
		"{{.Repository}}", event.Repository,
		"{{.Workflow}}", event.Workflow,
		"{{.URL}}", event.URL,
	)
	return replacer.Replace(text)
}

// escapeAppleScript escapes special characters for AppleScript
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// escapePS escapes special characters for PowerShell
func escapePS(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	return s
}
