package interfaces

import (
	"context"

	"github.com/m-mizutani/octap/pkg/domain/model"
)

type GitHubService interface {
	GetWorkflowRuns(ctx context.Context, repo model.Repository, commitSHA string) ([]*model.WorkflowRun, error)
	GetCurrentCommit(ctx context.Context, repoPath string) (string, error)
	GetRepositoryInfo(ctx context.Context, repoPath string) (*model.Repository, error)
}
