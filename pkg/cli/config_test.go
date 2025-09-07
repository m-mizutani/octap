package cli_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/cli"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

func TestConfig(t *testing.T) {
	t.Run("NewConfig defaults", func(t *testing.T) {
		config := cli.NewConfig()
		gt.Equal(t, config.Interval, 15*time.Second)
		gt.Equal(t, config.Silent, false)
	})

	t.Run("ToMonitorConfig", func(t *testing.T) {
		config := &cli.Config{
			CommitSHA:  "abc123",
			Interval:   30 * time.Second,
			ConfigPath: "/path/to/config",
			Silent:     true,
		}

		repo := model.Repository{
			Owner: "owner",
			Name:  "repo",
		}

		monitorConfig := config.ToMonitorConfig(repo)
		gt.Equal(t, monitorConfig.CommitSHA, "abc123")
		gt.Equal(t, monitorConfig.Interval, 30*time.Second)
		gt.Equal(t, monitorConfig.Repo.Owner, "owner")
		gt.Equal(t, monitorConfig.Repo.Name, "repo")
		gt.Equal(t, monitorConfig.ConfigPath, "/path/to/config")
	})
}
