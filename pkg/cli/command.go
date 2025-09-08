package cli

import (
	"github.com/urfave/cli/v3"
)

func NewCommand() *cli.Command {
	flags := append(DefineFlags(),
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "Enable verbose logging",
			Value: false,
		},
	)

	return &cli.Command{
		Name:    "octap",
		Usage:   "CLI GitHub Actions notifier",
		Version: "0.1.0",
		Description: `octap monitors GitHub Actions workflows for a specific commit and notifies you when they complete.
		
By default, it monitors the current commit in the current directory.
Use -c/--commit to specify a different commit SHA.`,
		Flags:  flags,
		Action: RunMonitor,
		Commands: []*cli.Command{
			NewConfigCommand(),
		},
	}
}
