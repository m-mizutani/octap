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
	Type string                 `yaml:"type"` // "sound" or "notify"
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

// ToNotifyAction converts Action to NotifyAction for type safety
func (a *Action) ToNotifyAction() (*NotifyAction, error) {
	if a.Type != "notify" {
		return nil, goerr.New("action is not a notify type")
	}

	messageValue, ok := a.Data["message"]
	if !ok {
		return nil, goerr.New("notify action requires 'message' field")
	}

	message, ok := messageValue.(string)
	if !ok {
		return nil, goerr.New("notify action 'message' must be a string")
	}

	action := &NotifyAction{
		Message: message,
		Title:   "octap", // default
		Sound:   nil,     // use default
	}

	// Optional title
	if titleValue, ok := a.Data["title"]; ok {
		if title, ok := titleValue.(string); ok {
			action.Title = title
		}
	}

	// Optional sound
	if soundValue, ok := a.Data["sound"]; ok {
		if sound, ok := soundValue.(bool); ok {
			action.Sound = &sound
		}
	}

	return action, nil
}

// SoundAction represents a sound playing action
type SoundAction struct {
	Path string `yaml:"path"`
}

// NotifyAction represents a desktop notification action
type NotifyAction struct {
	Title   string `yaml:"title,omitempty"`
	Message string `yaml:"message"`
	Sound   *bool  `yaml:"sound,omitempty"` // pointer to distinguish unset from false
}
