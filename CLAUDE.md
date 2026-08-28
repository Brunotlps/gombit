@AGENTS.md

## Claude Code

- Project skill available: `code-review` (`.claude/skills/code-review/SKILL.md`)
  — adversarial review of a diff/PR against the Agent Working Agreement in
  docs/GOMBIT_BUILD_PLAN.md §5 and the change's claimed contract. Invoke with
  `/code-review` or ask to review a PR/diff; it overrides the bundled
  `/code-review` for this repo.
- For what has shipped, defer to AGENTS.md "Current state" (kept current
  there) rather than a milestone number duplicated here that drifts out of
  date — as of this writing that is M0–M5, ADMIN-1..3, REL-1..9, and BENCH-1.
  Prefer a direct `Read` of `docs/GOMBIT_BUILD_PLAN.md` over spawning an
  Explore subagent for backlog/dependency lookups. Runtime packages and the
  Cobra `gombit` tree (`new`, `db`, `openapi`, `client`) exist; generated
  apps are written by `gombit new`, not committed in-tree.
- When creating or editing GitHub issues, keep the `[ID]` title prefix and the
  milestone/label mapping from build plan §6. Don't rename, re-bucket, or
  merge existing issues without being asked.

## Cursor skills

Cursor counterparts live under `.cursor/skills/` (`create-feature`,
`code-review`, `bugfix`). Keep the Claude `code-review` skill and the Cursor
`code-review` skill aligned with this file and AGENTS.md.
