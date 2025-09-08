package model

import (
	"time"

	"github.com/m-mizutani/goerr/v2"
)

type MonitorConfig struct {
	CommitSHA string
	Interval  time.Duration
	Repo      Repository
}

// Config represents the application configuration
type Config struct {
	Hooks HooksConfig `yaml:"hooks"`
}

// HooksConfig defines hooks for workflow events
type HooksConfig struct {
	CheckSuccess    []Action `yaml:"check_success,omitempty"`
	CheckFailure    []Action `yaml:"check_failure,omitempty"`
	CompleteSuccess []Action `yaml:"complete_success,omitempty"`
	CompleteFailure []Action `yaml:"complete_failure,omitempty"`
}

// Action represents an action to be executed
type Action struct {
	Type string                 `yaml:"type"` // "sound"
	Data map[string]interface{} `yaml:",inline"`
}

// ToSoundAction converts Action to SoundAction for type safety
func (a *Action) ToSoundAction() (*SoundAction, error) {
	if a.Type != "sound" {
		return nil, goerr.New("action is not a sound type")
	}

	pathValue, ok := a.Data["path"]
	if !ok {
		return nil, goerr.New("sound action requires 'path' field")
	}

	path, ok := pathValue.(string)
	if !ok {
		return nil, goerr.New("sound action 'path' must be a string")
	}

	return &SoundAction{
		Path: path,
	}, nil
}

// SoundAction represents a sound playing action
type SoundAction struct {
	Path string `yaml:"path"`
}
