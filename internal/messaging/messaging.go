package messaging

import (
	"briefly/internal/render"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MessagePlatform represents different messaging platforms
type MessagePlatform string

const (
	PlatformSlack   MessagePlatform = "slack"
	PlatformDiscord MessagePlatform = "discord"
)

// MessageFormat represents different message formats
type MessageFormat string

const (
	FormatBullets    MessageFormat = "bullets"    // Short bullet points
	FormatSummary    MessageFormat = "summary"    // Brief summary
	FormatHighlights MessageFormat = "highlights" // Key highlights only
)

// SlackMessage represents a Slack message structure
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
}

// SlackBlock represents a Slack block kit element
type SlackBlock struct {
	Type      string              `json:"type"`
	Text      *SlackText          `json:"text,omitempty"`
	Elements  []SlackBlockElement `json:"elements,omitempty"`
	Accessory *SlackAccessory     `json:"accessory,omitempty"`
}

// SlackText represents text in Slack blocks
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackBlockElement represents elements within blocks
type SlackBlockElement struct {
	Type string     `json:"type"`
	Text *SlackText `json:"text,omitempty"`
	URL  string     `json:"url,omitempty"`
}

// SlackAccessory represents block accessories
type SlackAccessory struct {
	Type string `json:"type"`
	Text string `json:"text"`
	URL  string `json:"url"`
}

// SlackAttachment represents legacy Slack attachments
type SlackAttachment struct {
	Color  string       `json:"color,omitempty"`
	Title  string       `json:"title,omitempty"`
	Text   string       `json:"text,omitempty"`
	Fields []SlackField `json:"fields,omitempty"`
	Footer string       `json:"footer,omitempty"`
	Ts     int64        `json:"ts,omitempty"`
}

// SlackField represents fields in attachments
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// DiscordMessage represents a Discord message structure
type DiscordMessage struct {
	Content   string         `json:"content,omitempty"`
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

// DiscordEmbedField represents fields in Discord embeds
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedFooter represents footer in Discord embeds
type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// MessagingClient handles sending messages to different platforms
type MessagingClient struct {
	SlackWebhookURL   string
	DiscordWebhookURL string
	HTTPClient        *http.Client
}

// NewMessagingClient creates a new messaging client
func NewMessagingClient(slackURL, discordURL string) *MessagingClient {
	return &MessagingClient{
		SlackWebhookURL:   slackURL,
		DiscordWebhookURL: discordURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ConvertToSlackMessage converts digest data to Slack message format
func ConvertToSlackMessage(digestItems []render.DigestData, title string, format MessageFormat, includeSentiment bool) *SlackMessage {
	switch format {
	case FormatBullets:
		return createSlackBulletMessage(digestItems, title, includeSentiment)
	case FormatSummary:
		return createSlackSummaryMessage(digestItems, title, includeSentiment)
	case FormatHighlights:
		return createSlackHighlightsMessage(digestItems, title, includeSentiment)
	default:
		return createSlackBulletMessage(digestItems, title, includeSentiment)
	}
}

// ConvertToDiscordMessage converts digest data to Discord message format
func ConvertToDiscordMessage(digestItems []render.DigestData, title string, format MessageFormat, includeSentiment bool) *DiscordMessage {
	switch format {
	case FormatBullets:
		return createDiscordBulletMessage(digestItems, title, includeSentiment)
	case FormatSummary:
		return createDiscordSummaryMessage(digestItems, title, includeSentiment)
	case FormatHighlights:
		return createDiscordHighlightsMessage(digestItems, title, includeSentiment)
	default:
		return createDiscordBulletMessage(digestItems, title, includeSentiment)
	}
}

// createSlackBulletMessage creates a bullet-point style Slack message
func createSlackBulletMessage(digestItems []render.DigestData, title string, includeSentiment bool) *SlackMessage {
	var blocks []SlackBlock

	// Header block
	blocks = append(blocks, SlackBlock{
		Type: "header",
		Text: &SlackText{
			Type: "plain_text",
			Text: title,
		},
	})

	// Divider
	blocks = append(blocks, SlackBlock{Type: "divider"})

	// Article bullets
	var bulletText strings.Builder
	for i, item := range digestItems {
		if i >= 10 { // Limit to 10 items for messaging platforms
			break
		}

		sentimentPrefix := ""
		if includeSentiment && item.SentimentEmoji != "" {
			sentimentPrefix = item.SentimentEmoji + " "
		}

		// Create short summary (max 100 chars)
		summary := item.SummaryText
		if len(summary) > 100 {
			summary = summary[:97] + "..."
		}

		bulletText.WriteString(fmt.Sprintf("• %s*%s*\n%s\n\n", sentimentPrefix, item.Title, summary))
	}

	blocks = append(blocks, SlackBlock{
		Type: "section",
		Text: &SlackText{
			Type: "mrkdwn",
			Text: bulletText.String(),
		},
	})

	// Footer
	blocks = append(blocks, SlackBlock{
		Type: "context",
		Elements: []SlackBlockElement{
			{
				Type: "mrkdwn",
				Text: &SlackText{
					Type: "mrkdwn",
					Text: fmt.Sprintf("📱 Generated by Briefly • %d articles • %s", len(digestItems), time.Now().Format("Jan 2, 3:04 PM")),
				},
			},
		},
	})

	return &SlackMessage{
		Blocks:    blocks,
		Username:  "Briefly",
		IconEmoji: ":newspaper:",
	}
}

// createSlackSummaryMessage creates a summary-style Slack message
func createSlackSummaryMessage(digestItems []render.DigestData, title string, includeSentiment bool) *SlackMessage {
	return &SlackMessage{
		Text:      fmt.Sprintf("📰 %s", title),
		Username:  "Briefly",
		IconEmoji: ":newspaper:",
		Attachments: []SlackAttachment{
			{
				Color:  "#2563eb",
				Title:  title,
				Text:   fmt.Sprintf("Summary of %d articles:", len(digestItems)),
				Fields: createSlackFieldsFromDigest(digestItems, includeSentiment),
				Footer: "Generated by Briefly",
				Ts:     time.Now().Unix(),
			},
		},
	}
}

// createSlackHighlightsMessage creates a highlights-style Slack message
func createSlackHighlightsMessage(digestItems []render.DigestData, title string, includeSentiment bool) *SlackMessage {
	var highlights []string
	for i, item := range digestItems {
		if i >= 5 { // Limit to top 5 highlights
			break
		}

		sentimentPrefix := ""
		if includeSentiment && item.SentimentEmoji != "" {
			sentimentPrefix = item.SentimentEmoji + " "
		}

		highlights = append(highlights, fmt.Sprintf("%s%s", sentimentPrefix, item.Title))
	}

	text := fmt.Sprintf("🎯 *%s*\n\nKey highlights:\n• %s", title, strings.Join(highlights, "\n• "))

	return &SlackMessage{
		Text:      text,
		Username:  "Briefly",
		IconEmoji: ":dart:",
	}
}

// createSlackFieldsFromDigest creates Slack fields from digest items
func createSlackFieldsFromDigest(digestItems []render.DigestData, includeSentiment bool) []SlackField {
	var fields []SlackField

	for i, item := range digestItems {
		if i >= 5 { // Limit to 5 fields to avoid message size limits
			break
		}

		title := item.Title
		if includeSentiment && item.SentimentEmoji != "" {
			title = item.SentimentEmoji + " " + title
		}

		// Truncate summary for field
		summary := item.SummaryText
		if len(summary) > 150 {
			summary = summary[:147] + "..."
		}

		fields = append(fields, SlackField{
			Title: title,
			Value: summary,
			Short: false,
		})
	}

	return fields
}

// createDiscordBulletMessage creates a bullet-point style Discord message
func createDiscordBulletMessage(digestItems []render.DigestData, title string, includeSentiment bool) *DiscordMessage {
	var description strings.Builder

	for i, item := range digestItems {
		if i >= 10 { // Limit to 10 items
			break
		}

		sentimentPrefix := ""
		if includeSentiment && item.SentimentEmoji != "" {
			sentimentPrefix = item.SentimentEmoji + " "
		}

		// Create short summary
		summary := item.SummaryText
		if len(summary) > 100 {
			summary = summary[:97] + "..."
		}

		description.WriteString(fmt.Sprintf("• %s**%s**\n%s\n\n", sentimentPrefix, item.Title, summary))
	}

	embed := DiscordEmbed{
		Title:       title,
		Description: description.String(),
		Color:       0x2563eb, // Blue color
		Footer: &DiscordEmbedFooter{
			Text: fmt.Sprintf("Generated by Briefly • %d articles", len(digestItems)),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return &DiscordMessage{
		Username:  "Briefly",
		AvatarURL: "", // Could be set to a bot avatar URL
		Embeds:    []DiscordEmbed{embed},
	}
}

// createDiscordSummaryMessage creates a summary-style Discord message
func createDiscordSummaryMessage(digestItems []render.DigestData, title string, includeSentiment bool) *DiscordMessage {
	embed := DiscordEmbed{
		Title:       title,
		Description: fmt.Sprintf("Summary of %d articles", len(digestItems)),
		Color:       0x2563eb,
		Fields:      createDiscordFieldsFromDigest(digestItems, includeSentiment),
		Footer: &DiscordEmbedFooter{
			Text: "Generated by Briefly",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return &DiscordMessage{
		Username: "Briefly",
		Embeds:   []DiscordEmbed{embed},
	}
}

// createDiscordHighlightsMessage creates a highlights-style Discord message
func createDiscordHighlightsMessage(digestItems []render.DigestData, title string, includeSentiment bool) *DiscordMessage {
	var description strings.Builder
	description.WriteString("🎯 **Key highlights:**\n\n")

	for i, item := range digestItems {
		if i >= 5 { // Limit to top 5 highlights
			break
		}

		sentimentPrefix := ""
		if includeSentiment && item.SentimentEmoji != "" {
			sentimentPrefix = item.SentimentEmoji + " "
		}

		description.WriteString(fmt.Sprintf("• %s%s\n", sentimentPrefix, item.Title))
	}

	embed := DiscordEmbed{
		Title:       title,
		Description: description.String(),
		Color:       0x10b981, // Green color for highlights
		Footer: &DiscordEmbedFooter{
			Text: "Generated by Briefly",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return &DiscordMessage{
		Username: "Briefly",
		Embeds:   []DiscordEmbed{embed},
	}
}

// createDiscordFieldsFromDigest creates Discord fields from digest items
func createDiscordFieldsFromDigest(digestItems []render.DigestData, includeSentiment bool) []DiscordEmbedField {
	var fields []DiscordEmbedField

	for i, item := range digestItems {
		if i >= 5 { // Limit to 5 fields
			break
		}

		title := item.Title
		if includeSentiment && item.SentimentEmoji != "" {
			title = item.SentimentEmoji + " " + title
		}

		// Truncate summary for field
		summary := item.SummaryText
		if len(summary) > 150 {
			summary = summary[:147] + "..."
		}

		fields = append(fields, DiscordEmbedField{
			Name:   title,
			Value:  summary,
			Inline: false,
		})
	}

	return fields
}

// SendSlackMessage sends a message to Slack webhook
func (c *MessagingClient) SendSlackMessage(message *SlackMessage) error {
	if c.SlackWebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.SlackWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %s\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendDiscordMessage sends a message to Discord webhook
func (c *MessagingClient) SendDiscordMessage(message *DiscordMessage) error {
	if c.DiscordWebhookURL == "" {
		return fmt.Errorf("discord webhook URL not configured")
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.DiscordWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Discord message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendMessage sends a message to the specified platform
func (c *MessagingClient) SendMessage(platform MessagePlatform, digestItems []render.DigestData, title string, format MessageFormat, includeSentiment bool) error {
	switch platform {
	case PlatformSlack:
		message := ConvertToSlackMessage(digestItems, title, format, includeSentiment)
		return c.SendSlackMessage(message)
	case PlatformDiscord:
		message := ConvertToDiscordMessage(digestItems, title, format, includeSentiment)
		return c.SendDiscordMessage(message)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

// ValidateWebhookURL validates if a webhook URL is properly formatted
func ValidateWebhookURL(platform MessagePlatform, url string) error {
	if url == "" {
		return fmt.Errorf("%s webhook URL cannot be empty", platform)
	}

	switch platform {
	case PlatformSlack:
		if !strings.Contains(url, "hooks.slack.com") {
			return fmt.Errorf("invalid Slack webhook URL format")
		}
	case PlatformDiscord:
		if !strings.Contains(url, "discord.com/api/webhooks") {
			return fmt.Errorf("invalid Discord webhook URL format")
		}
	default:
		return fmt.Errorf("unknown platform: %s", platform)
	}

	return nil
}

// GetAvailableFormats returns available message formats
func GetAvailableFormats() []string {
	return []string{
		string(FormatBullets),
		string(FormatSummary),
		string(FormatHighlights),
	}
}

// GetAvailablePlatforms returns available messaging platforms
func GetAvailablePlatforms() []string {
	return []string{
		string(PlatformSlack),
		string(PlatformDiscord),
	}
}
