package usecase_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/domain/model"
	"github.com/m-mizutani/octap/pkg/usecase"
)

func TestSlackAction(t *testing.T) {
	t.Run("Send basic notification", func(t *testing.T) {
		// Setup test server
		var receivedPayload model.SlackPayload
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gt.Equal(t, r.Method, http.MethodPost)
			gt.Equal(t, r.Header.Get("Content-Type"), "application/json")

			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&receivedPayload)
			gt.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		// Create action
		action := model.Action{
			Type: "slack",
			Data: map[string]interface{}{
				"webhook_url": server.URL,
				"message":     "Test notification for {{.Repository}}",
			},
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test-workflow",
			RunID:      123,
			URL:        "https://github.com/test/repo/actions/runs/123",
		}

		// Execute
		slackAction := usecase.NewSlackAction()
		err := slackAction.Execute(context.Background(), action, event)
		gt.NoError(t, err)

		// Verify payload
		gt.Equal(t, receivedPayload.Text, "Test notification for test/repo")
	})

	t.Run("Send notification with color", func(t *testing.T) {
		// Setup test server
		var receivedPayload model.SlackPayload
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&receivedPayload)
			gt.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		// Create action with color
		action := model.Action{
			Type: "slack",
			Data: map[string]interface{}{
				"webhook_url": server.URL,
				"message":     "{{.Workflow}} failed",
				"color":       "danger",
			},
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckFailure,
			Repository: "test/repo",
			Workflow:   "CI Build",
			RunID:      456,
			URL:        "https://github.com/test/repo/actions/runs/456",
		}

		// Execute
		slackAction := usecase.NewSlackAction()
		err := slackAction.Execute(context.Background(), action, event)
		gt.NoError(t, err)

		// Verify payload has attachment with color
		gt.Equal(t, len(receivedPayload.Attachments), 1)
		gt.Equal(t, receivedPayload.Attachments[0].Color, "danger")
		gt.Equal(t, receivedPayload.Attachments[0].Text, "CI Build failed")
	})

	t.Run("Handle server error", func(t *testing.T) {
		// Setup test server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		// Create action
		action := model.Action{
			Type: "slack",
			Data: map[string]interface{}{
				"webhook_url": server.URL,
				"message":     "Test",
			},
		}

		// Create event
		event := model.WorkflowEvent{
			Type:       model.HookCheckSuccess,
			Repository: "test/repo",
			Workflow:   "test",
			RunID:      789,
		}

		// Execute - should return error
		slackAction := usecase.NewSlackAction()
		err := slackAction.Execute(context.Background(), action, event)
		gt.Error(t, err)
	})

	t.Run("Invalid action data", func(t *testing.T) {
		testCases := []struct {
			name string
			data map[string]interface{}
		}{
			{
				name: "missing webhook_url",
				data: map[string]interface{}{
					"message": "Test",
				},
			},
			{
				name: "missing message",
				data: map[string]interface{}{
					"webhook_url": "https://hooks.slack.com/test",
				},
			},
			{
				name: "empty webhook_url",
				data: map[string]interface{}{
					"webhook_url": "",
					"message":     "Test",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				action := model.Action{
					Type: "slack",
					Data: tc.data,
				}

				event := model.WorkflowEvent{
					Type:       model.HookCheckSuccess,
					Repository: "test/repo",
					Workflow:   "test",
					RunID:      123,
				}

				slackAction := usecase.NewSlackAction()
				err := slackAction.Execute(context.Background(), action, event)
				gt.Error(t, err)
			})
		}
	})
}
