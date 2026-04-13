package confluence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
)

// Client wraps an HTTP client for Confluence API calls.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Confluence API client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Page represents a Confluence page.
type Page struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

func (c *Client) authHeaders(auth config.BitbucketAuth) (map[string]string, error) {
	var credentials string
	switch {
	case auth.Email != "" && auth.APIToken != "":
		credentials = auth.Email + ":" + auth.APIToken
	case auth.Username != "" && auth.AppPassword != "":
		credentials = auth.Username + ":" + auth.AppPassword
	default:
		return nil, fmt.Errorf("invalid auth config for Confluence")
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))

	return map[string]string{
		"Authorization": "Basic " + encoded,
		"Accept":        "application/json",
		"Content-Type":  "application/json",
	}, nil
}

func (c *Client) doRequest(ctx context.Context, method, url string, headers map[string]string, body string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create confluence request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute confluence request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read confluence response: %w", err)
	}

	return data, resp.StatusCode, nil
}

// GetPage fetches a Confluence page by ID, returning its title and storage-format body.
func (c *Client) GetPage(ctx context.Context, auth config.BitbucketAuth, baseURL, pageID string) (*Page, error) {
	apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s?body-format=storage", strings.TrimRight(baseURL, "/"), pageID)

	headers, err := c.authHeaders(auth)
	if err != nil {
		return nil, err
	}

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("confluence API error fetching page (%d): %s", status, string(data))
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse confluence page: %w", err)
	}

	page := &Page{
		ID:    pageID,
		Title: jsonStr(raw, "title"),
	}

	if bodyObj, ok := raw["body"].(map[string]any); ok {
		if storage, ok := bodyObj["storage"].(map[string]any); ok {
			page.Body = jsonStr(storage, "value")
		}
	}

	if links, ok := raw["_links"].(map[string]any); ok {
		if webUI, ok := links["webui"].(string); ok {
			page.URL = strings.TrimRight(baseURL, "/") + "/wiki" + webUI
		}
	}

	return page, nil
}

// CreatePage creates a new Confluence page under the given parent.
func (c *Client) CreatePage(ctx context.Context, auth config.BitbucketAuth, baseURL, spaceID, parentID, title, body string) (*Page, error) {
	apiURL := fmt.Sprintf("%s/wiki/api/v2/pages", strings.TrimRight(baseURL, "/"))

	headers, err := c.authHeaders(auth)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"spaceId":  spaceID,
		"status":   "current",
		"title":    title,
		"parentId": parentID,
		"body": map[string]any{
			"representation": "storage",
			"value":          body,
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal confluence page body: %w", err)
	}

	data, status, err := c.doRequest(ctx, "POST", apiURL, headers, string(payloadJSON))
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("confluence API error creating page (%d): %s", status, string(data))
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse confluence create response: %w", err)
	}

	page := &Page{
		ID:    jsonStr(raw, "id"),
		Title: jsonStr(raw, "title"),
	}

	if links, ok := raw["_links"].(map[string]any); ok {
		if webUI, ok := links["webui"].(string); ok {
			page.URL = strings.TrimRight(baseURL, "/") + "/wiki" + webUI
		}
	}

	return page, nil
}

// GetSpaceID resolves a Confluence space key to its space ID.
func (c *Client) GetSpaceID(ctx context.Context, auth config.BitbucketAuth, baseURL, spaceKey string) (string, error) {
	apiURL := fmt.Sprintf("%s/wiki/api/v2/spaces?keys=%s", strings.TrimRight(baseURL, "/"), spaceKey)

	headers, err := c.authHeaders(auth)
	if err != nil {
		return "", err
	}

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("confluence API error fetching space (%d): %s", status, string(data))
	}

	var resp struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse confluence space response: %w", err)
	}

	if len(resp.Results) == 0 {
		return "", fmt.Errorf("confluence space %q not found", spaceKey)
	}

	return resp.Results[0].ID, nil
}

func jsonStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		switch s := v.(type) {
		case string:
			return s
		case float64:
			return fmt.Sprintf("%.0f", s)
		}
	}
	return ""
}
