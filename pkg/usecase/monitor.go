package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/m-mizutani/octap/pkg/domain"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type MonitorUseCase struct {
	github   interfaces.GitHubService
	notifier interfaces.Notifier
	display  interfaces.Display
	config   *model.MonitorConfig
	logger   *slog.Logger
}

type MonitorUseCaseOptions struct {
	GitHub   interfaces.GitHubService
	Notifier interfaces.Notifier
	Display  interfaces.Display
	Config   *model.MonitorConfig
	Logger   *slog.Logger
}

func NewMonitorUseCase(opts MonitorUseCaseOptions) *MonitorUseCase {
	return &MonitorUseCase{
		github:   opts.GitHub,
		notifier: opts.Notifier,
		display:  opts.Display,
		config:   opts.Config,
		logger:   opts.Logger,
	}
}

func (u *MonitorUseCase) Execute(ctx context.Context) error {
	startTime := time.Now()
	knownRuns := make(map[int64]*model.WorkflowRun)
	completedRuns := make(map[int64]bool)
	var lastUpdate time.Time
	var currentRuns []*model.WorkflowRun

	u.logger.Debug("starting monitor",
		slog.String("repo", u.config.Repo.FullName()),
		slog.String("commit", u.config.CommitSHA),
		slog.Duration("interval", u.config.Interval),
	)

	ticker := time.NewTicker(u.config.Interval)
	defer ticker.Stop()

	// Add 1-second refresh ticker for display updates
	refreshTicker := time.NewTicker(1 * time.Second)
	defer refreshTicker.Stop()

	// Perform initial check immediately
	initialCheck := true

	for {
		// Check if this is the initial run
		isInitial := initialCheck
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-refreshTicker.C:
			// Check for cancellation during refresh
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			// Refresh display with current data every second
			if u.display != nil && len(currentRuns) > 0 {
				u.display.Update(currentRuns, lastUpdate, u.config.Interval)
			}
			continue
		case <-ticker.C:
			// Main polling logic after initial check
		default:
			// Skip delay on initial run
			if !initialCheck {
				continue
			}
		}

		
		runs, err := u.github.GetWorkflowRuns(ctx, u.config.Repo, u.config.CommitSHA)
		if err != nil {
			u.logger.Error("failed to get workflow runs",
				slog.String("error", err.Error()),
			)
			if domain.ErrAuthentication.Is(err) {
				return err
			}
			// Don't wait on initial check error
			if !isInitial {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
			continue
		}

		lastUpdate = time.Now()
		currentRuns = runs
		

		allCompleted := true
		hasNewCompletions := false
		
		for _, run := range runs {
			previous, exists := knownRuns[run.ID]
			knownRuns[run.ID] = run

			if run.Status != model.WorkflowStatusCompleted {
				allCompleted = false
				continue
			}

			if !completedRuns[run.ID] {
				completedRuns[run.ID] = true

				// Only notify on status change, not on first discovery of already completed runs
				if exists && previous.Status != model.WorkflowStatusCompleted {
					hasNewCompletions = true
					switch run.Conclusion {
					case model.WorkflowConclusionSuccess:
						if err := u.notifier.NotifySuccess(ctx, run); err != nil {
							u.logger.Warn("failed to notify success",
								slog.String("error", err.Error()),
							)
						}
					case model.WorkflowConclusionFailure:
						if err := u.notifier.NotifyFailure(ctx, run); err != nil {
							u.logger.Warn("failed to notify failure",
								slog.String("error", err.Error()),
							)
						}
					}
				}
			}
		}

		if u.display != nil {
			u.display.Update(runs, lastUpdate, u.config.Interval)
		}

		// Exit when all workflows are completed
		// For initial check: exit immediately if all are already completed
		// For subsequent checks: only exit if there were new completions
		if allCompleted && len(runs) > 0 {
			if isInitial || hasNewCompletions {
				summary := u.buildSummary(runs, startTime)
				if err := u.notifier.NotifyComplete(ctx, summary); err != nil {
					u.logger.Warn("failed to notify completion",
						slog.String("error", err.Error()),
					)
				}
				return nil
			}
		}

		// Only show waiting message after first check if no runs found
		if len(runs) == 0 && !isInitial {
			if u.display != nil {
				u.display.ShowWaiting(u.config.CommitSHA, u.config.Repo.FullName())
			}
		}

		// Mark initial check as done
		if isInitial {
			initialCheck = false
		}

		// Wait for next interval (skip on initial run)
		if !isInitial {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	}
}

func (u *MonitorUseCase) buildSummary(runs []*model.WorkflowRun, startTime time.Time) *model.Summary {
	summary := &model.Summary{
		TotalRuns: len(runs),
		Duration:  time.Since(startTime).Round(time.Second),
	}

	for _, run := range runs {
		switch run.Conclusion {
		case model.WorkflowConclusionSuccess:
			summary.SuccessCount++
		case model.WorkflowConclusionFailure:
			summary.FailureCount++
		default:
			summary.OtherCount++
		}
	}

	return summary
}
