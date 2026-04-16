# Health Check

`kubecfg status` checks whether the API server behind each kubeconfig context is reachable and how quickly it responds.

## Commands

Check every context in the kubeconfig:

```bash
kubecfg status
```

Check a single context:

```bash
kubecfg status production-eks
kubecfg status --context production-eks
```

Watch mode re-runs the checks every 10 seconds:

```bash
kubecfg status --watch
```

Structured output is available for scripts:

```bash
kubecfg status --output json
kubecfg status --output yaml
```

## Flags

- `--context string`: Check one context instead of all contexts.
- `--watch`: Re-run checks every 10 seconds until interrupted.
- `--timeout duration`: Override the per-check timeout. The default is `4s`.
- `--output string`: Output format: `table`, `json`, or `yaml`.

## Latency Thresholds

- `healthy`: The API server responded with HTTP 200 within `800ms`.
- `degraded`: The API server responded with HTTP 200 slower than `800ms`, or `/readyz` and `/healthz` both returned HTTP errors.
- `unhealthy`: The connection timed out, was refused, or failed TLS validation.
- `unreachable`: DNS resolution failed.

## TUI Behavior

The TUI starts background health checks as soon as it loads contexts. Rendering is not blocked while the checks run.

- `✓` means healthy
- `~` means degraded
- `✗` means unhealthy or unreachable
- `?` means no result has been cached yet

When a refresh is in progress, unknown entries show a spinner. Press `r` in the context list to trigger a fresh check for the currently visible contexts.

## Cache

Health results are cached in memory for 30 seconds.

- Opening the TUI starts a background refresh immediately.
- `kubecfg status` always performs live checks.
- Pressing `r` in the TUI forces a refresh for the visible contexts.
