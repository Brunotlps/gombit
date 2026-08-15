---
name: code-review
description: Reviews a Gombit diff or pull request against the agent working agreement, locked architecture decisions, and Go/React security conventions. Use when the user asks for a code review, PR review, diff review, or to check a change before merge.
---

# Code Review

Review the current diff (or a named PR/branch) against Gombit's definition of done. Read `AGENTS.md` first. This skill is the Cursor checklist that `AGENTS.md` refers to; it overrides generic review habits for this repo.

The Claude Code counterpart is `.claude/skills/code-review/SKILL.md`. Keep both aligned.

## When not to use

- Implementing a new backlog item → `create-feature`
- Reproducing and fixing a defect → `bugfix` (review the fix afterward with this skill)

## Setup

1. Determine the review surface: `git diff` / `git diff main...HEAD` / the named PR.
2. Identify the linked issue (`[ID] ...`) and its acceptance criteria in `docs/GOMBIT_BUILD_PLAN.md` §4.
3. If the repo is still pre-code, review docs and scaffolding for invented APIs, scope creep, and decision regressions — not imaginary runtime files.
4. Load [references/checklist.md](references/checklist.md) and walk every item that applies to the diff.

## How to review

Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) and [Go Test Comments](https://go.dev/wiki/TestComments) for style. Gombit-specific bars outrank generic style nits.

Prefer findings over praise. Do not re-litigate locked decisions (Huma, feature-packages, Atlas, JWT-in-memory, D10 envelope, AST-only generators). If the author did re-litigate one, that is a **blocker**.

## Output format

Write the review in this order:

```markdown
## Verdict
Approve | Request changes | Comment-only

## Issue / AC
- Issue: [ID] ...
- AC covered: ...
- AC missing: ...

## Blockers
- ...

## Major
- ...

## Minor / nits
- ...

## Questions
- ...
```

Severity:

| Severity | Meaning |
| --- | --- |
| **Blocker** | Violates the working agreement, a locked decision, a security invariant, or stated AC. Must fix before merge. |
| **Major** | Correctness, test gap, contract drift, or extraction-turned-rewrite. Should fix in this PR. |
| **Minor** | Idiom, naming, docs. Fix if cheap; do not expand scope. |

Cite `path:line` (or a hunk) for every finding. Suggest the fix; do not implement unless the user asked.

## What "approve" requires

All of build plan §5 / `AGENTS.md` working agreement:

1. Tests for new behavior; DB changes green on SQLite + PostgreSQL + MySQL.
2. Docs + example for stable features.
3. Extraction, not rewrite, of code that already passes tests.
4. Generator safety (`go/ast`, idempotent, `--dry-run` / `--force`, no silent overwrite).
5. No secrets in generated frontend; `VITE_*` is public; Appendix C prod checks still fail loudly.
6. API changes regenerate OpenAPI + TS client in the same PR.
7. Scope stays in-milestone; no M6 batteries.
8. PR links its issue and lists satisfied AC.

If the change is docs-only or pre-code, apply the subset that still makes sense (scope, decisions, AC, no invented APIs).
