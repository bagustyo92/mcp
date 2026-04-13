package gchat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Notifier sends formatted messages to Google Chat webhooks.
type Notifier struct {
	httpClient *http.Client
}

// NewNotifier creates a new Notifier.
func NewNotifier() *Notifier {
	return &Notifier{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (n *Notifier) send(ctx context.Context, webhookURL, text string) error {
	payload := map[string]string{"text": text}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// ChangeEntry represents one undeployed change for notifications.
type ChangeEntry struct {
	JiraTicket string
	JiraURL    string
	PRTitle    string
	PRURL      string
	Author     string
	PRSummary  string // first line of the PR description
}

// UndeployedChangesMessage is the data for an undeployed changes notification.
type UndeployedChangesMessage struct {
	RepoSlug   string
	LatestTag  string
	TotalCount int
	Changes    []ChangeEntry
}

// RepoNotifSummary holds the notification payload for a single repository.
type RepoNotifSummary struct {
	RepoSlug  string
	LatestTag string
	Changes   []ChangeEntry
	Error     string
}

// TagCreatedMessage is the data for a tag creation notification.
type TagCreatedMessage struct {
	RepoSlug    string
	TagName     string
	CommitHash  string
	PreviousTag string
	TagURL      string
}

// PipelineMessage is the data for a pipeline trigger notification.
type PipelineMessage struct {
	RepoSlug    string
	Environment string
	RefName     string
	PipelineURL string
	BuildNumber int
}

// NotifyUndeployedChanges sends a summary of undeployed changes.
func (n *Notifier) NotifyUndeployedChanges(ctx context.Context, webhookURL string, msg UndeployedChangesMessage) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*Undeployed Changes — %s*\n", msg.RepoSlug))
	sb.WriteString(fmt.Sprintf("Latest tag: `%s` | Undeployed: *%d*\n\n", msg.LatestTag, msg.TotalCount))

	for i, c := range msg.Changes {
		if i >= 20 {
			sb.WriteString(fmt.Sprintf("... and %d more\n", msg.TotalCount-20))
			break
		}
		line := fmt.Sprintf("• %s", c.JiraTicket)
		if c.JiraURL != "" {
			line = fmt.Sprintf("• <%s|%s>", c.JiraURL, c.JiraTicket)
		}
		if c.PRTitle != "" {
			line += fmt.Sprintf(" — %s", c.PRTitle)
		}
		if c.PRURL != "" {
			line += fmt.Sprintf(" (<%s|PR>)", c.PRURL)
		}
		if c.Author != "" {
			line += fmt.Sprintf(" by %s", c.Author)
		}
		sb.WriteString(line + "\n")
	}

	return n.send(ctx, webhookURL, sb.String())
}

// NotifyTagCreated sends a tag creation notification.
func (n *Notifier) NotifyTagCreated(ctx context.Context, webhookURL string, msg TagCreatedMessage) error {
	text := fmt.Sprintf(
		"*Tag Created — %s*\nTag: `%s` (previous: `%s`)\nCommit: `%s`\n<%s|View tag>",
		msg.RepoSlug, msg.TagName, msg.PreviousTag, msg.CommitHash[:12], msg.TagURL,
	)
	return n.send(ctx, webhookURL, text)
}

// NotifyPipelineTriggered sends a pipeline trigger notification.
func (n *Notifier) NotifyPipelineTriggered(ctx context.Context, webhookURL string, msg PipelineMessage) error {
	text := fmt.Sprintf(
		"*Pipeline Triggered — %s*\nEnvironment: `%s` | Ref: `%s` | Build: #%d\n<%s|View pipeline>",
		msg.RepoSlug, msg.Environment, msg.RefName, msg.BuildNumber, msg.PipelineURL,
	)
	return n.send(ctx, webhookURL, text)
}

// cleanAuthorName strips the email portion from raw author strings like "Name <email@example.com>".
func cleanAuthorName(raw string) string {
	if idx := strings.Index(raw, " <"); idx >= 0 {
		return strings.TrimSpace(raw[:idx])
	}
	return strings.TrimSpace(raw)
}

// NotifyAllReposUndeployed sends a single combined message covering all repositories.
// Every repo is listed regardless of whether it has pending changes.
func (n *Notifier) NotifyAllReposUndeployed(ctx context.Context, webhookURL string, repos []RepoNotifSummary) error {
	var sb strings.Builder

	totalPending := 0
	for _, r := range repos {
		totalPending += len(r.Changes)
	}

	sb.WriteString("📦 *Undeployed Changes Report*\n")
	sb.WriteString(fmt.Sprintf("_%d repos checked  •  %d total pending_\n", len(repos), totalPending))

	for _, r := range repos {
		sb.WriteString("\n")

		// Short repo name: strip workspace prefix
		shortName := r.RepoSlug
		if idx := strings.LastIndex(r.RepoSlug, "/"); idx >= 0 {
			shortName = r.RepoSlug[idx+1:]
		}

		tag := r.LatestTag
		if tag == "" {
			tag = "(no tag)"
		}

		if r.Error != "" {
			sb.WriteString(fmt.Sprintf("*%s*  —  `%s`  —  ⚠️ %s\n", shortName, tag, r.Error))
			continue
		}

		if len(r.Changes) == 0 {
			sb.WriteString(fmt.Sprintf("*%s*  —  `%s`  —  ✅ up to date\n", shortName, tag))
			continue
		}

		sb.WriteString(fmt.Sprintf("*%s*  —  `%s`  —  *%d undeployed*\n", shortName, tag, len(r.Changes)))

		for i, c := range r.Changes {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(r.Changes)-20))
				break
			}

			// Jira ticket part (linked if URL available)
			ticketPart := c.JiraTicket
			if ticketPart == "" || ticketPart == "unknown ticket" {
				ticketPart = "(no ticket)"
			} else if c.JiraURL != "" {
				ticketPart = fmt.Sprintf("<%s|%s>", c.JiraURL, c.JiraTicket)
			}

			// PR title linked to PR URL
			titlePart := c.PRTitle
			if c.PRURL != "" && titlePart != "" {
				titlePart = fmt.Sprintf("<%s|%s>", c.PRURL, titlePart)
			} else if c.PRURL != "" {
				titlePart = fmt.Sprintf("<%s|View PR>", c.PRURL)
			}

			author := cleanAuthorName(c.Author)

			// Main bullet line: • TICKET — PR Title (linked) — by Author
			line := fmt.Sprintf("• %s", ticketPart)
			if titlePart != "" {
				line += fmt.Sprintf(" — %s", titlePart)
			}
			if author != "" {
				line += fmt.Sprintf(" — by *%s*", author)
			}
			sb.WriteString(line + "\n")

			// PR summary as indented second line (if available)
			if c.PRSummary != "" {
				sb.WriteString(fmt.Sprintf("   _%s_\n", c.PRSummary))
			}
		}
	}

	return n.send(ctx, webhookURL, sb.String())
}
