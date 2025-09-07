package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/octap/pkg/domain"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/usecase"
	"github.com/urfave/cli/v3"
)

func RunMonitor(ctx context.Context, cmd *cli.Command) error {
	// Create cancellable context for immediate shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	logLevel := slog.LevelWarn
	if cmd.Bool("debug") {
		logLevel = slog.LevelDebug
	} else if cmd.Bool("verbose") {
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Inject logger into context
	ctx = ctxlog.With(ctx, logger)

	authService := usecase.NewAuthService()
	githubService := usecase.NewGitHubService(authService)

	currentDir, err := os.Getwd()
	if err != nil {
		return domain.ErrConfiguration.Wrap(err)
	}

	repo, err := githubService.GetRepositoryInfo(ctx, currentDir)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %w\nPlease run this command in a Git repository with GitHub remote", err)
	}

	commitSHA := cmd.String("commit")
	if commitSHA == "" {
		commitSHA, err = githubService.GetCurrentCommit(ctx, currentDir)
		if err != nil {
			// More user-friendly error message
			if strings.Contains(err.Error(), "has not been pushed") {
				return fmt.Errorf("⚠️  Current commit has not been pushed to GitHub.\nPlease push your commits first: git push")
			}
			return fmt.Errorf("failed to get current commit: %w", err)
		}
		logger.Debug("Got current commit SHA",
			slog.String("sha", commitSHA),
			slog.Int("length", len(commitSHA)),
		)
	}

	if len(commitSHA) < 7 {
		return fmt.Errorf("invalid commit SHA: %s", commitSHA)
	}

	config := &Config{
		CommitSHA: commitSHA,
		Interval:  cmd.Duration("interval"),
		Silent:    cmd.Bool("silent"),
	}

	var notifier interfaces.Notifier
	if config.Silent {
		notifier = usecase.NewNoOpNotifier()
	} else {
		notifier = usecase.NewSoundNotifier()
	}

	display := NewDisplayManager(repo.FullName(), commitSHA)

	monitor := usecase.NewMonitorUseCase(usecase.MonitorUseCaseOptions{
		GitHub:   githubService,
		Notifier: notifier,
		Display:  display,
		Config:   config.ToMonitorConfig(*repo),
	})

	// Run monitor
	err = monitor.Execute(ctx)
	if err != nil && err != context.Canceled {
		return err
	}

	return nil
}
