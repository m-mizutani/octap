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
			Type: "notify",
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

	t.Run("ToNotifyAction success", func(t *testing.T) {
		action := model.Action{
			Type: "notify",
			Data: map[string]interface{}{
				"title":   "Test Title",
				"message": "Test message",
				"sound":   false,
			},
		}

		notifyAction, err := action.ToNotifyAction()
		gt.NoError(t, err)
		gt.Equal(t, "Test Title", notifyAction.Title)
		gt.Equal(t, "Test message", notifyAction.Message)
		gt.NotNil(t, notifyAction.Sound)
		gt.False(t, *notifyAction.Sound)
	})

	t.Run("ToNotifyAction with defaults", func(t *testing.T) {
		action := model.Action{
			Type: "notify",
			Data: map[string]interface{}{
				"message": "Test message",
			},
		}

		notifyAction, err := action.ToNotifyAction()
		gt.NoError(t, err)
		gt.Equal(t, "octap", notifyAction.Title)
		gt.Equal(t, "Test message", notifyAction.Message)
		gt.Nil(t, notifyAction.Sound) // default should be nil
	})

	t.Run("ToNotifyAction with wrong type", func(t *testing.T) {
		action := model.Action{
			Type: "sound",
			Data: map[string]interface{}{
				"path": "/path/to/sound.mp3",
			},
		}

		_, err := action.ToNotifyAction()
		gt.Error(t, err)
	})

	t.Run("ToNotifyAction without message", func(t *testing.T) {
		action := model.Action{
			Type: "notify",
			Data: map[string]interface{}{
				"title": "Test Title",
			},
		}

		_, err := action.ToNotifyAction()
		gt.Error(t, err)
	})
}
