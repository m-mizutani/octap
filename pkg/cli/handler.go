package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/m-mizutani/octap/pkg/domain"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/usecase"
	"github.com/urfave/cli/v3"
)

func RunMonitor(ctx context.Context, cmd *cli.Command) error {
	logLevel := slog.LevelWarn
	if cmd.Bool("debug") {
		logLevel = slog.LevelDebug
	} else if cmd.Bool("verbose") {
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	authService := usecase.NewAuthService(logger)
	githubService := usecase.NewGitHubService(authService, logger)

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
			return fmt.Errorf("failed to get current commit: %w", err)
		}
	}

	if len(commitSHA) < 7 {
		return fmt.Errorf("invalid commit SHA: %s", commitSHA)
	}

	config := &Config{
		CommitSHA:  commitSHA,
		Interval:   cmd.Duration("interval"),
		ConfigPath: cmd.String("config"),
		Silent:     cmd.Bool("silent"),
	}

	var notifier interfaces.Notifier
	if config.Silent {
		notifier = usecase.NewNoOpNotifier()
	} else {
		notifier = usecase.NewSoundNotifier(logger)
	}

	display := NewDisplayManager(repo.FullName(), commitSHA)

	monitor := usecase.NewMonitorUseCase(usecase.MonitorUseCaseOptions{
		GitHub:   githubService,
		Notifier: notifier,
		Display:  display,
		Config:   config.ToMonitorConfig(*repo),
		Logger:   logger,
	})

	// Display initial status instead of logging
	fmt.Printf("\n🚀 Starting octap monitor\n")
	fmt.Printf("Repository: %s\n", repo.FullName())
	fmt.Printf("Commit: %s\n", commitSHA[:8])
	fmt.Printf("Interval: %s\n\n", config.Interval)

	return monitor.Execute(ctx)
}
