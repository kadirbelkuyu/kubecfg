---
layout: home
title: kubecfg
---

# kubecfg

kubecfg is a Kubernetes kubeconfig manager with fast context switching, namespace management, context groups, health checks, guarded sessions, and policy profiles.

## Install

```bash
brew tap kadirbelkuyu/tap
brew install kadirbelkuyu/tap/kubecfg
```

```bash
go install github.com/kadirbelkuyu/kubecfg@latest
```

## Quickstart

```bash
kubecfg use
kubecfg ns kube-system
kubecfg current
kubecfg status
kubecfg guard start --ttl 30m
```

## Core Commands

- `kubecfg use` switches contexts directly or interactively.
- `kubecfg ns` sets or shows the namespace for the current context.
- `kubecfg status` checks Kubernetes API reachability.
- `kubecfg group` manages named context groups.
- `kubecfg policy` inspects and validates guard policy profiles.
- `kubecfg guard` starts, inspects, and stops guarded sessions.

## Documentation

- [Shell completion](shell-completion.md)
- [Context groups](context-groups.md)
- [Health check](health-check.md)
- [GitHub repository](https://github.com/kadirbelkuyu/kubecfg)
