package model

// SlackAction represents a Slack notification action
type SlackAction struct {
	WebhookURL string `yaml:"webhook_url"`
	Message    string `yaml:"message"`
	Color      string `yaml:"color,omitempty"`      // good, warning, danger, or #hex
	IconEmoji  string `yaml:"icon_emoji,omitempty"` // :emoji: format (only works if webhook allows customization)
	UserName   string `yaml:"username,omitempty"`   // sender name (only works if webhook allows customization)
}

// SlackPayload represents the JSON payload for Slack webhook
type SlackPayload struct {
	Text        string       `json:"text"`
	UserName    string       `json:"username,omitempty"`   // Only works if webhook allows customization
	IconEmoji   string       `json:"icon_emoji,omitempty"` // Only works if webhook allows customization
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a Slack message attachment
type Attachment struct {
	Color     string  `json:"color,omitempty"`
	Title     string  `json:"title,omitempty"`
	TitleLink string  `json:"title_link,omitempty"`
	Text      string  `json:"text,omitempty"`
	Footer    string  `json:"footer,omitempty"`
	Timestamp int64   `json:"ts,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
}

// Field represents a field in Slack attachment
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}
