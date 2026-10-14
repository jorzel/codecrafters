# Git workflow

Each challenge folder is its own git repo (remote: `git.codecrafters.io`,
branch `master`). Pushing there is how stages are tested and submitted.
The root repo (remote: GitHub, branch `main`) keeps a copy of every
challenge's files. So every change is committed twice: once in the
challenge repo, once in the root repo.

## Daily work on an existing challenge

```sh
# 1. Commit and submit to CodeCrafters
cd codecrafters-redis-go
git add -A
git commit -m "Implement XRANGE"
git push origin master          # runs the CodeCrafters tests

# 2. Mirror to GitHub
cd ..
git add -A codecrafters-redis-go
git commit -m "redis: implement XRANGE"
git push
```

Step 2 can wait. You can batch several challenge commits into one root commit.

## Adding a new challenge

Plain `git add <folder>` on a new challenge records a submodule pointer
(mode `160000`) instead of the files, and git warns:
`adding embedded git repository`. Import it once with:

```sh
./scripts/add-challenge.sh codecrafters-<name>-go
git commit -m "Add <name> challenge"
git push
```

After that, the daily workflow above works for it as normal.
Also add a row to the challenges table in `README.md`.

## If you already ran `git add <folder>` by mistake

```sh
git update-index --force-remove codecrafters-<name>-go   # drop the pointer (files on disk untouched)
./scripts/add-challenge.sh codecrafters-<name>-go
```

## Checking it's right

```sh
git ls-files -s codecrafters-<name>-go | head
```

Good: one line per file, mode `100644` or `100755`.
Bad: a single line with mode `160000`.
