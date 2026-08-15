@AGENTS.md

## Claude Code

- Canonical project skills live under `.cursor/skills/` (Cursor + Agent
  Skills standard). Use those workflows even when invoking from Claude Code:
  - `create-feature` — implement one `[ID]` backlog issue
  - `code-review` — review a diff/PR against build plan §5 / AGENTS.md
  - `bugfix` — reproduce → failing test → minimal fix → verify
- If a local `.claude/skills/code-review/SKILL.md` is present, it should
  defer to `.cursor/skills/code-review/SKILL.md` rather than diverge.
- This repo has no source tree yet (see AGENTS.md "Current state"). Prefer a
  direct `Read` of `docs/GOMBIT_BUILD_PLAN.md` over spawning an Explore
  subagent for backlog/dependency lookups — there's nothing to search yet.
- When creating or editing GitHub issues, keep the `[ID]` title prefix and the
  milestone/label mapping from build plan §6. Don't rename, re-bucket, or
  merge existing issues without being asked.
