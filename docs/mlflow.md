# MLflow Backend

cwl can browse MLflow experiments and runs using the same TUI as CloudWatch. Experiments map to log groups, runs map to streams, and metric history maps to events.

## Setup

### Self-hosted MLflow

Point cwl at your MLflow tracking server:

```bash
cwl --mlflow-url http://localhost:5000
```

### SageMaker-managed MLflow

Provide the tracking server ARN (uses your AWS credentials to generate a presigned URL):

```bash
cwl --mlflow-arn arn:aws:sagemaker:us-west-2:123456789012:mlflow-tracking-server/my-server
```

This works with any AWS profile:

```bash
cwl --mlflow-arn $ARN -p my-profile
```

### Environment variable

Set `MLFLOW_TRACKING_URI` to avoid passing flags every time. cwl auto-detects whether it's a URL or SageMaker ARN:

```bash
# Direct URL
export MLFLOW_TRACKING_URI=http://localhost:5000

# SageMaker ARN
export MLFLOW_TRACKING_URI=arn:aws:sagemaker:us-west-2:123456789012:mlflow-tracking-server/my-server
```

Then just run:

```bash
cwl
```

## Usage

Once connected, the TUI works identically to CloudWatch mode:

- **Experiments** are shown in the groups view
- **Runs** are shown in the streams view (ordered by start time)
- **Metrics** are shown in the events view (all metric keys for the selected run)

All TUI features work: filtering (`/`), preview pane (`p`), save to disk (`s`), jump to group (`-g`), stream filter (`-s`).

### Jump to an experiment

```bash
cwl --mlflow-url http://localhost:5000 -g my-experiment
```

### Filter runs

```bash
cwl --mlflow-url http://localhost:5000 -g my-experiment -s "lr-sweep"
```

## How it maps

| cwl concept | CloudWatch | MLflow |
|-------------|-----------|--------|
| Group | Log Group | Experiment |
| Stream | Log Stream | Run (by name or ID) |
| Event | Log Event | Metric data point (`step=N key=value`) |
| Preview | Last 20 events | Last 20 metric data points |
| Polling | 5s interval | Disabled (metrics are immutable) |

## SageMaker auth details

When using `--mlflow-arn`, cwl:

1. Calls `CreatePresignedMlflowTrackingServerUrl` via the SageMaker API
2. Authenticates with the presigned URL to establish a session
3. Automatically re-authenticates on 403 (session expiry)
4. Retries with backoff on 429 (rate limiting)

The presigned URL expires after ~5 minutes, but the session cookie lasts longer. cwl handles re-auth transparently — you won't see interruptions during normal browsing.

## Limitations

- **Metrics only**: The events view shows metric history logged via `mlflow.log_metric()`. Metrics stored as JSON artifacts (e.g. auxiliary timing metrics) are not currently fetched.
- **No live polling**: MLflow metrics are immutable once logged, so the events view doesn't poll for new data.
- **CLI subcommands**: The `events`, `streams`, `groups`, `query`, and `put` CLI subcommands are CloudWatch-specific. The MLflow backend is TUI-only for now.
