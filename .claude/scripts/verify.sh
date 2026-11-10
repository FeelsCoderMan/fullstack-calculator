#!/usr/bin/env bash
# Pre-commit gate. Runs every check against exactly what is staged and, only if
# all of them pass, records the staged tree hash in .git/claude-verified-tree.
# The PreToolUse hook (.claude/hooks/require-verified-commit.sh) lets a commit
# through only when that hash still matches the staged tree.
#
# Usage: .claude/scripts/verify.sh   (from anywhere inside the repo)
# Exit:  0 = PASS (stamp written), 1 = FAIL (stamp removed)
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"
stamp="$(git rev-parse --git-path claude-verified-tree)"
rm -f "$stamp" # a failed or interrupted run never leaves an old approval behind

fail() {
	echo "VERIFY: FAIL - $*" >&2
	exit 1
}
step() { echo "==> $*"; }

# 1. Something must be staged.
staged="$(git diff --cached --name-only --diff-filter=d)"
[ -n "$(git diff --cached --name-only)" ] || fail "nothing is staged"

# 2. What gets checked must be what gets committed: the checks run on the
#    working tree, so it has to equal the index.
step "working tree matches the staged index"
git diff --quiet || fail "unstaged changes in tracked files (git diff --stat); stage or stash them"
untracked="$(git ls-files --others --exclude-standard -- backend)"
[ -z "$untracked" ] || fail "untracked files under backend/ would be checked but not committed:
$untracked"
tree_before="$(git write-tree)"

# 3. No secrets or generated artifacts in the commit.
step "no secrets or generated artifacts staged"
if [ -n "$staged" ]; then
	envfiles="$(printf '%s\n' "$staged" | grep -E '(^|/)\.env(\..+)?$' | grep -vE '(^|/)\.env\.example$' || true)"
	[ -z "$envfiles" ] || fail "env files must never be committed (only .env.example):
$envfiles"
	artifacts="$(printf '%s\n' "$staged" | grep -E '(^|/)(bin|tmp|logs|node_modules|dist|coverage)/|(^|/)coverage\.(out|html)$|\.test$' || true)"
	[ -z "$artifacts" ] || fail "generated artifacts must not be committed (AGENTS.md):
$artifacts"
	keys="$(git diff --cached -G'-----BEGIN [A-Z ]*PRIVATE KEY-----' --name-only || true)"
	[ -z "$keys" ] || fail "staged content contains a private key:
$keys"
fi

# 4. Backend gate: tidy, format, vet, lint, tests (race), vulnerabilities.
step "backend: make check"
make -C backend check || fail "backend checks failed (see output above)"

# 5. Nothing changed while the checks ran.
git diff --quiet || fail "files changed while checks ran; re-run verification"
tree_after="$(git write-tree)"
[ "$tree_before" = "$tree_after" ] || fail "the index changed while checks ran; re-run verification"

echo "$tree_after" >"$stamp"
echo "VERIFY: PASS tree=$tree_after"
