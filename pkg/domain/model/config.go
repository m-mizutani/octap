package model

import "time"

type MonitorConfig struct {
	CommitSHA string
	Interval  time.Duration
	Repo      Repository
}
