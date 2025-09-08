package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"github.com/m-mizutani/octap/pkg/domain/model"
)

type slackAction struct {
	httpClient *http.Client
}

// NewSlackAction creates a new SlackAction instance
func NewSlackAction() interfaces.ActionExecutor {
	return &slackAction{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute sends a notification to Slack
func (s *slackAction) Execute(ctx context.Context, action model.Action, event model.WorkflowEvent) error {
	logger := ctxlog.From(ctx)

	logger.Debug("slackAction.Execute called",
		slog.String("event_type", string(event.Type)),
		slog.Any("action_data", action.Data),
	)

	// Convert to typed action
	slackAction, err := action.ToSlackAction()
	if err != nil {
		logger.Error("Failed to parse slack action",
			slog.String("error", err.Error()),
		)
		return goerr.Wrap(err, "failed to parse slack action")
	}

	// Expand environment variables in webhook URL
	webhookURL := expandEnvVars(slackAction.WebhookURL)
	if webhookURL == "" {
		return goerr.New("webhook URL is empty after expansion")
	}

	// Build message from template
	message, err := s.buildMessage(slackAction.Message, event)
	if err != nil {
		logger.Error("Failed to build message from template",
			slog.String("template", slackAction.Message),
			slog.String("error", err.Error()),
		)
		return goerr.Wrap(err, "failed to build message")
	}

	// Prepare payload
	payload := model.SlackPayload{
		Text: message,
	}

	// Add optional fields
	if slackAction.UserName != "" {
		payload.UserName = slackAction.UserName
	}
	if slackAction.IconEmoji != "" {
		payload.IconEmoji = slackAction.IconEmoji
	}

	// Add attachment with color if specified
	if slackAction.Color != "" {
		payload.Attachments = []model.Attachment{
			{
				Color:     slackAction.Color,
				Text:      message,
				Footer:    fmt.Sprintf("octap - %s", event.Repository),
				Timestamp: time.Now().Unix(),
			},
		}
		// Clear main text to avoid duplication
		payload.Text = ""
	}

	// Send to Slack with retry
	for attempt := 0; attempt < 3; attempt++ {
		err = s.sendToSlack(ctx, webhookURL, payload)
		if err == nil {
			logger.Debug("Slack notification sent successfully",
				slog.Int("attempt", attempt+1),
			)
			return nil
		}

		// Check if it's a rate limit error
		if strings.Contains(err.Error(), "429") && attempt < 2 {
			// Exponential backoff: 1s, 2s
			backoff := time.Duration(1<<attempt) * time.Second
			logger.Warn("Rate limited by Slack, retrying",
				slog.Int("attempt", attempt+1),
				slog.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
			continue
		}

		// For other errors, retry only once
		if attempt == 0 && !strings.Contains(err.Error(), "4") {
			logger.Warn("Failed to send Slack notification, retrying",
				slog.Int("attempt", attempt+1),
				slog.String("error", err.Error()),
			)
			time.Sleep(1 * time.Second)
			continue
		}

		// Give up
		break
	}

	logger.Error("Failed to send Slack notification after retries",
		slog.String("error", err.Error()),
	)
	return goerr.Wrap(err, "failed to send slack notification")
}

// buildMessage processes the message template
func (s *slackAction) buildMessage(messageTemplate string, event model.WorkflowEvent) (string, error) {
	// Prepare template data
	data := struct {
		Repository string
		Workflow   string
		RunID      int64
		EventType  string
		RunURL     string
		Timestamp  time.Time
	}{
		Repository: event.Repository,
		Workflow:   event.Workflow,
		RunID:      event.RunID,
		EventType:  string(event.Type),
		RunURL:     event.URL,
		Timestamp:  time.Now(),
	}

	// Parse and execute template
	tmpl, err := template.New("message").Parse(messageTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse message template")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute message template")
	}

	return buf.String(), nil
}

// sendToSlack sends the payload to Slack webhook
func (s *slackAction) sendToSlack(ctx context.Context, webhookURL string, payload model.SlackPayload) error {
	logger := ctxlog.From(ctx)

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal slack payload")
	}

	// Log payload (mask webhook URL for security)
	maskedURL := maskWebhookURL(webhookURL)
	logger.Debug("Sending to Slack",
		slog.String("webhook_url", maskedURL),
		slog.String("payload", string(jsonData)),
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return goerr.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return goerr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		var respBody bytes.Buffer
		_, _ = respBody.ReadFrom(resp.Body) // Best effort to read response
		return goerr.New(fmt.Sprintf("slack webhook returned status %d: %s", resp.StatusCode, respBody.String()))
	}

	return nil
}

// expandEnvVars expands environment variables in the string
func expandEnvVars(s string) string {
	// Support both ${VAR} and $VAR formats
	return os.ExpandEnv(s)
}

// maskWebhookURL masks the webhook URL for logging
func maskWebhookURL(url string) string {
	if strings.Contains(url, "hooks.slack.com") {
		parts := strings.Split(url, "/")
		if len(parts) > 3 {
			// Mask the token parts
			for i := len(parts) - 3; i < len(parts); i++ {
				if len(parts[i]) > 4 {
					parts[i] = parts[i][:2] + "***"
				}
			}
			return strings.Join(parts, "/")
		}
	}
	// For non-Slack URLs, just mask the end
	if len(url) > 20 {
		return url[:20] + "***"
	}
	return "***"
}
