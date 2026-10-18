# SIS Project - Checkpoint 1: Text Analyzer

## Overview
This application is an in-memory synchronous Go task processing system designed for text analysis.

## Task Lifecycle
- **Pending**: Task has been created and input has been validated.
- **Processing**: Task is currently being evaluated.
- **Completed**: Task processing finished successfully and results are stored.
- **Failed**: Task failed during processing or validation.

### Allowed Transitions:
- `Pending` -> `Processing` -> `Completed`
- `Pending` / `Processing` -> `Failed`

*Note: Re-processing a task that is already `Completed` is prohibited and returns an error.*

## Running Verification Commands
```bash
gofmt -l .
go vet ./...
go test -v ./...