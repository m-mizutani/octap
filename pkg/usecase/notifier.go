package usecase

import (
	"context"
	"log/slog"
	"os/exec"
	"runtime"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type SoundNotifier struct {
	config       *model.Config
	hookExecutor interfaces.HookExecutor
}

func NewSoundNotifier() interfaces.Notifier {
	return &SoundNotifier{}
}

func (n *SoundNotifier) NotifySuccess(ctx context.Context, workflow *model.WorkflowRun) error {
	logger := ctxlog.From(ctx)
	logger.Debug("workflow succeeded",
		slog.String("name", workflow.Name),
		slog.Int64("id", workflow.ID),
	)

	// Execute hooks if configured
	if n.hookExecutor != nil {
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: workflow.Repository,
			Workflow:   workflow.Name,
			RunID:      workflow.ID,
			URL:        workflow.URL,
		}
		if err := n.hookExecutor.Execute(ctx, event); err != nil {
			logger.Warn("failed to execute hooks",
				slog.String("error", err.Error()),
			)
		}
		return nil
	}

	// Fallback to default sound
	return n.playSystemSound(ctx, true)
}

func (n *SoundNotifier) NotifyFailure(ctx context.Context, workflow *model.WorkflowRun) error {
	logger := ctxlog.From(ctx)
	logger.Debug("workflow failed",
		slog.String("name", workflow.Name),
		slog.Int64("id", workflow.ID),
	)

	// Execute hooks if configured
	if n.hookExecutor != nil {
		event := model.WorkflowEvent{
			Type:       model.HookCheckFailure,
			Repository: workflow.Repository,
			Workflow:   workflow.Name,
			RunID:      workflow.ID,
			URL:        workflow.URL,
		}
		if err := n.hookExecutor.Execute(ctx, event); err != nil {
			logger.Warn("failed to execute hooks",
				slog.String("error", err.Error()),
			)
		}
		return nil
	}

	// Fallback to default sound
	return n.playSystemSound(ctx, false)
}

func (n *SoundNotifier) NotifyComplete(ctx context.Context, summary *model.Summary) error {
	logger := ctxlog.From(ctx)
	logger.Debug("NotifyComplete called",
		slog.Int("total_runs", summary.TotalRuns),
		slog.Int("success_count", summary.SuccessCount),
		slog.Int("failure_count", summary.FailureCount),
		slog.Bool("has_hook_executor", n.hookExecutor != nil),
	)

	// Execute hooks if configured
	if n.hookExecutor != nil {
		var eventType model.HookEvent
		if summary.FailureCount > 0 {
			eventType = model.HookCompleteFailure
			logger.Debug("Using HookCompleteFailure event type")
		} else {
			eventType = model.HookCompleteSuccess
			logger.Debug("Using HookCompleteSuccess event type")
		}

		event := model.WorkflowEvent{
			Type: eventType,
		}
		logger.Debug("Calling hookExecutor.Execute",
			slog.String("event_type", string(eventType)),
		)
		if err := n.hookExecutor.Execute(ctx, event); err != nil {
			logger.Warn("failed to execute hooks",
				slog.String("error", err.Error()),
			)
		} else {
			logger.Debug("hookExecutor.Execute completed")
		}
		return nil
	}

	logger.Debug("No hookExecutor, using fallback default sound")
	// Fallback to default sound
	if summary.FailureCount > 0 {
		return n.playSystemSound(ctx, false)
	}
	return n.playSystemSound(ctx, true)
}

func (n *SoundNotifier) playSystemSound(ctx context.Context, success bool) error {
	logger := ctxlog.From(ctx)

	switch runtime.GOOS {
	case "darwin":
		var soundFile string
		if success {
			soundFile = "/System/Library/Sounds/Glass.aiff"
		} else {
			soundFile = "/System/Library/Sounds/Basso.aiff"
		}
		cmd := exec.Command("afplay", soundFile) // #nosec G204 - soundFile is hardcoded
		if err := cmd.Run(); err != nil {
			logger.Warn("failed to play sound",
				slog.String("error", err.Error()),
			)
		}

	case "linux":
		soundFile := "complete"
		if !success {
			soundFile = "dialog-error"
		}
		cmd := exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/"+soundFile+".oga")
		if err := cmd.Run(); err != nil {
			// paplay failed, try aplay
			cmd = exec.Command("aplay", "/usr/share/sounds/alsa/Front_Center.wav")
			if err := cmd.Run(); err != nil {
				logger.Warn("failed to play sound with both paplay and aplay",
					slog.String("error", err.Error()),
				)
			}
		}

	case "windows":
		// Windows uses PowerShell to play sounds
		var soundFile string
		if success {
			soundFile = "C:\\Windows\\Media\\chimes.wav"
		} else {
			soundFile = "C:\\Windows\\Media\\chord.wav"
		}
		script := `(New-Object Media.SoundPlayer "` + soundFile + `").PlaySync()`
		cmd := exec.Command("powershell", "-Command", script) // #nosec G204 - soundFile is hardcoded
		if err := cmd.Run(); err != nil {
			logger.Warn("failed to play sound on Windows",
				slog.String("error", err.Error()),
			)
		}

	default:
		logger.Warn("sound notification not supported on this platform",
			slog.String("os", runtime.GOOS),
		)
	}

	return nil
}

type NoOpNotifier struct{}

func NewNoOpNotifier() interfaces.Notifier {
	return &NoOpNotifier{}
}

func (n *NoOpNotifier) NotifySuccess(ctx context.Context, workflow *model.WorkflowRun) error {
	return nil
}

func (n *NoOpNotifier) NotifyFailure(ctx context.Context, workflow *model.WorkflowRun) error {
	return nil
}

func (n *SoundNotifier) SetConfig(config *model.Config) {
	logger := ctxlog.From(context.Background())
	if config != nil {
		n.config = config
		n.hookExecutor = NewHookExecutor(config)
		logger.Debug("SoundNotifier.SetConfig: hookExecutor created",
			slog.Bool("has_config", config != nil),
			slog.Int("check_success_count", len(config.Hooks.CheckSuccess)),
			slog.Int("check_failure_count", len(config.Hooks.CheckFailure)),
			slog.Int("complete_success_count", len(config.Hooks.CompleteSuccess)),
			slog.Int("complete_failure_count", len(config.Hooks.CompleteFailure)),
		)
	} else {
		logger.Debug("SoundNotifier.SetConfig: config is nil")
	}
}

func (n *NoOpNotifier) NotifyComplete(ctx context.Context, summary *model.Summary) error {
	return nil
}

func (n *NoOpNotifier) SetConfig(config *model.Config) {
	// NoOp
}

func (n *NoOpNotifier) WaitForPendingActions() {
	// NoOp - no actions to wait for
}

// WaitForPendingActions waits for all pending hook actions to complete.
func (n *SoundNotifier) WaitForPendingActions() {
	if n.hookExecutor != nil {
		n.hookExecutor.WaitForCompletion()
	}
}
