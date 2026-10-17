# kubecfg

[![CI](https://github.com/kadirbelkuyu/kubecfg/actions/workflows/ci.yml/badge.svg)](https://github.com/kadirbelkuyu/kubecfg/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kadirbelkuyu/kubecfg)](https://github.com/kadirbelkuyu/kubecfg)
[![License](https://img.shields.io/github/license/kadirbelkuyu/kubecfg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/kadirbelkuyu/kubecfg)](https://github.com/kadirbelkuyu/kubecfg/releases)

kubecfg is a Kubernetes kubeconfig manager for people who switch contexts often and need more than a thin wrapper around `kubectl config`.

It provides fast context and namespace switching, context groups, cluster health checks, and guarded Kubernetes sessions that can route requests through a local policy proxy.

![kubecfg TUI main menu](img/main.png)

## Key Features

- Interactive TUI for browsing and switching contexts.
- Direct CLI commands for context, namespace, merge, export, validate, and backup restore workflows.
- Optional `fzf` picker integration for context and namespace selection.
- Previous-context toggle with `kubecfg use -`.
- In-place context editing with `kubecfg edit` for renewed certificates, moved API servers, or changed credentials.
- Context groups stored in `~/.kubecfg/groups.yaml`.
- Health checks for all contexts or a single context, with table, JSON, and YAML output.
- Guard sessions with generated kubeconfig files, TTLs, audit logs, policy profiles, and automatic activation from policy-bound groups.
- Shell completions for Bash, Zsh, Fish, and PowerShell.

## Screenshots

### Context Selection

![kubecfg use context selection](img/kubecfg-use.png)

### Context List

![kubecfg list output](img/kubecfg-list.png)

### Policy Profiles

![kubecfg policy list output](img/policy-list.png)

### Guard Commands

![kubecfg guard help output](img/guard.png)

### Help

![kubecfg help output](img/help.png)

## Why kubecfg vs kubectx

`kubectx` is excellent for quick context switching. kubecfg targets a broader kubeconfig workflow:

- Use `kubecfg use -` for kubectx-style previous-context switching.
- Use `kubecfg ns` for namespace switching without writing `kubectl config set-context`.
- Use `kubecfg group` to organize related contexts by environment or team.
- Use `kubecfg status` before switching into stale or unreachable clusters.
- Use `kubecfg guard` and `kubecfg policy` when you want a temporary, auditable access mode around risky clusters.

## Installation

### Go

```bash
go install github.com/kadirbelkuyu/kubecfg@latest
```

For a pinned v0.2.0 install after the release is tagged:

```bash
go install github.com/kadirbelkuyu/kubecfg@v0.2.0
```

### Homebrew

```bash
brew tap kadirbelkuyu/tap
brew install kadirbelkuyu/tap/kubecfg
```

### GitHub Releases

Download the archive for your OS and architecture from:

```text
https://github.com/kadirbelkuyu/kubecfg/releases
```

Release archives include the `kubecfg` binary, `README.md`, `LICENSE`, and SHA-256 checksums.

### From Source

```bash
git clone https://github.com/kadirbelkuyu/kubecfg.git
cd kubecfg
go build -o kubecfg .
./kubecfg --help
```

### Krew

Krew packaging is not configured in this repository for v0.2.0.

## Shell Completion

```bash
# bash
echo 'eval "$(kubecfg completion bash)"' >> ~/.bashrc

# zsh
echo 'source <(kubecfg completion zsh)' >> ~/.zshrc

# fish
kubecfg completion fish | source
```

PowerShell, plugin-manager, and troubleshooting notes are in [docs/shell-completion.md](docs/shell-completion.md).

## Optional fzf Integration

If `fzf` is installed, kubecfg uses it automatically for context and namespace pickers.

```bash
export KUBECFG_IGNORE_FZF=1
```

Set `KUBECFG_IGNORE_FZF` to force the built-in prompt UI.

## Quickstart

These commands assume your kubeconfig is available at `~/.kube/config` or through `KUBECONFIG`.

```bash
kubecfg list
kubecfg current
kubecfg use
kubecfg ns kube-system
kubecfg status
```

Use a specific kubeconfig file with:

```bash
kubecfg --kubeconfig ./config list
```

Persistently choose which kubeconfig file kubecfg should use:

```bash
kubecfg source dir add ~/team-kubeconfigs
kubecfg source list
kubecfg source use ~/team-kubeconfigs/prod.yaml
kubecfg source current
```

The active source is stored in `~/.kubecfg/config.yaml` under `kubeconfig_sources.active`.
Directories scanned for selectable kubeconfig files are stored under `kubeconfig_sources.dirs`.
The TUI also includes a `Kubeconfig Sources` screen for switching the active file without leaving the app.

## Core Commands

### use

Switch context directly, interactively, or back to the previous context.

```bash
kubecfg use
kubecfg use production
kubecfg use -
kubecfg use production --namespace kube-system
kubecfg use production --namespace
```

`--namespace` without a value opens namespace selection after the context is chosen.

### ns

Set or show the namespace for the current context.

```bash
kubecfg ns
kubecfg ns kube-system
kubecfg ns current
```

### status

Check Kubernetes API reachability for one or more contexts.

```bash
kubecfg status
kubecfg status production
kubecfg status --watch
kubecfg status --output json
```

The command exits non-zero when a checked context is unhealthy or unreachable, so it can be used in scripts.

Contexts using an `exec` credential plugin require `kubectl` on `PATH` for health checks and namespace discovery. These requests run in a separate process so credential plugin diagnostics do not corrupt the TUI. While the TUI owns the terminal it also captures standard error, so output from credential plugins such as `gke-gcloud-auth-plugin` cannot be painted over the screen; a context that needs a new sign-in is reported in its health row instead. Set `KUBECFG_DEBUG_STDERR=1` to print the captured output after the TUI exits.

If Google Cloud credentials need reauthentication, leave the TUI, run `gcloud auth login` in a terminal, and refresh health status after signing in.

### edit

Update the definition of an existing context in place, instead of removing it and adding it again.

```bash
kubecfg edit
kubecfg edit production
kubecfg edit production --from ./renewed-prod.yaml
kubecfg edit production --print
```

`kubecfg edit` opens the context entry together with the cluster and user it references as a single-context kubeconfig in an editor, taken from `KUBECFG_EDITOR`, `KUBE_EDITOR`, `VISUAL`, or `EDITOR`. Saving the file applies the document; saving it unchanged or empty cancels the edit. Without a context name the current context is edited.

The names of the context, cluster, and user must stay unchanged; use `kubecfg rename` to rename a context. A rejected document is reopened with the reason at the top, so edits are not lost.

`--from` reads the definition from a kubeconfig file and keeps the local names, so a kubeconfig issued after a certificate rotation can be applied to an existing context:

```bash
kubecfg edit production --from ~/Downloads/prod-renewed.yaml
```

The server, certificates, tokens, and `exec` credential plugin settings are taken from the file. The namespace is taken from the file when it sets one, otherwise the local namespace is kept. The kubeconfig is backed up before it is written, so `kubecfg undo` can restore the previous definition.

`--print` writes the definition to stdout without editing it.

### group

Create named sets of contexts and switch within them.

```bash
kubecfg group create prod --contexts eks-prod,gke-prod --color red
kubecfg group create prod --contexts eks-prod,gke-prod --policy prod
kubecfg group list
kubecfg group show prod
kubecfg group use prod
kubecfg group add prod aks-prod
kubecfg group remove prod old-prod
kubecfg group set-policy prod prod
kubecfg group unset-policy prod
kubecfg group delete prod --force
```

Group data is stored in `~/.kubecfg/groups.yaml`. A group may bind a guard policy with `--policy`; using that group automatically starts Guard with the bound policy. See [docs/context-groups.md](docs/context-groups.md).

### policy

Inspect and validate guard policy profiles.

```bash
kubecfg policy list
kubecfg policy show prod
kubecfg policy init
kubecfg policy validate
kubecfg policy create restricted --from prod
```

Built-in profiles are `prod`, `staging`, and `debug`. User-defined profiles live in `~/.kubecfg/config.yaml`.

## Guard and Policy

Guard sessions create a temporary kubeconfig that points the selected cluster at a local kubecfg proxy. The proxy enforces the selected policy profile and records audit events.

```bash
kubecfg guard start --ttl 30m --profile prod
export KUBECONFIG=/path/from/guard/output
kubectl get pods -A
kubecfg guard status
kubecfg guard stop
```

The `prod` profile is readonly and blocks protected resources such as secrets. `staging` allows more traffic but requires confirmation for destructive operations. `debug` is permissive and records allowed requests for troubleshooting.

Audit events can be inspected with:

```bash
kubecfg audit tail
kubecfg audit tail --limit 20
```

## Health Check

`kubecfg status` checks `/readyz` and `/healthz` using the kubeconfig credentials for each context. Results are classified as healthy, degraded, unhealthy, unreachable, or unknown.

```bash
kubecfg status
kubecfg status production --timeout 2s
kubecfg status --output yaml
```

See [docs/health-check.md](docs/health-check.md) for thresholds, cache behavior, TUI refresh behavior, and output details.

## Example Workflows

### Daily context switching

```bash
kubecfg status
kubecfg group use prod
kubecfg ns payments
kubecfg current
```

### Import and validate a kubeconfig

```bash
kubecfg add ./team-prod.yaml --name team-prod
kubecfg validate
kubecfg status team-prod
```

### Prepare a readonly production session

```bash
kubecfg use production
kubecfg policy show prod
kubecfg guard start --ttl 30m --profile prod
kubecfg audit tail
```

Policy-bound groups combine the switch and guard steps:

```bash
kubecfg group create prod --contexts prod-eu,prod-us --policy prod
kubecfg group use prod
```

If another Guard session is active, the group binding wins. For example, using a `prod` group replaces an active `debug` guard session with `prod`.

### Renew the credentials of a context

```bash
# edit the definition by hand
kubecfg edit production

# or apply the kubeconfig the cluster provider issued
kubecfg edit production --from ~/Downloads/prod-renewed.yaml

kubecfg status production
```

The context keeps its name, its group membership, and its namespace, so nothing that refers to it has to be updated. `kubecfg undo` restores the previous definition from the backup taken before the write.

### Merge kubeconfigs safely

```bash
kubecfg merge east.yaml west.yaml --output merged.yaml --on-conflict rename
kubecfg --kubeconfig ./merged.yaml validate
kubecfg --kubeconfig ./merged.yaml list
```

## Other Commands

```bash
kubecfg add ./cluster.yaml --name dev
kubecfg list --filter prod
kubecfg current --context production
kubecfg search kube-system
kubecfg export production --output production.yaml
kubecfg merge a.yaml b.yaml --output merged.yaml
kubecfg remove old-dev --force
kubecfg rename old-name new-name
kubecfg edit dev --print
kubecfg undo --list
kubecfg version
```

## TUI Keys

| Key | Action |
| --- | --- |
| `Up` / `Down` / `k` / `j` | Navigate |
| `Enter` | Select |
| `/` | Filter |
| `Ctrl+e` | Edit the selected context definition in `$EDITOR` |
| `r` | Refresh guard status in Guard view |
| `[` / `]` | Change guard TTL preset in Guard view |
| `Esc` | Back or cancel |
| `q` | Quit |

## Limitations

- kubecfg manages kubeconfig files; it does not install or configure Kubernetes clusters.
- `fzf` is optional. Without it, kubecfg uses the built-in prompt UI.
- Guard sessions protect traffic that uses the generated guarded kubeconfig. Other kubeconfig files or direct cluster endpoints are outside the guard proxy.
- Krew, APT, Chocolatey, Winget, MacPorts, and AUR packaging are not configured in this repository for v0.2.0.
- Health checks depend on network access and the credentials in the selected kubeconfig.

## Documentation

- [Context Groups](docs/context-groups.md)
- [Health Check](docs/health-check.md)
- [Shell Completion](docs/shell-completion.md)
- [Git Workflow Guide](docs/GIT_WORKFLOW.md)
- [Git Cheat Sheet](docs/GIT_CHEATSHEET.md)

Project docs are published at <https://kadirbelkuyu.github.io/kubecfg/>.

## Development

```bash
go build ./...
go test ./... -race
go vet ./...
golangci-lint run ./...
goreleaser check
```

## License

MIT License
