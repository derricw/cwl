---
name: cwl-usage
description: User guide for cwl, a CLI/TUI for browsing and filtering AWS CloudWatch Logs. Use when the user wants to find, view, filter, tail, query, or save CloudWatch logs.
---

# cwl — CloudWatch Logs Viewer

A fast CLI and TUI for browsing AWS CloudWatch Logs in the terminal.

## Setup

Install:
```bash
go install github.com/derricw/cwl@latest
```

Uses the standard AWS credential chain. Specify a profile with `-p`:
```bash
cwl -p my-profile
```

## Interactive TUI

Launch with no subcommand to browse visually through groups → streams → events:

```bash
cwl
```

### Jump directly to a log group's streams:
```bash
cwl -g /aws/lambda/my-function
```

### Jump to streams with a filter pre-applied:
```bash
cwl -g /aws/batch/job -s "2025/04"
```

### TUI Keybindings

| Key | Action |
|-----|--------|
| `enter` | Select group or stream |
| `esc` | Go back (or quit if launched with `-g`) |
| `ctrl+c` | Quit |
| `/` | Filter (type to search in groups, streams, or events) |
| `p` | Toggle stream preview pane |
| `t` | Toggle timestamps on events |
| `w` | Toggle line wrap on events |
| `s` | Save all loaded events to `~/Downloads/cwl/logs/<timestamp>.log` |
| `home`/`end` | Jump to top/bottom of events |

## CLI Commands

### List log groups
```bash
cwl groups
cwl groups --json
```

### List streams for a group
```bash
cwl streams /aws/batch/job
cwl streams /aws/batch/job --prefix "2025/04/"    # filter by prefix
cwl streams /aws/batch/job -f                      # keep watching for new streams
```

### View events from a stream
```bash
# By ARN
cwl events arn:aws:logs:us-west-2:123456789012:log-group:/my/group:log-stream:my-stream

# By group + stream name
cwl events --group /my/group --stream my-stream

# Tail a stream (like tail -f)
cwl events -f --group /my/group --stream my-stream

# Limit output to first 5000 events (useful for very large streams)
cwl events --limit 5000 --group /my/group --stream my-stream

# Follow all streams matching a prefix
cwl events -f --group /my/group --follow-prefix "2025/04/"

# JSON output
cwl events --json --group /my/group --stream my-stream
```

### Query logs (CloudWatch Logs Insights)
```bash
# Query a specific group
cwl query /aws/batch/job -q "fields @timestamp, @message | sort @timestamp desc | limit 100"

# Query all groups
cwl query -q "fields @timestamp, @message | filter @message like /ERROR/ | limit 50"

# With time range
cwl query /my/group -q "fields @message" -s $(date -d "2 hours ago" +%s) -e $(date +%s)
```

### Write events to a stream
```bash
cwl put arn:aws:logs:us-west-2:123456789012:log-group:/my/group:log-stream:my-stream "my message"
echo "hello" | cwl put <stream-arn>
```

## Piping & Composition

Commands read from stdin, so they compose with each other and standard tools:

```bash
# Interactive stream picker with fzf
cwl streams /aws/batch/job | fzf | cwl events

# Search for errors across all streams
cwl streams /aws/batch/job | cwl events | grep "ERROR"

# Dump errors to file
cwl streams /aws/batch/job | cwl events | grep "ERROR" > errors.log

# Count events per stream
cwl streams /aws/batch/job | while read arn; do
  echo "$(cwl events "$arn" | wc -l) $arn"
done
```

## Common Recipes

### Find recent errors in a Lambda function
```bash
cwl query /aws/lambda/my-function -q "fields @timestamp, @message | filter @message like /ERROR/ | sort @timestamp desc | limit 20"
```

### Tail a batch job's logs
```bash
cwl events -f --group /aws/batch/job --stream my-job-id
```

### Browse a specific group's streams interactively
```bash
cwl -g /aws/ecs/my-service
```

### Save logs from the TUI
Open a stream in the TUI, then press `s` to save all loaded events to `~/Downloads/cwl/logs/`.

### Export a full stream to a file
```bash
cwl events --group /my/group --stream my-stream > output.log
```

### Limit output for very large streams
```bash
cwl events --limit 10000 --group /my/group --stream my-stream
```
