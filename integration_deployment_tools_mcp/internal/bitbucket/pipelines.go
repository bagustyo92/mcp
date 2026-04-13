package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
)

// TriggerPipeline starts a Bitbucket pipeline for the given repository.
func (c *Client) TriggerPipeline(ctx context.Context, auth config.BitbucketAuth, workspace, repo, refType, refName, pipelineName string) (*PipelineRun, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/pipelines/", apiBase, workspace, repo)

	headers := postHeaders(auth)

	body := map[string]any{
		"target": map[string]any{
			"type":     "pipeline_ref_target",
			"ref_type": refType,
			"ref_name": refName,
			"selector": map[string]string{
				"type":    "custom",
				"pattern": pipelineName,
			},
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal pipeline body: %w", err)
	}

	data, status, err := c.doRequest(ctx, "POST", apiURL, headers, string(bodyJSON))
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error triggering pipeline (%d): %s", status, string(data))
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse pipeline response: %w", err)
	}

	run := &PipelineRun{
		UUID:        jsonString(raw, "uuid"),
		BuildNumber: int(jsonFloat(raw, "build_number")),
	}

	// Build the pipeline URL
	run.URL = fmt.Sprintf("https://bitbucket.org/%s/%s/pipelines/results/%d", workspace, repo, run.BuildNumber)

	return run, nil
}
