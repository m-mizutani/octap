package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

func TestActionConversions(t *testing.T) {
	t.Run("ToSoundAction success", func(t *testing.T) {
		action := model.Action{
			Type: "sound",
			Data: map[string]interface{}{
				"path": "/path/to/sound.mp3",
			},
		}

		soundAction, err := action.ToSoundAction()
		gt.NoError(t, err)
		gt.Equal(t, "/path/to/sound.mp3", soundAction.Path)
	})

	t.Run("ToSoundAction with wrong type", func(t *testing.T) {
		action := model.Action{
			Type: "unknown",
			Data: map[string]interface{}{
				"message": "test",
			},
		}

		_, err := action.ToSoundAction()
		gt.Error(t, err)
	})

	t.Run("ToSoundAction without path", func(t *testing.T) {
		action := model.Action{
			Type: "sound",
			Data: map[string]interface{}{},
		}

		_, err := action.ToSoundAction()
		gt.Error(t, err)
	})
}
