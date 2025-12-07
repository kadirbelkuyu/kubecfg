# kubecfg

A CLI tool for managing Kubernetes kubeconfig files.

![kubecfg demo](img/kubecfg.gif)

## Installation (Coming soon for package managers)

### Homebrew

```bash
brew install kadirbelkuyu/tap/kubecfg
```

### Krew

```bash
kubectl krew install kubecfg
```

### Go

```bash
go install github.com/kadirbelkuyu/kubecfg@latest
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

## Features

- **Add** - Import kubeconfig files with custom context names
- **List** - View all contexts with cluster details
- **Use** - Interactive context switching
- **Remove** - Delete contexts with confirmation
- **Rename** - Change context names
- **Merge** - Combine multiple kubeconfig files

## Usage

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

## Flags

| Flag | Description |
|------|-------------|
| `--kubeconfig` | Path to kubeconfig file (default: `~/.kube/config`) |

## License

MIT License
