package usecase

import (
	"context"
	"log/slog"
	"os/exec"
	"runtime"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type soundAction struct{}

// NewSoundAction creates a new SoundAction instance
func NewSoundAction() interfaces.ActionExecutor {
	return &soundAction{}
}

// Execute plays a sound file
func (s *soundAction) Execute(ctx context.Context, action model.Action, event model.WorkflowEvent) error {
	logger := ctxlog.From(ctx)

	// Convert to typed action
	soundAction, err := action.ToSoundAction()
	if err != nil {
		return goerr.Wrap(err, "failed to parse sound action")
	}

	expandedPath := expandPath(soundAction.Path)

	// Play sound based on OS
	switch runtime.GOOS {
	case "darwin":
		return s.playMacOS(ctx, expandedPath)
	case "linux":
		return s.playLinux(ctx, expandedPath)
	case "windows":
		return s.playWindows(ctx, expandedPath)
	default:
		logger.Warn("sound playback not supported on this OS",
			slog.String("os", runtime.GOOS),
		)
		return nil
	}
}

func (s *soundAction) playMacOS(ctx context.Context, path string) error {
	logger := ctxlog.From(ctx)
	cmd := exec.Command("afplay", path) // #nosec G204 - path is from config file
	if err := cmd.Run(); err != nil {
		logger.Error("sound playback failed on macOS",
			slog.String("command", "afplay"),
			slog.String("path", path),
			slog.String("error", err.Error()),
			slog.String("os", "darwin"),
		)
		return goerr.Wrap(err, "failed to play sound on macOS")
	}
	logger.Debug("sound played successfully", slog.String("path", path))
	return nil
}

func (s *soundAction) playLinux(ctx context.Context, path string) error {
	logger := ctxlog.From(ctx)

	// Try paplay first (PulseAudio)
	cmd := exec.Command("paplay", path) // #nosec G204 - path is from config file
	if err := cmd.Run(); err == nil {
		logger.Debug("sound played successfully with paplay", slog.String("path", path))
		return nil
	} else {
		logger.Warn("paplay failed, trying aplay fallback",
			slog.String("path", path),
			slog.String("paplay_error", err.Error()),
		)
	}

	// Fallback to aplay (ALSA)
	cmd = exec.Command("aplay", path) // #nosec G204 - path is from config file
	if err := cmd.Run(); err != nil {
		logger.Error("sound playback failed on Linux with both paplay and aplay",
			slog.String("path", path),
			slog.String("aplay_error", err.Error()),
			slog.String("os", "linux"),
		)
		return goerr.Wrap(err, "failed to play sound on Linux (both paplay and aplay failed)")
	}
	logger.Debug("sound played successfully with aplay", slog.String("path", path))
	return nil
}

func (s *soundAction) playWindows(ctx context.Context, path string) error {
	logger := ctxlog.From(ctx)

	// Use PowerShell to play sound
	script := `(New-Object Media.SoundPlayer "` + path + `").PlaySync()`
	cmd := exec.Command("powershell", "-Command", script) // #nosec G204 - path is from config file
	if err := cmd.Run(); err != nil {
		logger.Error("sound playback failed on Windows",
			slog.String("command", "powershell"),
			slog.String("script", script),
			slog.String("path", path),
			slog.String("error", err.Error()),
			slog.String("os", "windows"),
		)
		return goerr.Wrap(err, "failed to play sound on Windows")
	}
	logger.Debug("sound played successfully", slog.String("path", path))
	return nil
}
