package interfaces

import (
	"github.com/m-mizutani/octap/pkg/domain/model"
)

// ConfigService handles configuration file operations
type ConfigService interface {
	Load(path string) (*model.Config, error)
	LoadDefault() (*model.Config, error)
	GenerateTemplate() string
	SaveTemplate(path string, force bool) error
}
