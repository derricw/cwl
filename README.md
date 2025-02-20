# cwl

Simple CLI and TUI for browsing Cloudwatch logs in the terminal.

Aims to be as simple as possible but no simpler.

![Made with VHS](https://vhs.charm.sh/vhs-88s4i0E7gteUVOXfY4oi3.gif)

## Install

```bash
go install github.com/derricw/cwl@latest
```

## Use

### Basic

List log groups:
```bash
cwl groups
```

List streams for a group:
```bash
cwl streams /my/log/group
```

Write events from a stream to stdout:
```bash
cwl events arn:aws:logs:us-west-2:12345657890:log-group:/aws/batch/job:log-stream:my_batch_job_12345
```
Use a specific AWS profile (otherwise uses default credential chain):
```bash
cwl -p testProfile groups
```

### Using in a pipeline

Each command can read input from stdin, so you can compose with other tools like `fzf` or `grep`:
```bash
cwl streams /aws/batch/job | fzf | cwl events
```

Start searching many log streams for a keyword and dump to file:
```bash
cwl streams /aws/batch/job | cwl events | grep "ERROR" > errors.log
```

### TUI

Use with no arguments to start a TUI:
```bash
cwl
```
