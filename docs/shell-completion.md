# Shell Completion

`kubecfg completion` prints shell-native completion scripts for Bash, Zsh, Fish, and PowerShell.

## Bash

Load it for the current shell:

```bash
eval "$(kubecfg completion bash)"
```

Persist it:

```bash
echo 'eval "$(kubecfg completion bash)"' >> ~/.bashrc
source ~/.bashrc
```

## Zsh

Load it for the current shell:

```zsh
source <(kubecfg completion zsh)
```

Persist it in a standard setup:

```zsh
echo 'source <(kubecfg completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

### oh-my-zsh

Write the completion file into `oh-my-zsh`'s custom completions path:

```zsh
mkdir -p ~/.oh-my-zsh/custom/completions
kubecfg completion zsh > ~/.oh-my-zsh/custom/completions/_kubecfg
```

If needed, make sure `fpath` includes the directory:

```zsh
fpath=(~/.oh-my-zsh/custom/completions $fpath)
autoload -Uz compinit && compinit
```

### zplug

Store the generated file anywhere in your `fpath`, for example:

```zsh
mkdir -p ~/.zsh/completions
kubecfg completion zsh > ~/.zsh/completions/_kubecfg
```

Then add this to `~/.zshrc`:

```zsh
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit
```

## Fish

Load it for the current shell:

```fish
kubecfg completion fish | source
```

Persist it:

```fish
mkdir -p ~/.config/fish/completions
kubecfg completion fish > ~/.config/fish/completions/kubecfg.fish
```

### fisher

`fisher` still reads completions from Fish's standard completions directory, so the same install path works:

```fish
mkdir -p ~/.config/fish/completions
kubecfg completion fish > ~/.config/fish/completions/kubecfg.fish
```

## PowerShell

Load it for the current session:

```powershell
kubecfg completion powershell | Out-String | Invoke-Expression
```

Persist it in your profile:

```powershell
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $PROFILE) | Out-Null
'kubecfg completion powershell | Out-String | Invoke-Expression' | Add-Content $PROFILE
. $PROFILE
```
