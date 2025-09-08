package usecase_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/usecase"
)

func TestConfigService(t *testing.T) {
	t.Run("LoadDefault returns empty config when file doesn't exist", func(t *testing.T) {
		service := usecase.NewConfigService()
		config, err := service.LoadDefault()
		gt.NoError(t, err)
		gt.NotNil(t, config)
	})

	t.Run("GenerateTemplate returns valid template", func(t *testing.T) {
		service := usecase.NewConfigService()
		template := service.GenerateTemplate()
		gt.NotEqual(t, "", template)
		gt.True(t, strings.Contains(template, "hooks:"))
		gt.True(t, strings.Contains(template, "check_success:"))
		gt.True(t, strings.Contains(template, "check_failure:"))
		gt.True(t, strings.Contains(template, "complete_success:"))
		gt.True(t, strings.Contains(template, "complete_failure:"))
		// Should contain OS-specific sound paths
		gt.True(t, strings.Contains(template, "- type: sound"))
		gt.True(t, strings.Contains(template, "path:"))
	})

	t.Run("SaveTemplate creates file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test-config.yml")

		service := usecase.NewConfigService()
		err := service.SaveTemplate(configPath, false)
		gt.NoError(t, err)

		// Check file exists
		_, err = os.Stat(configPath)
		gt.NoError(t, err)

		// Check content
		content, err := os.ReadFile(configPath)
		gt.NoError(t, err)
		gt.True(t, strings.Contains(string(content), "hooks:"))
	})

	t.Run("SaveTemplate fails without force when file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test-config.yml")

		service := usecase.NewConfigService()

		// Create file first time
		err := service.SaveTemplate(configPath, false)
		gt.NoError(t, err)

		// Try to create again without force
		err = service.SaveTemplate(configPath, false)
		gt.Error(t, err)

		// Try with force
		err = service.SaveTemplate(configPath, true)
		gt.NoError(t, err)
	})

	t.Run("Load parses valid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test-config.yml")

		yamlContent := `
hooks:
  check_success:
    - type: sound
      path: /path/to/sound.mp3
  check_failure:
    - type: notify
      message: "Test failed"
`
		err := os.WriteFile(configPath, []byte(yamlContent), 0644)
		gt.NoError(t, err)

		service := usecase.NewConfigService()
		config, err := service.Load(configPath)
		gt.NoError(t, err)
		gt.NotNil(t, config)
		gt.Equal(t, 1, len(config.Hooks.CheckSuccess))
		gt.Equal(t, "sound", config.Hooks.CheckSuccess[0].Type)
		gt.Equal(t, 1, len(config.Hooks.CheckFailure))
		gt.Equal(t, "notify", config.Hooks.CheckFailure[0].Type)
	})
}
