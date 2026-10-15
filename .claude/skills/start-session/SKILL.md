---
name: start-session
description: Start a codecrafters coding session — pick up where the last session ended, check the code, and frame the next step.
disable-model-invocation: true
---

# Start session

Follow the learning-mode rules in the root `CLAUDE.md`. Do not write or
change challenge code here. Keep every message short and conversational.

## 1. Find the challenge
Use the challenge named in the arguments. Otherwise take the challenge of the
latest file in `sessions/`. If unclear, ask.

## 2. Read where we left off
Read the latest `sessions/NNN-*-<challenge>.md`. Note its "Next time" list
and which stage was last passed or in progress.

## 3. Run the checks
In the challenge folder:
- `go test -count=1 ./...`
- `go vet ./...`
- `gofmt -l .`
- `codecrafters test --previous`

Report in one or two lines. A failure of the stage not started yet is expected.
Also mention uncommitted changes (`git status --short`), if any.

## 4. The next stage
If I paste a stage description, save it as
`<challenge-folder>/challanges/<stage-code>-<slug>.md` (cleaned-up markdown).
Don't explain the spec in a way that removes the work.

## 5. Pick the first step
Show the open "Next time" items as a short numbered list (one line each)
and ask which one to take first, or whether to go straight to the next stage.
If we start the stage, ask: "What design decision does this stage force?"
