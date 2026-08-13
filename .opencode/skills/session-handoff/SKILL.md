---
name: session-handoff
description: Use ONLY when user requests a session handoff, context handoff, or continuation document for a fresh agent/session.
compatibility: opencode
metadata:
  invocation: manual
  source: https://github.com/mattpocock/skills/tree/main/skills/productivity/handoff
---

# Session handoff

Compact current conversation so a fresh agent can continue work.

1. Determine OS temporary directory and write a clearly named Markdown handoff there, never inside workspace.
2. If user supplied a next-session focus, tailor handoff to it.
3. Include only:
   - goal and current state
   - completed work
   - unresolved decisions, blockers, and exact next steps
   - validation already run and remaining validation
   - relevant paths, commands, branches, commits, issues, or URLs
   - `Suggested skills` section listing useful skills for continuation
4. Do not copy content already present in specs, plans, ADRs, issues, commits, diffs, or Engram session summaries; reference artifact path or URL instead.
5. Redact secrets, credentials, personal records, and personally identifiable information.
6. Save same concise state through `mem_session_summary` for durable searchable recall. Reference temp handoff path; do not duplicate large artifact bodies.
7. Return temp handoff path and Engram save result.

Repository HANDOFF.md no longer exists: handoffs live in Engram session summaries plus the temp file this skill writes. Never create a new repo-root HANDOFF.md.
