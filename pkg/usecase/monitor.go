package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/octap/pkg/domain"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type MonitorUseCase struct {
	github   interfaces.GitHubService
	notifier interfaces.Notifier
	display  interfaces.Display
	config   *model.MonitorConfig
}

type MonitorUseCaseOptions struct {
	GitHub   interfaces.GitHubService
	Notifier interfaces.Notifier
	Display  interfaces.Display
	Config   *model.MonitorConfig
}

func NewMonitorUseCase(opts MonitorUseCaseOptions) *MonitorUseCase {
	return &MonitorUseCase{
		github:   opts.GitHub,
		notifier: opts.Notifier,
		display:  opts.Display,
		config:   opts.Config,
	}
}

func (u *MonitorUseCase) Execute(ctx context.Context) error {
	logger := ctxlog.From(ctx)
	startTime := time.Now()
	knownRuns := make(map[int64]*model.WorkflowRun)
	completedRuns := make(map[int64]bool)
	var lastUpdate time.Time

	logger.Debug("starting monitor",
		slog.String("repo", u.config.Repo.FullName()),
		slog.String("commit", u.config.CommitSHA),
		slog.Duration("interval", u.config.Interval),
	)

	// Create main ticker for polling interval
	pollTicker := time.NewTicker(u.config.Interval)
	defer pollTicker.Stop()

	// Create countdown ticker for UI updates (10 updates per second)
	countdownTicker := time.NewTicker(100 * time.Millisecond)
	defer countdownTicker.Stop()

	// Channel to trigger immediate check
	checkNow := make(chan struct{}, 1)
	
	// Trigger initial check
	checkNow <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-checkNow:
			// Immediate check (used for initial check)
			if err := u.performCheck(ctx, &knownRuns, &completedRuns, &lastUpdate, startTime, true); err != nil {
				if err == errAllCompleted {
					return nil
				}
				return err
			}

		case <-pollTicker.C:
			// Regular interval check
			if err := u.performCheck(ctx, &knownRuns, &completedRuns, &lastUpdate, startTime, false); err != nil {
				if err == errAllCompleted {
					return nil
				}
				return err
			}

		case <-countdownTicker.C:
			// Update countdown display
			remaining := u.config.Interval - time.Since(lastUpdate)
			if remaining > 0 {
				if extDisplay, ok := u.display.(interfaces.ExtendedDisplay); ok {
					extDisplay.ShowCountdown(remaining)
				}
			}
		}
	}
}

// Sentinel error to signal successful completion
var errAllCompleted = errors.New("all workflows completed")

func (u *MonitorUseCase) performCheck(
	ctx context.Context,
	knownRuns *map[int64]*model.WorkflowRun,
	completedRuns *map[int64]bool,
	lastUpdate *time.Time,
	startTime time.Time,
	isInitial bool,
) error {
	logger := ctxlog.From(ctx)
	
	runs, err := u.github.GetWorkflowRuns(ctx, u.config.Repo, u.config.CommitSHA)
	if err != nil {
		logger.Error("failed to get workflow runs",
			slog.String("error", err.Error()),
		)
		if domain.ErrAuthentication.Is(err) {
			return err
		}
		// Don't update lastUpdate on error
		return nil
	}

	*lastUpdate = time.Now()

	// Collect newly completed workflows for notifications
	var newlyCompleted []*model.WorkflowRun
	allCompleted := true
	hasNewCompletions := false

	for _, run := range runs {
		previous, exists := (*knownRuns)[run.ID]
		(*knownRuns)[run.ID] = run

		if run.Status != model.WorkflowStatusCompleted {
			allCompleted = false
			continue
		}

		if !(*completedRuns)[run.ID] {
			(*completedRuns)[run.ID] = true

			// Check if this is a new completion (status change)
			if exists && previous.Status != model.WorkflowStatusCompleted {
				hasNewCompletions = true
				newlyCompleted = append(newlyCompleted, run)
			}
		}
	}

	// Update display
	if u.display != nil {
		u.display.Update(runs, *lastUpdate, u.config.Interval)
	}

	// Handle sound notifications in background goroutines (non-blocking)
	for _, workflow := range newlyCompleted {
		go u.handleWorkflowNotification(ctx, workflow)
	}

	// Exit when all workflows are completed
	if allCompleted && len(runs) > 0 {
		if isInitial || hasNewCompletions {
			// Show final summary if display supports it
			if extDisplay, ok := u.display.(interfaces.ExtendedDisplay); ok {
				extDisplay.ShowFinalSummary()
			}

			summary := u.buildSummary(runs, startTime)
			if err := u.notifier.NotifyComplete(ctx, summary); err != nil {
				logger.Warn("failed to notify completion",
					slog.String("error", err.Error()),
				)
			}
			return errAllCompleted
		}
	}

	// Show waiting message if no runs found
	if len(runs) == 0 && !isInitial {
		if u.display != nil {
			u.display.ShowWaiting(u.config.CommitSHA, u.config.Repo.FullName())
		}
	}

	return nil
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

func (u *MonitorUseCase) handleWorkflowNotification(ctx context.Context, workflow *model.WorkflowRun) {
	logger := ctxlog.From(ctx)
	switch workflow.Conclusion {
	case model.WorkflowConclusionSuccess:
		if err := u.notifier.NotifySuccess(ctx, workflow); err != nil {
			logger.Warn("failed to notify success",
				slog.String("error", err.Error()),
			)
		}
	case model.WorkflowConclusionFailure:
		if err := u.notifier.NotifyFailure(ctx, workflow); err != nil {
			logger.Warn("failed to notify failure",
				slog.String("error", err.Error()),
			)
		}
	}
}