# Context Groups

Context Groups let you organize kubeconfig contexts into named sets such as `prod`, `staging`, or `eu-clusters`.

Groups are stored in `~/.kubecfg/groups.yaml` and do not modify your kubeconfig until you explicitly run `kubecfg group use <name>`.

## Command Reference

### Create a Group

```bash
kubecfg group create prod \
  --contexts eks-prod-us-east-1,gke-prod-eu-west-1 \
  --description "All production clusters" \
  --color red
```

Flags:

- `--contexts` required comma-separated or repeated list of context names
- `--description` optional human-readable description
- `--color` optional hint: `red`, `yellow`, `green`, `blue`, `cyan`, `magenta`

### List Groups

```bash
kubecfg group list
kubecfg group list --wide
```

`kubecfg group list` shows the group name, member count, and description.

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

### Use a Group

```bash
kubecfg group use prod
```

Behavior:

- If the group has one valid context, `kubecfg` switches to it directly.
- If the group has multiple valid contexts and `fzf` is installed, `kubecfg` opens an `fzf` picker scoped to that group.
- If `fzf` is unavailable, `kubecfg` falls back to the built-in interactive selector.
- Stale members are warned about before selection so you can clean them up without losing the rest of the group.

## Example Workflow

For a team with `prod`, `staging`, and `dev` environments:

```bash
kubecfg group create prod --contexts eks-prod,gke-prod --color red
kubecfg group create staging --contexts eks-staging,gke-staging --color yellow
kubecfg group create dev --contexts kind-dev,minikube --color green

kubecfg group list
kubecfg group use prod
kubecfg group show staging
```

This keeps environment-level navigation separate from the individual kubeconfig context names.

## Guard Profiles

Context Groups and guard profiles are independent today.

You can use groups to choose a target context quickly, then start a guard session with the existing guard commands for that context. A future workflow may bind group selection more directly to guard profiles, but that is not implemented yet.

## Edit groups.yaml Manually

`~/.kubecfg/groups.yaml` is plain YAML and can be hand-edited.

Example:

```yaml
groups:
  - name: prod
    description: All production clusters
    color: red
    contexts:
      - eks-prod-us-east-1
      - gke-prod-eu-west-1
  - name: staging
    description: Staging environments
    color: yellow
    contexts:
      - eks-staging
      - gke-staging
```

Rules to keep in mind:

- Group names must be lowercase alphanumeric with optional internal hyphens.
- Every group must contain at least one context name.
- Context names are validated against your current kubeconfig when you create, update, or use a group.
- If a context is later removed from kubeconfig, `kubecfg group show` and `kubecfg group use` warn instead of crashing.