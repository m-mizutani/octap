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
	// Execute hooks if configured
	if n.hookExecutor != nil {
		var eventType model.HookEvent
		if summary.FailureCount > 0 {
			eventType = model.HookCompleteFailure
		} else {
			eventType = model.HookCompleteSuccess
		}

		event := model.WorkflowEvent{
			Type: eventType,
		}
		if err := n.hookExecutor.Execute(context.Background(), event); err != nil {
			logger := ctxlog.From(ctx)
			logger.Warn("failed to execute hooks",
				slog.String("error", err.Error()),
			)
		}
		return nil
	}

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
		// Sound not supported on Windows
		return nil

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
	if config != nil {
		n.config = config
		n.hookExecutor = NewHookExecutor(config)
	}
}

func (n *NoOpNotifier) NotifyComplete(ctx context.Context, summary *model.Summary) error {
	return nil
}

func (n *NoOpNotifier) SetConfig(config *model.Config) {
	// NoOp
}
