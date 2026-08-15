@AGENTS.md

## Claude Code

- Project skill available: `code-review` (`.claude/skills/code-review/SKILL.md`)
  — reviews a diff/PR against the Agent Working Agreement in
  docs/GOMBIT_BUILD_PLAN.md §5. Invoke with `/code-review` or ask to review a
  PR/diff; it overrides the bundled `/code-review` for this repo.
- This repo is in the M0 bootstrap stage (see AGENTS.md "Current state").
  Prefer a direct `Read` of `docs/GOMBIT_BUILD_PLAN.md` over spawning an
  Explore subagent for backlog/dependency lookups — there is no runtime source
  tree yet.
- When creating or editing GitHub issues, keep the `[ID]` title prefix and the
  milestone/label mapping from build plan §6. Don't rename, re-bucket, or
  merge existing issues without being asked.
