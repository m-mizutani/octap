package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/octap/pkg/domain"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
	"github.com/m-mizutani/octap/pkg/usecase"
	"github.com/urfave/cli/v3"
)

// hasHooks checks if the HooksConfig has any configured hooks
func hasHooks(hooks model.HooksConfig) bool {
	return len(hooks.CheckSuccess) > 0 ||
		len(hooks.CheckFailure) > 0 ||
		len(hooks.CompleteSuccess) > 0 ||
		len(hooks.CompleteFailure) > 0
}

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

	// Inject logger into context
	ctx = ctxlog.With(ctx, logger)

	// Get OAuth client ID from flag/env
	clientID := cmd.String("github-oauth-client-id")
	if clientID == "" {
		logger.Info("Using default GitHub OAuth Client ID. For production use, set OCTAP_GITHUB_OAUTH_CLIENT_ID environment variable")
	}

	authService := usecase.NewAuthService(clientID)
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
			if domain.ErrNotPushed.Is(err) {
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

	// Load configuration
	configService := usecase.NewConfigService()
	configPath := cmd.String("config")

	var appConfig *model.Config
	var configErr error

	if configPath != "" {
		// Load from specified path (highest priority)
		appConfig, configErr = configService.Load(configPath)
		if configErr != nil {
			logger.Warn("Failed to load configuration file, using defaults",
				slog.String("path", configPath),
				slog.String("error", configErr.Error()),
			)
		} else {
			logger.Info("Loaded configuration file",
				slog.String("path", configPath),
			)
		}
	} else {
		// Try to load from current directory first
		appConfig, configErr = configService.LoadFromDirectory(currentDir)
		if configErr == nil && appConfig != nil && hasHooks(appConfig.Hooks) {
			// Found and loaded config from current directory
			// We need to find which config file was actually loaded for logging
			candidates := []string{
				filepath.Join(currentDir, ".octap.yml"),
				filepath.Join(currentDir, ".octap.yaml"),
			}
			for _, candidate := range candidates {
				if _, err := os.Stat(candidate); err == nil {
					logger.Info("Loaded configuration file from current directory",
						slog.String("path", candidate),
					)
					break
				}
			}
		} else {
			// No config found in current directory, try default path
			defaultPath := configService.GetDefaultPath()

			if defaultPath == "" {
				logger.Debug("Default configuration path not available (home directory could not be determined)")
			} else {
				// Check if default config file exists before attempting to load
				if _, err := os.Stat(defaultPath); err != nil {
					if os.IsNotExist(err) {
						logger.Debug("No configuration file found",
							slog.String("current_dir", currentDir),
							slog.String("default_path", defaultPath),
						)
					}
				} else {
					appConfig, configErr = configService.LoadDefault()
					if configErr == nil && appConfig != nil {
						logger.Info("Loaded default configuration file",
							slog.String("path", defaultPath),
						)
					}
				}
			}
		}
	}

	// Set config if loaded successfully
	if configErr == nil && appConfig != nil {
		notifier.SetConfig(appConfig)
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

	// Wait for all pending hook actions to complete before exiting
	notifier.WaitForPendingActions()

	return nil
}
