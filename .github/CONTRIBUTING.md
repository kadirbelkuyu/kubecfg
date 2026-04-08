# Contributing to kubecfg

Thanks for contributing to `kubecfg`.

## Before You Start

- search existing issues and pull requests before opening a new one
- keep changes focused and easy to review
- open an issue first for large features or behavioral changes

## Development Setup

```bash
git clone https://github.com/kadirbelkuyu/kubecfg.git
cd kubecfg
go mod tidy
go build -o kubecfg .
```

## Recommended Workflow

1. Create a feature or fix branch from `main`.
2. Make small, focused commits.
3. Run validation locally before opening a pull request.
4. Open a pull request with a clear summary and testing notes.

## Validation

Run these commands before submitting a pull request:

```bash
gofmt -w .
go test ./...
go vet ./...
golangci-lint run
```

If `golangci-lint` is not installed locally, use:

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run
```

## Commit Style

This repository follows Conventional Commits.

Examples:

- `feat: add readonly guard foundation`
- `fix: harden guard proxy process launch`
- `docs: update guard usage examples`

## Pull Requests

Please include:

- a short summary of the change
- why the change is needed
- validation you performed
- screenshots or terminal output when the UI or UX changed

Try to keep pull requests reviewable. Small, isolated changes are easier to merge safely.

## Architecture Notes

This project follows a layered structure:

- `cmd/` for Cobra commands
- `internal/application/` for business logic
- `internal/domain/` for core models and contracts
- `internal/infrastructure/` for implementations and integrations
- `internal/tui/` for Bubble Tea UI

Please preserve that separation when adding new functionality.
