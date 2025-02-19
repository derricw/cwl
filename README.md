# cwl

Simple CLI and TUI for browsing Cloudwatch logs in the terminal.

Aims to be as simple as possible but no simpler.

< demo video soon >

### Install

```bash
go install github.com/derricw/cwl@latest
```

### Use

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

### Using in a pipeline

Each command can read input from stdin, so you can compose with other tools like `fzf` or `grep`:
```bash
cwl streams /aws/batch/job | fzf | cwl events
```

Start searching many log streams for a keyword and dump to file:
```bash
cwl streams /aws/batch/job | cwl events | grep "ERROR" > errors.log
```
