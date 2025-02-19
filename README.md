# cwl

Browse Cloudwatch logs in the terminal.

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

Write events to stdout:
```bash
cwl events /my/log/group::my/log/stream
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
