package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

func TestRepository(t *testing.T) {
	t.Run("FullName", func(t *testing.T) {
		repo := model.Repository{
			Owner: "m-mizutani",
			Name:  "octap",
		}

		gt.Equal(t, repo.FullName(), "m-mizutani/octap")
	})

	t.Run("Empty repository", func(t *testing.T) {
		repo := model.Repository{}
		gt.Equal(t, repo.FullName(), "/")
	})
}
