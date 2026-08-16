# ADR-014: CLI Uses Cobra

## Status

Accepted.

## Context

Gombit's CLI is a first-class product surface (build plan M2–M4): nested
command families (`gombit db …`, `gombit make …`, `gombit client …`), shared
flags, discoverable help, and Django-style management-command extensibility
(M4-7) where a feature-package registers custom `gombit <command>`s at
runtime.

The pre-M4 `cmd/gombit` implementation uses stdlib `flag` with a hand-rolled
`db` router. That is enough for the M2 migration subset. The full M4 tree and
M4-7 registration model need a command framework.

Two common Go options were considered:

- **Cobra (`spf13/cobra`)** — imperative command tree, `AddCommand`, persistent
  flags, help, and shell completions. De-facto standard for nested CLIs
  (`gh`, kubectl, Hugo).
- **Kong** — struct-tag wiring with less boilerplate for a closed, compile-time
  command set. Dynamic registration of app-owned commands is awkward.

## Decision

Gombit will use **Cobra** as the CLI framework (build plan **D13**).

M4-1 adopts Cobra for the root `gombit` command tree and migrates existing
`db` subcommands onto it. Later M3/M4 commands (`openapi`, `client`, `new`,
`dev`, `make`, introspection, `createsuperuser`) register as Cobra commands
on that tree.

M4-7 management-command extensibility must register custom commands through
Cobra's `AddCommand` (or a thin Gombit helper that wraps it). Do not invent a
second command router beside Cobra.

Kong and other struct-tag CLIs are rejected for Gombit because M4-7 requires
runtime registration from feature packages.

Pre-M4 code may keep stdlib `flag` until M4-1 lands; do not expand the
hand-rolled router as the long-term CLI architecture.

## Consequences

- M4-1 is the adoption point: Cobra dependency, root command, migration of
  `gombit db …`, and a stable place for later commands to attach.
- M4-7 scaffolds and docs must show Cobra-based registration, not a custom
  dispatcher.
- Shell completions and consistent `--help` come from Cobra; keep user-facing
  command names stable when moving off stdlib `flag`.
- Agents must not re-litigate Cobra vs Kong; D13 is locked.

## References

- Build plan D13 and M4 backlog (`docs/GOMBIT_BUILD_PLAN.md`)
- Design doc §26 (CLI Architecture) — command families; library choice is here
- Issue #21: `[M4-1] gombit new` (Cobra adoption + migrate `db` onto the tree)
- Issue #27: `[M4-7] Management-command extensibility (gombit make command)`
