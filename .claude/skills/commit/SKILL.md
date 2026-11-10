---
name: commit
description: The only way to commit in this repo. Runs the verifier subagent (format, vet, lint, race tests, govulncheck, tidy, secret/artifact checks on exactly the staged changes) and, only on PASS, the committer subagent. Use whenever the user asks to commit.
---

# Commit workflow (verifier → committer)

Commits are gated. A PreToolUse hook (`.claude/hooks/require-verified-commit.sh`) blocks
every `git commit` unless `.claude/scripts/verify.sh` passed for exactly the staged tree.
In addition, the git `pre-commit` hook (`.githooks/pre-commit`, enabled via
`core.hooksPath`) runs the same checks, and skips them when the tree was already verified.
Don't try to get around either one. Follow these steps.

1. **Only commit when the user asked to** (AGENTS.md). If you are on the default branch
   and the repo already has commits, ask whether to branch first.
2. **Settle what to commit.** Run `git status --short`. If nothing is staged, propose
   which files to stage and confirm with the user. The working tree must equal the index
   when verifying: stage everything that belongs to this commit, and stash or leave
   unstaged nothing else in tracked files. (Untracked files outside `backend/` are fine.)
3. **Verify.** Spawn the `verifier` subagent (Agent tool, `subagent_type: "verifier"`).
   - On `VERDICT: FAIL`: show the user the failed check and details. Don't commit. Offer
     the suggested fix (e.g. `make -C backend fmt`, `make -C backend lint`). After a fix,
     re-stage and start again from this step. A fix never carries over an old PASS.
   - On `VERDICT: PASS`: continue. Show any warnings to the user.
4. **Commit.** Spawn the `committer` subagent (`subagent_type: "committer"`) and pass it the
   verifier's **full report**, including the `TREE:` line, plus any context about *why*
   the change was made, which helps the message body.
5. **Report** the commit hash and subject. If the committer stopped, relay why.
