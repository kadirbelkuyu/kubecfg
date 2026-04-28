# Context Groups

Context Groups let you organize kubeconfig contexts into named sets such as `prod`, `staging`, or `eu-clusters`.

Groups are stored in `~/.kubecfg/groups.yaml` and do not modify your kubeconfig until you explicitly run `kubecfg group use <name>`.

## Command Reference

### Create a Group

```bash
kubecfg group create prod \
  --contexts eks-prod-us-east-1,gke-prod-eu-west-1 \
  --description "All production clusters" \
  --color red \
  --policy prod
```

Flags:

- `--contexts` required comma-separated or repeated list of context names
- `--description` optional human-readable description
- `--color` optional hint: `red`, `yellow`, `green`, `blue`, `cyan`, `magenta`
- `--policy` optional guard policy profile to activate when the group is used

### List Groups

```bash
kubecfg group list
kubecfg group list --wide
```

`kubecfg group list` shows the group name, member count, bound policy, and description.

`kubecfg group list --wide` also prints the member context names.

### Show One Group

```bash
kubecfg group show prod
```

This prints the group metadata and checks whether each stored context name still exists in the current kubeconfig.

### Add and Remove Members

```bash
kubecfg group add prod eks-prod-ca-central-1
kubecfg group remove prod eks-prod-ca-central-1
```

Removing the last context is refused. Delete the group entirely if you no longer need it.

### Delete a Group

```bash
kubecfg group delete prod --force
```

Deleting a group does not change your kubeconfig.

### Rename a Group

```bash
kubecfg group rename prod production
```

### Bind or Remove a Policy

```bash
kubecfg group set-policy prod prod
kubecfg group unset-policy prod
```

Policy names are validated before they are saved. Built-in profiles are `prod`, `staging`, and `debug`; user-defined profiles from `~/.kubecfg/config.yaml` are also supported.

### Use a Group

```bash
kubecfg group use prod
```

Behavior:

- If the group has one valid context, `kubecfg` switches to it directly.
- If the group has multiple valid contexts and `fzf` is installed, `kubecfg` opens an `fzf` picker scoped to that group.
- If `fzf` is unavailable, `kubecfg` falls back to the built-in interactive selector.
- Stale members are warned about before selection so you can clean them up without losing the rest of the group.
- If the group has a policy binding, Guard is started with that policy before the current context is changed.
- If another Guard session is already active, the group-bound policy wins and replaces the existing session.

The safe order is: validate the group, validate the policy, choose a valid context, start or replace Guard for that target context, then switch the kubeconfig current context. If Guard cannot be activated, `kubecfg` does not switch into the policy-bound group.

## Example Workflow

For a team with `prod`, `staging`, and `dev` environments:

```bash
kubecfg group create prod --contexts eks-prod,gke-prod --color red --policy prod
kubecfg group create staging --contexts eks-staging,gke-staging --color yellow --policy staging
kubecfg group create debug --contexts kind-dev,minikube --color green --policy debug

kubecfg group list
kubecfg group use prod
kubecfg group show staging
```

This keeps environment-level navigation separate from the individual kubeconfig context names.

## Guard Profiles

Context Groups can bind to guard profiles. A policy-bound group makes the operational intent explicit in `groups.yaml`, and `kubecfg group use <name>` automatically enables Guard for the selected context.

Example:

```bash
kubecfg group create prod --contexts prod-eu,prod-us --policy prod
kubecfg group use prod
```

Expected output includes both the context switch and Guard activation:

```text
Switched to context "prod-eu" (group: prod)
Guard activated with policy prod
```

If an active `debug` Guard session exists and the `prod` group requires `prod`, the group-bound policy replaces it:

```text
Guard policy changed from debug to prod because group prod requires it
```

## Edit groups.yaml Manually

`~/.kubecfg/groups.yaml` is plain YAML and can be hand-edited.

Example:

```yaml
groups:
  - name: prod
    description: All production clusters
    color: red
    policy: prod
    contexts:
      - eks-prod-us-east-1
      - gke-prod-eu-west-1
  - name: staging
    description: Staging environments
    color: yellow
    policy: staging
    contexts:
      - eks-staging
      - gke-staging
```

Rules to keep in mind:

- Group names must be lowercase alphanumeric with optional internal hyphens.
- Every group must contain at least one context name.
- Policy is optional. Empty or omitted policy means no automatic Guard activation.
- A non-empty policy must match an existing built-in or user-defined policy profile.
- Context names are validated against your current kubeconfig when you create, update, or use a group.
- If a context is later removed from kubeconfig, `kubecfg group show` and `kubecfg group use` warn instead of crashing.
