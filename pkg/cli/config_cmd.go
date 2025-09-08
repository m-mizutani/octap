package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-mizutani/octap/pkg/usecase"
	"github.com/urfave/cli/v3"
)

// NewConfigCommand creates a new config command
func NewConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage octap configuration",
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Generate configuration template",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output path for config file",
					},
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force overwrite existing file",
					},
				},
				Action: configInitAction,
			},
		},
	}
}

func configInitAction(ctx context.Context, cmd *cli.Command) error {
	service := usecase.NewConfigService()

	outputPath := cmd.String("output")
	if outputPath == "" {
		// Use default path
		homeDir, _ := os.UserHomeDir()
		outputPath = filepath.Join(homeDir, ".config", "octap", "config.yml")
	}

	force := cmd.Bool("force")

	if err := service.SaveTemplate(outputPath, force); err != nil {
		return fmt.Errorf("failed to create config template: %w", err)
	}

	return nil
}
