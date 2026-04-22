# kubecfg

[![CI](https://github.com/kadirbelkuyu/kubecfg/actions/workflows/ci.yml/badge.svg)](https://github.com/kadirbelkuyu/kubecfg/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kadirbelkuyu/kubecfg)](https://github.com/kadirbelkuyu/kubecfg)
[![License](https://img.shields.io/github/license/kadirbelkuyu/kubecfg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/kadirbelkuyu/kubecfg)](https://github.com/kadirbelkuyu/kubecfg/releases)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=kubecfg&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=kubecfg)

A CLI tool for managing Kubernetes kubeconfig files.

- **Fast context switching** – Interactive TUI or direct commands, no more `kubectl config use-context`
- **Namespace management** – Switch namespaces without remembering long `kubectl` flags
- **Multi-config merge** – Combine kubeconfig files from different clusters into one
- **Readonly guard sessions** – Route Kubernetes API traffic through a local proxy and block mutating requests
- **Audit trail** – Every guard session event is logged to disk; inspect it any time with `kubecfg audit tail`

![kubecfg TUI](img/kubecfg.png)

## Installation

### Homebrew

```bash
brew tap kadirbelkuyu/tap
brew install kadirbelkuyu/tap/kubecfg
```

### Go

```bash
go install github.com/kadirbelkuyu/kubecfg@latest
```

## Shell Completion

```bash
# bash
echo 'eval "$(kubecfg completion bash)"' >> ~/.bashrc

# zsh
echo 'source <(kubecfg completion zsh)' >> ~/.zshrc

# fish
kubecfg completion fish | source
```

Full setup notes for Bash, Zsh, Fish, PowerShell, oh-my-zsh, zplug, and fisher live in [docs/shell-completion.md](docs/shell-completion.md).

## Documentation

The project docs are published with GitHub Pages at <https://kadirbelkuyu.github.io/kubecfg/>.

## fzf Integration

Install `fzf` and `kubecfg` will use it automatically for context and namespace selection. To opt out:

```bash
export KUBECFG_IGNORE_FZF=1
```

## Quick Toggle

```bash
kubecfg use -    # switch to previous context (like kubectx -)
```

### From Source

```bash
git clone https://github.com/kadirbelkuyu/kubecfg.git
cd kubecfg
go build -o kubecfg .
sudo mv kubecfg /usr/local/bin/
```

<details>
<summary><strong>More installation options</strong></summary>

### Arch Linux (AUR)

```bash
yay -S kubecfg
```

### MacPorts

```bash
sudo port install kubecfg
```

### Chocolatey (Windows)

```powershell
choco install kubecfg
```

### Krew

```bash
kubectl krew install kubecfg
```

### Winget (Windows)

```powershell
winget install kadirbelkuyu.kubecfg
```

### APT (Debian/Ubuntu)

```bash
curl -fsSL https://kadirbelkuyu.github.io/apt-repo/public_key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/kubecfg-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/kubecfg-keyring.gpg] https://kadirbelkuyu.github.io/apt-repo stable main" | sudo tee /etc/apt/sources.list.d/kubecfg.list
sudo apt update && sudo apt install kubecfg
```

</details>

## Quickstart

Switch context interactively:

```bash
kubecfg use
```

Set namespace for current context:

```bash
kubecfg ns kube-system
```

Show current context info:

```bash
kubecfg current
```

Start a readonly guard session:

```bash
kubecfg guard start --ttl 30m
```

- **Add** - Import kubeconfig files with custom context names
- **List** - View all contexts with cluster details
- **Use** - Interactive context switching with optional namespace selection
- **Namespace** - Switch namespaces with interactive picker
- **Remove** - Delete contexts with confirmation
- **Rename** - Change context names
- **Merge** - Combine multiple kubeconfig files
- **Guard** - Start temporary readonly sessions with TTL, generated kubeconfig, and local session state
- **Audit** - Inspect guard session history with `kubecfg audit tail`

## Context Groups

Organize contexts into named groups for faster navigation across environments.

```bash
# Create a group
kubecfg group create prod --contexts eks-prod,gke-prod --color red

# Switch within a group (interactive when multiple contexts exist)
kubecfg group use prod

# List all groups
kubecfg group list

# See group details
kubecfg group show prod
```

See [docs/context-groups.md](docs/context-groups.md) for the full command reference and the `~/.kubecfg/groups.yaml` format.

## Health Check

Check reachability of all your clusters at once:

```bash
kubecfg status
```

Check a single cluster with detailed output:

```bash
kubecfg status production-eks
```

Watch mode re-runs checks every 10 seconds:

```bash
kubecfg status --watch
```

Use it in scripts:

```bash
kubecfg status || alert "cluster unreachable"
```

See [docs/health-check.md](docs/health-check.md) for flags, latency thresholds, TUI refresh behavior, and cache details.

## TUI Keys

| Key | Action |
|-----|--------|
| `↑` / `↓` / `k` / `j` | Navigate list |
| `Enter` | Select item |
| `/` | Start filtering |
| `r` | Refresh guard status in Guard view |
| `[` / `]` | Change guard TTL preset in Guard view |
| `Esc` | Cancel / Go back |
| `q` | Quit |

## Project Structure

```
kubecfg/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go             # Root command and global flags
│   ├── add.go              # Add kubeconfig
│   ├── use.go              # Switch context
│   ├── ns.go               # Namespace management
│   ├── list.go             # List contexts
│   ├── remove.go           # Remove context
│   ├── rename.go           # Rename context
│   ├── merge.go            # Merge configs
│   ├── current.go          # Show current context
│   ├── guard.go            # Guard session commands
│   └── audit.go            # Audit log commands
├── internal/
│   ├── application/        # Business logic (Service layer)
│   ├── domain/             # Domain models (KubeConfig, Context, Cluster)
│   ├── infrastructure/     # Repository implementations
│   ├── tui/                # Terminal UI (Bubble Tea)
│   ├── ui/                 # Prompt UI components
│   └── config/             # Configuration management
└── main.go                 # Entry point
```

**Architecture**: Clean Architecture with separation of concerns. Domain layer defines kubeconfig models, Application layer handles business logic, Infrastructure layer manages file I/O, and TUI/CLI layers provide user interfaces.

## Usage

![kubecfg demo](img/kubecfg2.gif)

### Add a Cluster

```bash
kubecfg add ./eks-cluster.yaml --name production-eks
```

### List Contexts

```bash
kubecfg list
```

### Switch Context

Interactive mode:

```bash
kubecfg use
```

Direct switch:

```bash
kubecfg use production-eks
```

With interactive namespace selection:

```bash
kubecfg use -n
kubecfg use production-eks -n
```

With specific namespace:

```bash
kubecfg use production-eks -n kube-system
```

### Switch Namespace

Interactive mode:

```bash
kubecfg ns
```

Direct switch:

```bash
kubecfg ns kube-system
```

Show current namespace:

```bash
kubecfg ns current
```

### Show Current Context

```bash
kubecfg current
```

### Rename Context

```bash
kubecfg rename old-name new-name
```

### Remove Context

```bash
kubecfg remove old-cluster
kubecfg remove old-cluster --force  # skip confirmation
```

### Merge Configs

```bash
kubecfg merge config1.yaml config2.yaml -o merged.yaml
```

### Start a Guard Session

Start a readonly guard session with a 30 minute TTL:

```bash
kubecfg guard start --ttl 30m
```

Example output includes:

- Session ID
- Proxy address
- Generated kubeconfig path
- Expiration time
- `export KUBECONFIG=...` example

Use the generated kubeconfig for readonly access:

```bash
export KUBECONFIG=/path/to/generated/config
kubectl get pods -A
```

Mutating requests are blocked by the local proxy:

```bash
kubectl delete pod my-pod
```

Blocked request example:

```text
guard readonly mode blocked mutating request: DELETE /api/v1/namespaces/default/pods/my-pod
```

### Show Guard Status

```bash
kubecfg guard status
```

Status output includes the current session id, readonly mode, context, namespace, proxy address, generated kubeconfig path, remaining TTL, and the most recent audit events for that session.

### Stop a Guard Session

```bash
kubecfg guard stop
```

Stopping a session shuts down the local proxy and removes temporary guard artifacts.

### View Audit Events

Every guard session lifecycle event — start, stop, expiry, and blocked requests — is appended to an audit log at `~/.kubecfg/guard/audit.log`.

Show the most recent events:

```bash
kubecfg audit tail
```

Limit the number of results:

```bash
kubecfg audit tail --limit 20
```

Each entry includes a timestamp, event type, session ID, context, namespace, and a short message. The log persists across sessions so you can review the history even after a session has ended.

### Guard Session State

kubecfg stores the active guard session on disk:

```text
~/.kubecfg/session.json
```

The generated guarded kubeconfig is temporary and points the selected cluster server to the local reverse proxy while preserving the original authentication settings.

## Flags

| Flag | Description |
|------|-------------|
| `--kubeconfig` | Path to kubeconfig file (default: `~/.kube/config`) |
| `-n, --namespace` | Set namespace for context (use without value for interactive selection) |
| `--ttl` | Guard session TTL for `kubecfg guard start` (example: `30m`, `1h`) |
| `--limit` | Number of recent events to show for `kubecfg audit tail` (default: `10`) |

## Development

### Documentation

- [Git Workflow Guide](docs/GIT_WORKFLOW.md) - Comprehensive Git workflow and release process
- [Git Cheat Sheet](docs/GIT_CHEATSHEET.md) - Quick reference for common Git operations

### Contributing

1. Use [Conventional Commits](https://www.conventionalcommits.org/)
2. Create feature branches: `feature/your-feature-name`
3. Use `--no-ff` merge to main
4. Follow [Semantic Versioning](https://semver.org/)

## License

MIT License
