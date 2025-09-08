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

	t.Run("LoadFromDirectory with no config file found", func(t *testing.T) {
		tempDir := t.TempDir()
		configService := usecase.NewConfigService()

		config, path, err := configService.LoadFromDirectory(tempDir)
		gt.NoError(t, err)
		gt.V(t, config).NotNil()
		gt.Equal(t, path, "")
		gt.Equal(t, len(config.Hooks.CheckSuccess), 0)
	})

	t.Run("LoadFromDirectory with octap.yml found and loaded", func(t *testing.T) {
		tempDir := t.TempDir()
		configService := usecase.NewConfigService()

		configPath := filepath.Join(tempDir, ".octap.yml")
		configContent := `hooks:
  check_success:
    - type: sound
      path: /test/success.wav
  check_failure:
    - type: sound
      path: /test/failure.wav
`
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		gt.NoError(t, err)

		config, loadedPath, err := configService.LoadFromDirectory(tempDir)
		gt.NoError(t, err)
		gt.V(t, config).NotNil()
		gt.Equal(t, loadedPath, configPath)
		gt.Equal(t, len(config.Hooks.CheckSuccess), 1)
		gt.Equal(t, len(config.Hooks.CheckFailure), 1)
		gt.Equal(t, config.Hooks.CheckSuccess[0].Type, "sound")
	})

	t.Run("LoadFromDirectory with octap.yaml found and loaded", func(t *testing.T) {
		tempDir := t.TempDir()
		configService := usecase.NewConfigService()

		configPath := filepath.Join(tempDir, ".octap.yaml")
		configContent := `hooks:
  complete_success:
    - type: sound
      path: /test/complete.wav
`
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		gt.NoError(t, err)

		config, loadedPath, err := configService.LoadFromDirectory(tempDir)
		gt.NoError(t, err)
		gt.V(t, config).NotNil()
		gt.Equal(t, loadedPath, configPath)
		gt.Equal(t, len(config.Hooks.CompleteSuccess), 1)
		gt.Equal(t, config.Hooks.CompleteSuccess[0].Type, "sound")
	})

	t.Run("LoadFromDirectory yml has priority over yaml", func(t *testing.T) {
		tempDir := t.TempDir()
		configService := usecase.NewConfigService()

		// Create both files
		ymlPath := filepath.Join(tempDir, ".octap.yml")
		yamlPath := filepath.Join(tempDir, ".octap.yaml")

		ymlContent := `hooks:
  check_success:
    - type: sound
      path: /test/yml.wav
`
		yamlContent := `hooks:
  check_success:
    - type: sound
      path: /test/yaml.wav
`

		err := os.WriteFile(ymlPath, []byte(ymlContent), 0600)
		gt.NoError(t, err)
		err = os.WriteFile(yamlPath, []byte(yamlContent), 0600)
		gt.NoError(t, err)

		config, loadedPath, err := configService.LoadFromDirectory(tempDir)
		gt.NoError(t, err)
		gt.V(t, config).NotNil()
		gt.Equal(t, loadedPath, ymlPath) // Should load .octap.yml (priority)
		gt.Equal(t, len(config.Hooks.CheckSuccess), 1)

		// Verify content from .yml file
		soundPath, ok := config.Hooks.CheckSuccess[0].Data["path"].(string)
		gt.True(t, ok)
		gt.Equal(t, soundPath, "/test/yml.wav")
	})

	t.Run("LoadFromDirectory with invalid yaml content", func(t *testing.T) {
		tempDir := t.TempDir()
		configService := usecase.NewConfigService()

		configPath := filepath.Join(tempDir, ".octap.yml")
		invalidContent := `hooks:
  check_success:
    - type sound  # invalid yaml syntax
      path: /test/invalid.wav
`
		err := os.WriteFile(configPath, []byte(invalidContent), 0600)
		gt.NoError(t, err)

		_, loadedPath, err := configService.LoadFromDirectory(tempDir)
		gt.Error(t, err)
		gt.Equal(t, loadedPath, configPath) // Path should still be returned even on error
	})
}
