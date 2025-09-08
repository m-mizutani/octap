package model

import "time"

// CommandAction represents a command execution action
type CommandAction struct {
	Command string        `yaml:"command"`
	Args    []string      `yaml:"args,omitempty"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
	Env     []string      `yaml:"env,omitempty"` // Additional environment variables
}
