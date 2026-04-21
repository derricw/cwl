package mlflow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/derricw/cwl/provider"
)

// TestFetchGroupsParsesExperiments verifies that the search experiments
// response is correctly parsed into provider.LogGroup values.
func TestFetchGroupsParsesExperiments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(searchExperimentsResp{
			Experiments: []experiment{
				{ExperimentID: "1", Name: "my-experiment"},
				{ExperimentID: "2", Name: "another-exp"},
			},
		})
	}))
	defer srv.Close()

	b := New(srv.URL)
	groups, err := b.FetchGroups("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "my-experiment" {
		t.Fatalf("expected 'my-experiment', got %q", groups[0].Name)
	}
	if groups[1].Desc != "experiment_id=2" {
		t.Fatalf("expected 'experiment_id=2', got %q", groups[1].Desc)
	}
}

// TestFetchGroupsWithPattern verifies that the filter parameter is passed
// to the API when a pattern is provided.
func TestFetchGroupsWithPattern(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(searchExperimentsResp{})
	}))
	defer srv.Close()

	b := New(srv.URL)
	b.FetchGroups("test")

	filter, ok := receivedBody["filter"].(string)
	if !ok || filter != "name ILIKE '%test%'" {
		t.Fatalf("expected ILIKE filter, got %q", filter)
	}
}

// TestRetryOn429 verifies that requests are retried on 429 Too Many Requests
// and eventually succeed when the server recovers.
func TestRetryOn429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(searchExperimentsResp{
			Experiments: []experiment{{ExperimentID: "1", Name: "ok"}},
		})
	}))
	defer srv.Close()

	b := New(srv.URL)
	b.retryDelay = 0
	groups, err := b.FetchGroups("")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "ok" {
		t.Fatalf("unexpected groups: %v", groups)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls (2 retries + 1 success), got %d", calls.Load())
	}
}

// TestMaxRetriesExhausted verifies that after max retries on 429, the
// request fails with an error.
func TestMaxRetriesExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	b := New(srv.URL)
	b.retryDelay = 0
	_, err := b.FetchGroups("")
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if err.Error() != "mlflow API error: too many retries" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGetMetricKeys verifies that metric keys are extracted from a run response.
func TestGetMetricKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(getRunResp{
			Run: run{
				Info: runInfo{RunID: "abc"},
				Data: runData{
					Metrics: []metric{
						{Key: "loss", Value: 0.5},
						{Key: "accuracy", Value: 0.9},
						{Key: "lr", Value: 0.001},
					},
				},
			},
		})
	}))
	defer srv.Close()

	b := New(srv.URL)
	keys, err := b.getMetricKeys("abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	expected := map[string]bool{"loss": true, "accuracy": true, "lr": true}
	for _, k := range keys {
		if !expected[k] {
			t.Fatalf("unexpected key: %s", k)
		}
	}
}

// TestFetchMetricHistory verifies that metric history is parsed into
// provider.LogEvent values with correct message format.
func TestFetchMetricHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(metricHistoryResp{
			Metrics: []metric{
				{Key: "loss", Value: 0.5, Timestamp: 1000, Step: 1},
				{Key: "loss", Value: 0.3, Timestamp: 2000, Step: 2},
			},
		})
	}))
	defer srv.Close()

	b := New(srv.URL)
	var events []provider.LogEvent
	err := b.fetchMetricHistory("abc", "loss", func(batch []provider.LogEvent) error {
		events = append(events, batch...)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Message != "step=1 loss=0.5" {
		t.Fatalf("unexpected message: %q", events[0].Message)
	}
	if events[1].Message != "step=2 loss=0.3" {
		t.Fatalf("unexpected message: %q", events[1].Message)
	}
}

// TestParseARN verifies that NewFromSageMakerARN rejects invalid ARNs
// without making any AWS calls.
func TestParseARNInvalid(t *testing.T) {
	_, err := NewFromSageMakerARN("not-an-arn", "")
	if err == nil {
		t.Fatal("expected error for invalid ARN")
	}

	_, err = NewFromSageMakerARN("arn:aws:sagemaker:us-west-2:123:bad-resource", "")
	if err == nil {
		t.Fatal("expected error for invalid resource format")
	}
}
