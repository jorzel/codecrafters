---
name: review-session
description: End a codecrafters coding session — design review of the current challenge plus a session summary for approval.
disable-model-invocation: true
---

# Review session

Follow the learning-mode rules in the root `CLAUDE.md`. Do not write or
change challenge code here.

## 1. Find the challenge
Use the challenge folder we worked on in this session. If unclear, ask.

## 2. Run the checks
In the challenge folder:
- `go test -count=1 ./...`
- `go vet ./...`
- `gofmt -l .`
- `codecrafters test --previous` (passed stages plus the current one)

Report results in a few lines. A failure of the stage not started yet is expected.

## 3. Design review
Review the challenge code as it is now:
- At most 5 points, most important first.
- For each point: what you see, the concept behind it (name it), and one
  question that leads me to the improvement. No code fixes.
- Call out refactoring signals and test triggers from `CLAUDE.md`.
- Note what improved since the start of the session.

## 4. Session summary
- Next number: highest `NNN` in `sessions/` + 1 (start at 001).
- Draft `sessions/NNN-YYYY-MM-DD-<challenge>.md` from the template in
  `sessions/README.md`, based only on what actually happened in this session.
  Put open review points under "Next time".
- Draft the `sessions/CONCEPTS.md` changes, following the rule in
  `CLAUDE.md`: extend existing broad concepts first; add a row only for a
  genuinely new theme.
- Write both files, show them, and **wait for my approval**.

## 5. After approval
Apply my edits, then commit and push both files in the root repo
(not in the challenge repo).
