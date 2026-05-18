// Package buddy is a thin client over the Buddy.works API used to trigger
// build pipelines from the orchestrator. Buddy is *only* an executor —
// it never sees orchestration logic.
package buddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL   string
	Workspace string
	Token     string
	HTTP      *http.Client
}

func New(baseURL, workspace, token string) *Client {
	return &Client{
		BaseURL: baseURL, Workspace: workspace, Token: token,
		HTTP: &http.Client{Timeout: 30 * time.Second},
	}
}

// TriggerInput is what the orchestrator hands Buddy. Buddy uses these as
// pipeline variables to build the right image and tag it appropriately.
type TriggerInput struct {
	Project    string            `json:"project"`
	Pipeline   string            `json:"pipeline"`
	Branch     string            `json:"branch"`
	CommitSHA  string            `json:"commit_sha"`
	ImageTag   string            `json:"image_tag"`
	Variables  map[string]string `json:"variables"`
}

type TriggerResult struct {
	RunID    string `json:"run_id"`
	Status   string `json:"status"`
	ImageRef string `json:"image_ref"`
}

// Trigger asks Buddy to start a pipeline run. We pass branch+sha and a
// pre-computed image tag the pipeline must produce.
func (c *Client) Trigger(ctx context.Context, in TriggerInput) (*TriggerResult, error) {
	url := fmt.Sprintf("%s/workspaces/%s/projects/%s/pipelines/%s/executions",
		c.BaseURL, c.Workspace, in.Project, in.Pipeline)

	body, _ := json.Marshal(map[string]any{
		"to_revision":   in.CommitSHA,
		"branch":        in.Branch,
		"variables":     toBuddyVars(in.Variables, in.ImageTag),
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	rb, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("buddy trigger %d: %s", res.StatusCode, string(rb))
	}
	var parsed struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(rb, &parsed)
	return &TriggerResult{
		RunID:    fmt.Sprintf("%d", parsed.ID),
		Status:   parsed.Status,
		ImageRef: in.ImageTag,
	}, nil
}

// GetRun queries Buddy for current run status. Used by reconciler when a
// deployment is in `building` state.
func (c *Client) GetRun(ctx context.Context, project, pipeline, runID string) (string, error) {
	url := fmt.Sprintf("%s/workspaces/%s/projects/%s/pipelines/%s/executions/%s",
		c.BaseURL, c.Workspace, project, pipeline, runID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)

	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		rb, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("buddy get run %d: %s", res.StatusCode, string(rb))
	}
	var parsed struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}
	return parsed.Status, nil
}

func toBuddyVars(in map[string]string, imageTag string) []map[string]string {
	out := make([]map[string]string, 0, len(in)+1)
	out = append(out, map[string]string{"key": "IMAGE_TAG", "value": imageTag})
	for k, v := range in {
		out = append(out, map[string]string{"key": k, "value": v})
	}
	return out
}
