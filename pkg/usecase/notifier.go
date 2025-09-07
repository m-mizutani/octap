package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"

	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type SoundNotifier struct {
	logger *slog.Logger
}

func NewSoundNotifier(logger *slog.Logger) interfaces.Notifier {
	return &SoundNotifier{
		logger: logger,
	}
}

func (n *SoundNotifier) NotifySuccess(ctx context.Context, workflow *model.WorkflowRun) error {
	n.logger.Debug("workflow succeeded",
		slog.String("name", workflow.Name),
		slog.Int64("id", workflow.ID),
	)
	fmt.Printf("✅ %s completed successfully\n", workflow.Name)
	return n.playSystemSound(true)
}

func (n *SoundNotifier) NotifyFailure(ctx context.Context, workflow *model.WorkflowRun) error {
	n.logger.Debug("workflow failed",
		slog.String("name", workflow.Name),
		slog.Int64("id", workflow.ID),
	)
	fmt.Printf("❌ %s failed\n", workflow.Name)
	return n.playSystemSound(false)
}

func (n *SoundNotifier) NotifyComplete(ctx context.Context, summary *model.Summary) error {
	fmt.Printf("\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("🎉 All workflows completed!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Total runs: %d\n", summary.TotalRuns)
	fmt.Printf("✅ Success: %d\n", summary.SuccessCount)
	if summary.FailureCount > 0 {
		fmt.Printf("❌ Failed: %d\n", summary.FailureCount)
	}
	if summary.OtherCount > 0 {
		fmt.Printf("⚠️  Other: %d\n", summary.OtherCount)
	}
	fmt.Printf("Duration: %s\n", summary.Duration)
	fmt.Printf("\n")

	if summary.FailureCount > 0 {
		return n.playSystemSound(false)
	}
	return n.playSystemSound(true)
}

func (n *SoundNotifier) playSystemSound(success bool) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		if success {
			cmd = exec.Command("afplay", "/System/Library/Sounds/Glass.aiff")
		} else {
			cmd = exec.Command("afplay", "/System/Library/Sounds/Basso.aiff")
		}
	case "linux":
		soundFile := "complete"
		if !success {
			soundFile = "dialog-error"
		}
		cmd = exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/"+soundFile+".oga")
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("aplay", "/usr/share/sounds/alsa/Front_Center.wav")
		}
	case "windows":
		return nil
	default:
		n.logger.Warn("sound notification not supported on this platform",
			slog.String("os", runtime.GOOS),
		)
		return nil
	}

	if cmd != nil {
		if err := cmd.Run(); err != nil {
			n.logger.Warn("failed to play sound",
				slog.String("error", err.Error()),
			)
		}
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
