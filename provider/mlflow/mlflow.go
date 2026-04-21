// Package mlflow implements provider.Backend for MLflow Tracking Server.
// Maps: experiments → groups, runs → streams, metric history → events.
//
// Supports two connection modes:
//   - Direct: connect to a self-hosted MLflow server via --mlflow-url
//   - SageMaker: connect via presigned URL from a SageMaker-managed MLflow
//     tracking server via --mlflow-arn (uses AWS credentials)
package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/derricw/cwl/provider"
)

type Backend struct {
	baseURL    string
	client     *http.Client
	retryDelay time.Duration // base unit for backoff; 0 disables sleep
	// SageMaker re-auth fields (nil for direct connections)
	smClient   *sagemaker.Client
	serverName string
}

// New creates a backend for a direct MLflow server URL.
func New(baseURL string) *Backend {
	return &Backend{
		baseURL:    strings.TrimRight(baseURL, "/"),
		client:     &http.Client{Timeout: 30 * time.Second},
		retryDelay: time.Second,
	}
}

// NewFromSageMakerARN creates a backend by generating a presigned URL from a
// SageMaker MLflow tracking server ARN. The ARN format is:
// arn:aws:sagemaker:<region>:<account>:mlflow-tracking-server/<name>
//
// This calls CreatePresignedMlflowTrackingServerUrl to get an auth URL,
// then hits it to establish a session cookie. Subsequent API calls reuse
// the authenticated session.
func NewFromSageMakerARN(arn, awsProfile string) (*Backend, error) {
	// Parse region and server name from ARN
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid SageMaker MLflow ARN: %s", arn)
	}
	region := parts[3]
	resource := parts[5] // "mlflow-tracking-server/<name>"
	nameParts := strings.SplitN(resource, "/", 2)
	if len(nameParts) != 2 {
		return nil, fmt.Errorf("invalid SageMaker MLflow ARN resource: %s", resource)
	}
	serverName := nameParts[1]

	// Create SageMaker client
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(awsProfile),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	smClient := sagemaker.NewFromConfig(cfg)

	// Get presigned URL
	resp, err := smClient.CreatePresignedMlflowTrackingServerUrl(context.TODO(),
		&sagemaker.CreatePresignedMlflowTrackingServerUrlInput{
			TrackingServerName: &serverName,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create presigned URL: %w", err)
	}

	authURL := *resp.AuthorizedUrl

	// Extract base URL (everything before /auth?token=...)
	baseURL := authURL
	if idx := strings.Index(authURL, "/auth?"); idx != -1 {
		baseURL = authURL[:idx]
	}

	// Create HTTP client with cookie jar to maintain session
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	// Hit the auth URL to establish session cookie
	authResp, err := client.Get(authURL)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with MLflow: %w", err)
	}
	authResp.Body.Close()

	return &Backend{
		baseURL:    baseURL,
		client:     client,
		retryDelay: time.Second,
		smClient:   smClient,
		serverName: serverName,
	}, nil
}

// API response types

type experiment struct {
	ExperimentID string `json:"experiment_id"`
	Name         string `json:"name"`
}

type searchExperimentsResp struct {
	Experiments   []experiment `json:"experiments"`
	NextPageToken string       `json:"next_page_token"`
}

type runInfo struct {
	RunID     string `json:"run_id"`
	RunName   string `json:"run_name"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    string `json:"status"`
}

type run struct {
	Info runInfo `json:"info"`
	Data runData `json:"data"`
}

type runData struct {
	Metrics []metric `json:"metrics"`
	Params  []param  `json:"params"`
}

type param struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type searchRunsResp struct {
	Runs          []run  `json:"runs"`
	NextPageToken string `json:"next_page_token"`
}

type getRunResp struct {
	Run run `json:"run"`
}

type metric struct {
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
	Step      int64   `json:"step"`
}

type metricHistoryResp struct {
	Metrics       []metric `json:"metrics"`
	NextPageToken string   `json:"next_page_token"`
}

// Backend interface implementation

func (b *Backend) FetchGroups(pattern string) ([]provider.LogGroup, error) {
	var all []provider.LogGroup
	var pageToken string
	for {
		body := map[string]interface{}{"max_results": 1000}
		if pageToken != "" {
			body["page_token"] = pageToken
		}
		if pattern != "" {
			body["filter"] = fmt.Sprintf("name ILIKE '%%%s%%'", pattern)
		}
		var resp searchExperimentsResp
		if err := b.post("/api/2.0/mlflow/experiments/search", body, &resp); err != nil {
			return nil, err
		}
		for _, exp := range resp.Experiments {
			all = append(all, provider.LogGroup{
				Name: exp.Name,
				Desc: "experiment_id=" + exp.ExperimentID,
			})
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return all, nil
}

func (b *Backend) FetchStreamsStreaming(group string, callback func([]provider.LogStream) error) error {
	expID, err := b.getExperimentID(group)
	if err != nil {
		return err
	}
	var pageToken string
	for {
		body := map[string]interface{}{
			"experiment_ids": []string{expID},
			"max_results":    1000,
			"order_by":       []string{"start_time DESC"},
		}
		if pageToken != "" {
			body["page_token"] = pageToken
		}
		var resp searchRunsResp
		if err := b.post("/api/2.0/mlflow/runs/search", body, &resp); err != nil {
			return err
		}
		if len(resp.Runs) > 0 {
			streams := make([]provider.LogStream, len(resp.Runs))
			for i, r := range resp.Runs {
				name := r.Info.RunName
				if name == "" {
					name = r.Info.RunID
				}
				streams[i] = provider.LogStream{Name: name}
				if r.Info.StartTime > 0 {
					t := time.UnixMilli(r.Info.StartTime)
					streams[i].LastEventTime = &t
				}
			}
			if err := callback(streams); err != nil {
				return err
			}
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return nil
}

func (b *Backend) FetchEventsStreaming(group, stream string, callback func([]provider.LogEvent) error) error {
	runID, err := b.resolveRunID(group, stream)
	if err != nil {
		return err
	}
	keys, err := b.getMetricKeys(runID)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return fmt.Errorf("no metrics found for run %s", runID)
	}
	for _, key := range keys {
		if err := b.fetchMetricHistory(runID, key, callback); err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) FetchLastEvents(group, stream string, limit int) ([]provider.LogEvent, error) {
	runID, err := b.resolveRunID(group, stream)
	if err != nil {
		return nil, err
	}
	keys, err := b.getMetricKeys(runID)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}
	// For preview, just show latest values for each metric
	var events []provider.LogEvent
	for _, key := range keys {
		err := b.fetchMetricHistory(runID, key, func(batch []provider.LogEvent) error {
			events = append(events, batch...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(events) > limit {
		events = events[len(events)-limit:]
	}
	return events, nil
}

func (b *Backend) FetchNewEvents(group, stream string, since *time.Time) ([]provider.LogEvent, error) {
	// MLflow metrics are immutable once logged — no new events after initial fetch
	return nil, nil
}

// Helpers

// getMetricKeys fetches the run and returns all metric keys it has logged.
func (b *Backend) getMetricKeys(runID string) ([]string, error) {
	var resp getRunResp
	url := fmt.Sprintf("/api/2.0/mlflow/runs/get?run_id=%s", runID)
	if err := b.get(url, &resp); err != nil {
		return nil, err
	}
	keys := make([]string, len(resp.Run.Data.Metrics))
	for i, m := range resp.Run.Data.Metrics {
		keys[i] = m.Key
	}
	return keys, nil
}

func (b *Backend) getExperimentID(name string) (string, error) {
	var resp struct {
		Experiment experiment `json:"experiment"`
	}
	url := fmt.Sprintf("/api/2.0/mlflow/experiments/get-by-name?experiment_name=%s", name)
	if err := b.get(url, &resp); err != nil {
		return "", fmt.Errorf("experiment %q not found: %w", name, err)
	}
	return resp.Experiment.ExperimentID, nil
}

func (b *Backend) resolveRunID(group, stream string) (string, error) {
	expID, err := b.getExperimentID(group)
	if err != nil {
		return "", err
	}
	body := map[string]interface{}{
		"experiment_ids": []string{expID},
		"filter":         fmt.Sprintf("run_name = '%s'", stream),
		"max_results":    1,
	}
	var resp searchRunsResp
	if err := b.post("/api/2.0/mlflow/runs/search", body, &resp); err != nil {
		return "", err
	}
	if len(resp.Runs) == 0 {
		return stream, nil
	}
	return resp.Runs[0].Info.RunID, nil
}

func (b *Backend) fetchMetricHistory(runID, metricKey string, callback func([]provider.LogEvent) error) error {
	var pageToken string
	for {
		url := fmt.Sprintf("/api/2.0/mlflow/metrics/get-history?run_id=%s&metric_key=%s&max_results=10000", runID, metricKey)
		if pageToken != "" {
			url += "&page_token=" + pageToken
		}
		var resp metricHistoryResp
		if err := b.get(url, &resp); err != nil {
			return err
		}
		if len(resp.Metrics) > 0 {
			events := make([]provider.LogEvent, len(resp.Metrics))
			for i, m := range resp.Metrics {
				t := time.UnixMilli(m.Timestamp)
				events[i] = provider.LogEvent{
					Message:   fmt.Sprintf("step=%d %s=%g", m.Step, m.Key, m.Value),
					Timestamp: &t,
				}
			}
			if err := callback(events); err != nil {
				return err
			}
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return nil
}

func (b *Backend) post(path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * b.retryDelay) // 1x, 4x, 9x
		}
		resp, err := b.client.Post(b.baseURL+path, "application/json", strings.NewReader(string(data)))
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusForbidden {
			resp.Body.Close()
			if err := b.reauth(); err != nil {
				return err
			}
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("mlflow API error: %s", resp.Status)
		}
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return fmt.Errorf("mlflow API error: too many retries")
}

func (b *Backend) get(path string, result interface{}) error {
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * b.retryDelay)
		}
		resp, err := b.client.Get(b.baseURL + path)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusForbidden {
			resp.Body.Close()
			if err := b.reauth(); err != nil {
				return err
			}
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("mlflow API error: %s", resp.Status)
		}
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return fmt.Errorf("mlflow API error: too many retries")
}

// reauth generates a new presigned URL and re-establishes the session.
// Only applicable for SageMaker-backed connections.
func (b *Backend) reauth() error {
	if b.smClient == nil {
		return fmt.Errorf("session expired and no SageMaker client available for re-auth")
	}
	resp, err := b.smClient.CreatePresignedMlflowTrackingServerUrl(context.TODO(),
		&sagemaker.CreatePresignedMlflowTrackingServerUrlInput{
			TrackingServerName: &b.serverName,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to refresh presigned URL: %w", err)
	}
	authResp, err := b.client.Get(*resp.AuthorizedUrl)
	if err != nil {
		return fmt.Errorf("failed to re-authenticate: %w", err)
	}
	authResp.Body.Close()
	return nil
}
