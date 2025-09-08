package usecase

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octap/pkg/domain"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type GitHubService struct {
	authService interfaces.AuthService
}

func NewGitHubService(authService interfaces.AuthService) interfaces.GitHubService {
	return &GitHubService{
		authService: authService,
	}
}

func (s *GitHubService) GetCurrentCommit(ctx context.Context, repoPath string) (string, error) {
	logger := ctxlog.From(ctx)
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", domain.ErrRepository.Wrap(err)
	}
	commitSHA := strings.TrimSpace(string(output))

	// Check if commit is pushed to remote
	checkCmd := exec.CommandContext(ctx, "git", "branch", "-r", "--contains", commitSHA) // #nosec G204 - commitSHA is validated and safe
	checkCmd.Dir = repoPath
	remoteOutput, err := checkCmd.Output()
	if err != nil || len(remoteOutput) == 0 {
		logger.Warn("Commit not found in remote branches",
			slog.String("sha", commitSHA[:8]),
		)
		return "", goerr.Wrap(domain.ErrNotPushed, "commit not found in remote", goerr.V("commit", commitSHA[:8]))
	}

	return commitSHA, nil
}

func (s *GitHubService) GetRepositoryInfo(ctx context.Context, repoPath string) (*model.Repository, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, domain.ErrRepository.Wrap(err)
	}

	remoteURL := strings.TrimSpace(string(output))
	owner, name := parseGitHubURL(remoteURL)
	if owner == "" || name == "" {
		return nil, domain.ErrRepository.Wrap(goerr.New("failed to parse GitHub URL: " + remoteURL))
	}

	return &model.Repository{
		Owner: owner,
		Name:  name,
	}, nil
}

func parseGitHubURL(url string) (owner, repo string) {
	url = strings.TrimSuffix(url, ".git")

	var path string
	switch {
	case strings.HasPrefix(url, "git@github.com:"):
		path = strings.TrimPrefix(url, "git@github.com:")
	case strings.HasPrefix(url, "https://github.com/"):
		path = strings.TrimPrefix(url, "https://github.com/")
	case strings.HasPrefix(url, "ssh://git@github.com/"):
		path = strings.TrimPrefix(url, "ssh://git@github.com/")
	default:
		return "", ""
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "", ""
}

func (s *GitHubService) GetWorkflowRuns(ctx context.Context, repo model.Repository, commitSHA string) ([]*model.WorkflowRun, error) {
	logger := ctxlog.From(ctx)

	client, err := s.authService.GetAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}

	opts := &github.ListWorkflowRunsOptions{
		HeadSHA: commitSHA,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	runs, _, err := client.Actions.ListRepositoryWorkflowRuns(ctx, repo.Owner, repo.Name, opts)
	if err != nil {
		return nil, domain.ErrAPIRequest.Wrap(err)
	}

	var workflowRuns []*model.WorkflowRun
	for _, run := range runs.WorkflowRuns {
		workflowRun := &model.WorkflowRun{
			ID:        run.GetID(),
			Name:      run.GetName(),
			Status:    convertStatus(run.GetStatus()),
			URL:       run.GetHTMLURL(),
			CreatedAt: run.GetCreatedAt().Time,
			UpdatedAt: run.GetUpdatedAt().Time,
		}

		if run.GetStatus() == "completed" {
			workflowRun.Conclusion = convertConclusion(run.GetConclusion())
		}

		workflowRuns = append(workflowRuns, workflowRun)
	}

	logger.Debug("fetched workflow runs",
		slog.String("repo", repo.FullName()),
		slog.String("commit", commitSHA),
		slog.Int("count", len(workflowRuns)),
	)

	return workflowRuns, nil
}

func convertStatus(status string) model.WorkflowStatus {
	switch status {
	case "queued":
		return model.WorkflowStatusQueued
	case "in_progress":
		return model.WorkflowStatusInProgress
	case "completed":
		return model.WorkflowStatusCompleted
	default:
		return model.WorkflowStatus(status)
	}
}

func convertConclusion(conclusion string) model.WorkflowConclusion {
	switch conclusion {
	case "success":
		return model.WorkflowConclusionSuccess
	case "failure":
		return model.WorkflowConclusionFailure
	case "cancelled":
		return model.WorkflowConclusionCancelled
	case "skipped":
		return model.WorkflowConclusionSkipped
	case "timed_out":
		return model.WorkflowConclusionTimedOut
	default:
		return model.WorkflowConclusion(conclusion)
	}
}
