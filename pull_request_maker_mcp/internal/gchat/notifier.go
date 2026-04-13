package gchat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// jiraKeyRegex matches Jira-style ticket keys like TD-1234, CORE-567, etc.
var jiraKeyRegex = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

// Notifier sends messages to a Google Chat space via an incoming webhook.
type Notifier struct {
	httpClient *http.Client
}

// NewNotifier creates a Notifier with sensible defaults.
func NewNotifier() *Notifier {
	return &Notifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NotifyPRCreated sends a PR creation notification to Google Chat.
func (n *Notifier) NotifyPRCreated(ctx context.Context, webhookURL string, msg PRMessage) error {
	if webhookURL == "" {
		return fmt.Errorf("gchat webhook_url is not configured")
	}

	text := buildMessage(msg)

	payload := map[string]string{"text": text}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal gchat payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create gchat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send gchat message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gchat webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// PRMessage holds the data needed to build the notification message.
type PRMessage struct {
	Title        string
	Author       string
	AuthorUUID   string
	AuthorEmail  string // Author's email for mention (e.g. "bagus@example.com")
	Description  string
	PRURL        string
	SourceBranch string
	JiraBaseURL  string
}

// ExtractJiraTicket extracts the first Jira ticket key from a string (branch name, title, etc.).
func ExtractJiraTicket(s string) string {
	match := jiraKeyRegex.FindString(strings.ToUpper(s))
	return match
}

// JiraURL builds a full Jira ticket URL from a ticket key and base URL.
func JiraURL(baseURL, ticketKey string) string {
	if baseURL == "" || ticketKey == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/" + ticketKey
}

func buildMessage(msg PRMessage) string {
	// Extract Jira ticket from source branch or title
	ticketKey := ExtractJiraTicket(msg.SourceBranch)
	if ticketKey == "" {
		ticketKey = ExtractJiraTicket(msg.Title)
	}

	jiraLink := JiraURL(msg.JiraBaseURL, ticketKey)

	// Truncate description if too long
	desc := strings.TrimSpace(msg.Description)
	if len(desc) > 300 {
		desc = desc[:297] + "..."
	}
	if desc == "" {
		desc = "-"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*New Pull Request: %s*\n", msg.Title))

	// Display author email if available, otherwise use display name
	authorDisplay := msg.Author
	if msg.AuthorEmail != "" {
		authorDisplay = msg.AuthorEmail
	}
	sb.WriteString(fmt.Sprintf("Author: %s\n", authorDisplay))
	sb.WriteString(fmt.Sprintf("Description: %s\n", desc))

	if jiraLink != "" {
		sb.WriteString(fmt.Sprintf("Ticket: %s (%s)\n", jiraLink, ticketKey))
	} else {
		sb.WriteString("Ticket: -\n")
	}

	sb.WriteString(fmt.Sprintf("PR Link: %s\n", msg.PRURL))
	sb.WriteString("\nplease help to review and approve this pull request. <users/all> thank you!")

	return sb.String()
}
