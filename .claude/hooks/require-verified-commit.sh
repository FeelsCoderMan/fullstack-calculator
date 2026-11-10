#!/usr/bin/env bash
# PreToolUse(Bash) hook: block `git commit` unless .claude/scripts/verify.sh
# passed for exactly the tree that is about to be committed.
# Exit 0 = allow, exit 2 = block (stderr is shown to Claude).
set -uo pipefail

cmd="$(jq -r '.tool_input.command // empty')"

# Only act on git commit (also `git -C dir commit`, `git -c k=v commit`, after && ; | etc.).
commit_re='(^|[;&|({`"'\''[:space:]])git([[:space:]]+-[Cc][[:space:]]+[^[:space:]]+)*[[:space:]]+commit([[:space:]"'\''`;&|)]|$)'
grep -qE "$commit_re" <<<"$cmd" || exit 0

block() {
	echo "BLOCKED: $*" >&2
	echo "Commits must go through the verifier first: run the verifier subagent (or .claude/scripts/verify.sh), then commit via the committer subagent. Never bypass this hook." >&2
	exit 2
}

grep -qE '(^|[[:space:]])--no-verify([[:space:]]|$)|(^|[[:space:]])-n([[:space:]]|$)' <<<"$cmd" &&
	block "--no-verify is not allowed"

repo="${CLAUDE_PROJECT_DIR:-.}"
stamp="$(git -C "$repo" rev-parse --git-path claude-verified-tree 2>/dev/null)" ||
	block "not inside a git repository"
case "$stamp" in /*) ;; *) stamp="$repo/$stamp" ;; esac

[ -f "$stamp" ] || block "no verification on record"

# -a / pathspec commits take content from the working tree, so it must equal the
# verified index as well.
git -C "$repo" diff --quiet ||
	block "unstaged changes exist; the commit could include unverified content"

current="$(git -C "$repo" write-tree 2>/dev/null)" || block "cannot read the staged tree"
[ "$(cat "$stamp")" = "$current" ] ||
	block "the staged changes differ from what was verified (verified $(cat "$stamp"), staged $current)"

exit 0
