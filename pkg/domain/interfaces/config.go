package interfaces

import (
	"github.com/m-mizutani/octap/pkg/domain/model"
)

// ConfigService handles configuration file operations
type ConfigService interface {
	Load(path string) (*model.Config, error)
	LoadDefault() (*model.Config, error)
	LoadFromDirectory(dir string) (*model.Config, error)
	GetDefaultPath() string
	GenerateTemplate() string
	SaveTemplate(path string, force bool) error
}
