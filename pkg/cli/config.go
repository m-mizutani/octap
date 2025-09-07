package cli

import (
	"time"

	"github.com/m-mizutani/octap/pkg/domain/model"
	"github.com/urfave/cli/v3"
)

type Config struct {
	CommitSHA  string
	Interval   time.Duration
	ConfigPath string
	Silent     bool
}

func NewConfig() *Config {
	return &Config{
		Interval: 5 * time.Second,
	}
}

func (c *Config) ToMonitorConfig(repo model.Repository) *model.MonitorConfig {
	return &model.MonitorConfig{
		CommitSHA:  c.CommitSHA,
		Interval:   c.Interval,
		Repo:       repo,
		ConfigPath: c.ConfigPath,
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
		&cli.StringFlag{
			Name:  "config",
			Usage: "Config file path",
		},
		&cli.BoolFlag{
			Name:  "silent",
			Usage: "Disable sound notifications",
			Value: false,
		},
	}
}
