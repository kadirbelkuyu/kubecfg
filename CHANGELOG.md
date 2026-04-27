# Changelog

## v0.2.0

### Added

- feat: add UX parity improvements for context switching, namespace selection, shell completion, `fzf` integration, and previous-context toggle.
- feat: add context groups for organizing and switching among named sets of kubeconfig contexts.
- feat: add health checks for Kubernetes contexts, including table, JSON, YAML, watch mode, latency classification, and TUI health indicators.
- feat: add guard sessions with generated kubeconfig files, local policy proxy, TTL handling, audit events, and session status.
- feat: add policy profiles for guard sessions, including built-in `prod`, `staging`, and `debug` profiles plus user-defined profile validation and scaffolding.

### Changed

- docs: rewrite README for v0.2.0 installation, quickstart, core commands, guard, policy, health checks, and release limitations.
- docs: standardize CLI help text, examples, and flag descriptions across public commands.
- chore: validate GoReleaser configuration for multi-platform archives, checksums, generated changelog release notes, and Homebrew publishing.

### Fixed

- fix: ensure release validation passes for build, race-enabled tests, vet, and golangci-lint.
- fix: clarify unsupported package channels in documentation to avoid broken install paths.
