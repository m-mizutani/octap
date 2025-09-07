package interfaces

import (
	"time"

	"github.com/m-mizutani/octap/pkg/domain/model"
)

type Display interface {
	Update(runs []*model.WorkflowRun, lastUpdate time.Time, interval time.Duration)
	ShowWaiting(commitSHA, repoName string)
	Clear()
}
