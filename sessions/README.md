# Sessions

A learning log: one file per coding session, numbered across all challenges.
Session files are append-only; they are not edited after they are committed.
Broad concepts are collected in [`CONCEPTS.md`](CONCEPTS.md), a living index.

File name: `NNN-YYYY-MM-DD-<challenge>.md`, e.g. `001-2026-10-14-redis.md`.

## Template

```markdown
# Session NNN — <Challenge> — YYYY-MM-DD

**Stages:** <stage name(s)> — passed / in progress

## What was done
- Bullet points, mine and Claude's, in the order things happened

## Concepts learned
- **<Concept>** — what it is in one line; why it came up here; where we applied it (`file:line`)

## Design decisions
- Decision — alternatives considered — why we chose this

## Hints & help
- Hint levels reached, escape hatch used (yes/no), tests written by Claude

## Next time
- Open questions, refactors we noticed, the next stage
```
