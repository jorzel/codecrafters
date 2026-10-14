#!/usr/bin/env bash
# Import a challenge folder into the root repo as regular files.
#
# Each challenge folder is its own git repo, so a plain `git add <folder>`
# would record a submodule pointer (mode 160000) instead of its files.
# This stages every tracked or non-ignored file in the folder directly,
# preserving executable bits. Run it once per new challenge; afterwards
# a plain `git add -A <folder>` works.
#
# Usage: scripts/add-challenge.sh <challenge-folder>
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: $0 <challenge-folder>" >&2
  exit 1
fi

dir="${1%/}"
root="$(git rev-parse --show-toplevel)"
cd "$root"

if [ ! -d "$dir/.git" ]; then
  echo "error: $dir is not a git repo (no $dir/.git)" >&2
  exit 1
fi

if [ "$(git ls-files -s -- "$dir" | awk '$1 == "160000"' | wc -l)" -ne 0 ]; then
  echo "error: $dir is staged as a submodule pointer; remove it first:" >&2
  echo "  git update-index --force-remove $dir" >&2
  exit 1
fi

count=0
while IFS= read -r f; do
  path="$dir/$f"
  [ -f "$path" ] || continue
  mode=100644
  [ -x "$path" ] && mode=100755
  sha="$(git hash-object -w -- "$path")"
  git update-index --add --cacheinfo "$mode,$sha,$path"
  count=$((count + 1))
done < <(git -C "$dir" ls-files --cached --others --exclude-standard)

echo "staged $count files from $dir"
