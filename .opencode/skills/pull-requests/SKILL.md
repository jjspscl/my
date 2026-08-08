---

name: git-commit-and-push
description: Create production-grade git commits using Conventional Commits and Gitmoji. Use when committing, pushing, preparing PRs, generating release-quality commit history, or documenting code changes.
compatibility: opencode
-----------------------

# Git Commit and Push

Use this skill whenever changes need to be committed and pushed.

The goal is to produce commit history that remains useful months later for:

* developers
* reviewers
* changelog generation
* release notes
* debugging
* git bisect investigations

---

## Commit Philosophy

Optimize for:

1. Atomic commits
2. Future maintainers
3. Release-note quality history
4. Clear intent
5. Accurate change documentation

Never optimize for:

* speed
* short commit messages
* vague summaries

Bad examples:

* update code
* fix issue
* cleanup
* changes
* misc improvements
* wip

---

## Analyze Before Committing

Before creating any commit:

1. Review staged changes.
2. Review unstaged changes.
3. Review git diff.
4. Determine whether changes belong to:

   * feature
   * fix
   * refactor
   * performance
   * documentation
   * tests
   * infrastructure
   * ci/cd
5. Split unrelated work into separate commits.

Never create a large mixed-purpose commit if changes can be separated logically.

---

## Commit Format

Use:

<gitmoji> <type>(<scope>): <summary>

Examples:

✨ feat(auth): implement session-based authentication

🐛 fix(api): prevent duplicate invoice creation

♻️ refactor(state): simplify dashboard data flow

⚡ perf(cache): reduce redundant database queries

📝 docs(readme): document local development workflow

✅ test(auth): improve session validation coverage

---

## Commit Body Requirements

Every commit body must contain:

### Overview

High-level explanation of what the commit accomplishes.

### Why

Explain:

* problem being solved
* motivation
* impact
* user benefit
* developer benefit

### Changes

Provide detailed bullet points.

Mention:

* components
* services
* APIs
* schemas
* database changes
* UI changes
* infrastructure
* configuration
* tooling
* tests

### Technical Notes

Document:

* architectural decisions
* tradeoffs
* state management changes
* dependency updates
* migration requirements
* performance considerations
* security considerations

### Breaking Changes

Explicitly state:

Breaking Changes:

* None

or

Breaking Changes:

* <description>

### Validation

Document verification performed:

* tests executed
* lint results
* build status
* manual testing
* integration testing

---

## Conventional Commit Mapping

Use:

* feat
* fix
* refactor
* perf
* docs
* test
* build
* ci
* chore
* revert

Choose the most accurate type.

Do not default to chore unless no other type applies.

---

## Gitmoji Mapping

Common mappings:

✨ feat
🐛 fix
♻️ refactor
⚡ perf
📝 docs
✅ test
👷 ci
📦 build
🔧 config
🚑 hotfix
🔥 removal
🚚 move
🎨 style

Use the closest semantic match.

---

## Quality Review

Before committing:

Verify:

* commit type is correct
* scope is correct
* title is concise
* body explains WHY
* major changes are documented
* commit is atomic
* no unrelated files included

If the commit message could not serve as release notes, improve it.

---

## Push Workflow

After commit creation:

1. Confirm git status.
2. Commit changes.
3. Push to current upstream branch.
4. Display:

* branch name
* commit hash
* commit title
* complete commit body
* changed files

---

## Success Criteria

A developer unfamiliar with the project should understand:

* what changed
* why it changed
* affected systems
* potential risks
* validation performed

without opening the diff.

