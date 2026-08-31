#!/usr/bin/env bash
# Promote the tracked web/ tree from a source ref onto feature/website-foundation
# so AWS Amplify rebuilds the public site.
#
# Use this while `promote-website-foundation.yml` is not yet on the repository
# default branch (GitHub only dispatches workflows present on the default branch).
#
# Usage: bash scripts/promote-website-tree.sh [source_ref] [--push]
#   Without --push the promotion is staged and shown, then discarded.
set -euo pipefail

source_ref="${1:-HEAD}"
push="${2:-}"
website_branch="feature/website-foundation"
worktree="$(mktemp -d)/website-promote"
stage="$(mktemp -d)/website-stage"

cleanup() {
  git worktree remove --force "$worktree" >/dev/null 2>&1 || true
  rm -rf "$stage"
}
trap cleanup EXIT

git fetch origin "$website_branch"
git worktree add --detach "$worktree" "origin/${website_branch}" >/dev/null

mkdir -p "$stage"
# Build artifacts are rebuilt by Amplify and must not be promoted.
git archive "$source_ref" web | tar -x -C "$stage" \
  --exclude='web/out*' \
  --exclude='web/.next*' \
  --exclude='web/node_modules*' \
  --exclude='web/test-results*' \
  --exclude='web/playwright-report*'

rm -rf "$worktree/web"
cp -a "$stage/web" "$worktree/web"

cd "$worktree"
git add -A -f -- web
changed="$(git status --porcelain -- web | wc -l | tr -d ' ')"
git status --short -- web

if [ "$changed" = "0" ]; then
  echo "Website tree already matches ${website_branch}."
  exit 0
fi

if [ "$push" != "--push" ]; then
  echo "Dry run: ${changed} file(s) would be promoted. Re-run with --push to publish."
  exit 0
fi

source_sha="$(git -C "$OLDPWD" rev-parse --short "$source_ref")"
git commit -q -m "chore(web): promote website from ${source_ref} (${source_sha})"
git push origin "HEAD:${website_branch}"
echo "Promoted ${changed} file(s) to ${website_branch}. Amplify will rebuild."
