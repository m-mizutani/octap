package cli

import (
	"time"

	"github.com/m-mizutani/octap/pkg/domain/model"
	"github.com/urfave/cli/v3"
)

type Config struct {
	CommitSHA string
	Interval  time.Duration
	Silent    bool
}

func NewConfig() *Config {
	return &Config{
		Interval: 5 * time.Second,
	}
}

func (c *Config) ToMonitorConfig(repo model.Repository) *model.MonitorConfig {
	return &model.MonitorConfig{
		CommitSHA: c.CommitSHA,
		Interval:  c.Interval,
		Repo:      repo,
	}
}

func DefineFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "commit",
			Aliases: []string{"c"},
			Usage:   "Specify commit SHA to monitor",
		},
		&cli.DurationFlag{
			Name:    "interval",
			Aliases: []string{"i"},
			Usage:   "Polling interval",
			Value:   5 * time.Second,
		},
		&cli.BoolFlag{
			Name:  "silent",
			Usage: "Disable sound notifications",
			Value: false,
		},
		&cli.StringFlag{
			Name:    "github-oauth-client-id",
			Sources: cli.EnvVars("OCTAP_GITHUB_OAUTH_CLIENT_ID"),
			Usage:   "GitHub OAuth App Client ID (defaults to built-in ID)",
		},
		&cli.StringFlag{
			Name:  "config",
			Usage: "Path to configuration file",
		},
	}
}
