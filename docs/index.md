---
layout: home
title: kubecfg
---

# kubecfg

A CLI tool for managing Kubernetes kubeconfig files with fast context switching, namespace management, multi-config merge, readonly guard sessions, and cluster health checks.

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

## Documentation

- [Shell completion](shell-completion.md)
- [Context groups](context-groups.md)
- [Health check](health-check.md)
- [GitHub repository](https://github.com/kadirbelkuyu/kubecfg)
