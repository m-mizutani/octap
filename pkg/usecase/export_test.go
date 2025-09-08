package usecase

// Export for testing
var ParseGitHubURL = parseGitHubURL

// ConfigService exports for testing
type ConfigService = configService

// Export configService methods for testing
func (c *configService) FindConfigInDirectory(dir string) string {
	return c.findConfigInDirectory(dir)
}
