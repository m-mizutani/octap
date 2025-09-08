package model

import (
	"fmt"
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
	Type string                 `yaml:"type"` // "sound", "slack", "command"
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

// ToSlackAction converts Action to SlackAction for type safety
func (a *Action) ToSlackAction() (*SlackAction, error) {
	if a.Type != "slack" {
		return nil, goerr.New("action is not a slack type")
	}

	webhookURL, ok := a.Data["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, goerr.New("slack action requires 'webhook_url' field")
	}

	message, ok := a.Data["message"].(string)
	if !ok || message == "" {
		return nil, goerr.New("slack action requires 'message' field")
	}

	slackAction := &SlackAction{
		WebhookURL: webhookURL,
		Message:    message,
	}

	// Optional fields
	if color, ok := a.Data["color"].(string); ok {
		slackAction.Color = color
	}
	if iconEmoji, ok := a.Data["icon_emoji"].(string); ok {
		slackAction.IconEmoji = iconEmoji
	}
	if userName, ok := a.Data["username"].(string); ok {
		slackAction.UserName = userName
	}

	return slackAction, nil
}

// parseStringSlice is a helper function to parse string arrays from interface{}
func parseStringSlice(value interface{}, fieldName string) ([]string, error) {
	switch v := value.(type) {
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			itemStr, ok := item.(string)
			if !ok {
				return nil, goerr.New(fmt.Sprintf("command action '%s' must be string array", fieldName))
			}
			result[i] = itemStr
		}
		return result, nil
	case []string:
		return v, nil
	default:
		return nil, goerr.New(fmt.Sprintf("command action '%s' must be an array", fieldName))
	}
}

// ToCommandAction converts Action to CommandAction for type safety
func (a *Action) ToCommandAction() (*CommandAction, error) {
	if a.Type != "command" {
		return nil, goerr.New("action is not a command type")
	}

	command, ok := a.Data["command"].(string)
	if !ok || command == "" {
		return nil, goerr.New("command action requires 'command' field")
	}

	cmdAction := &CommandAction{
		Command: command,
	}

	// Optional args field
	if argsValue, ok := a.Data["args"]; ok {
		args, err := parseStringSlice(argsValue, "args")
		if err != nil {
			return nil, err
		}
		cmdAction.Args = args
	}

	// Optional timeout field
	if timeoutValue, ok := a.Data["timeout"]; ok {
		switch v := timeoutValue.(type) {
		case string:
			timeout, err := time.ParseDuration(v)
			if err != nil {
				return nil, goerr.Wrap(err, "invalid timeout format")
			}
			cmdAction.Timeout = timeout
		case time.Duration:
			cmdAction.Timeout = v
		default:
			return nil, goerr.New("command action 'timeout' must be a duration string")
		}
	}

	// Optional env field
	if envValue, ok := a.Data["env"]; ok {
		env, err := parseStringSlice(envValue, "env")
		if err != nil {
			return nil, err
		}
		cmdAction.Env = env
	}

	return cmdAction, nil
}
