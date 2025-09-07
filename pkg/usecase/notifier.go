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

type SoundNotifier struct{}

func NewSoundNotifier() interfaces.Notifier {
	return &SoundNotifier{}
}

func (n *SoundNotifier) NotifySuccess(ctx context.Context, workflow *model.WorkflowRun) error {
	logger := ctxlog.From(ctx)
	logger.Debug("workflow succeeded",
		slog.String("name", workflow.Name),
		slog.Int64("id", workflow.ID),
	)
	return n.playSystemSound(ctx, true)
}

func (n *SoundNotifier) NotifyFailure(ctx context.Context, workflow *model.WorkflowRun) error {
	logger := ctxlog.From(ctx)
	logger.Debug("workflow failed",
		slog.String("name", workflow.Name),
		slog.Int64("id", workflow.ID),
	)
	return n.playSystemSound(ctx, false)
}

func (n *SoundNotifier) NotifyComplete(ctx context.Context, summary *model.Summary) error {
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
		cmd := exec.Command("afplay", soundFile)
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

func (n *NoOpNotifier) NotifyComplete(ctx context.Context, summary *model.Summary) error {
	return nil
}
