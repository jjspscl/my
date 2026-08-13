#!/bin/sh
# release-guard.sh — pre-flight checks and safe housekeeping before a release.
#
# Usage:
#   scripts/release-guard.sh v1.1.0               # audit only; exit 1 on findings
#   scripts/release-guard.sh v1.1.0 --fix         # apply auto-fixable rewrites
#   scripts/release-guard.sh v1.1.0 --skip-build  # skip the heavy build gates
#
# The version may also come from $MY_RELEASE_VERSION (mise run release:guard).
#
# Run this BEFORE `mise run release:tag`. It never tags, pushes, or merges —
# those stay with release:tag and the git push step.

set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"

VERSION="${1:-${MY_RELEASE_VERSION:-}}"
FIX=0
SKIP_BUILD=0
for arg in "$@"; do
	case "$arg" in
		--fix) FIX=1 ;;
		--skip-build) SKIP_BUILD=1 ;;
		--*) echo "unknown flag: $arg" >&2; exit 2 ;;
	esac
done

[ -n "$VERSION" ] || { echo "usage: release-guard.sh <vX.Y.Z> [--fix] [--skip-build]" >&2; exit 2; }

FAIL=0
WARN=0

say() { printf '%s\n' "$1"; }
ok()  { printf '  [OK]   %s\n' "$1"; }
fixed(){ printf '  [FIXED] %s\n' "$1"; }
warn() { WARN=$((WARN+1)); printf '  [WARN] %s\n' "$1"; }
fail() { FAIL=$((FAIL+1)); printf '  [FAIL] %s\n' "$1"; }

# ---- helpers -------------------------------------------------------------

semver_ok() {
	case "$1" in
		v[0-9]*.[0-9]*.[0-9]*) return 0 ;;
		*) return 1 ;;
	esac
}

version_gt() {
	# returns 0 if $1 > $2 (both vX.Y.Z)
	a=$1; b=$2
	am=${a#v}; bm=${b#v}
	IFS=. read -r amj ami ap <<EOF
$am
EOF
	IFS=. read -r bmj bmi bp <<EOF
$bm
EOF
	[ "$amj" -gt "$bmj" ] && return 0
	[ "$amj" -lt "$bmj" ] && return 1
	[ "$ami" -gt "$bmi" ] && return 0
	[ "$ami" -lt "$bmi" ] && return 1
	[ "$ap" -gt "$bp" ]
}

latest_tag() {
	git tag --sort=version:refname | tail -n 1
}

# ---- A. release blockers -------------------------------------------------

say "A. Release blockers"

if [ -n "$(git status --porcelain)" ]; then
	fail "working tree is dirty"
else
	ok "working tree clean"
fi

if semver_ok "$VERSION"; then
	ok "version $VERSION is valid semver"
else
	fail "version $VERSION is not v<major>.<minor>.<patch>"
fi

if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
	fail "tag $VERSION already exists locally"
else
	ok "tag $VERSION does not exist locally"
fi

if git ls-remote --tags --exit-code origin "$VERSION" >/dev/null 2>&1; then
	fail "tag $VERSION already exists on origin"
else
	ok "tag $VERSION does not exist on origin"
fi

# Branch/tag name collision: the v1.2.0 branch/tag trap.
if git branch --list "$VERSION" | grep -q .; then
	fail "a branch named $VERSION exists — tag would collide (delete or rename the branch first)"
else
	ok "no branch collides with tag name $VERSION"
fi

LAST=$(latest_tag)
if [ -n "$LAST" ]; then
	if version_gt "$VERSION" "$LAST"; then
		ok "version $VERSION > latest tag $LAST"
	else
		fail "version $VERSION is NOT greater than latest tag $LAST"
	fi
else
	warn "no tags found; cannot check version monotonicity"
fi

if git ls-files | grep -qE '(^|/)\.env$'; then
	fail "tracked .env file found"
else
	ok "no tracked .env files"
fi

if git grep -lE 'AKIA[0-9A-Z]{16}|BEGIN (RSA|OPENSSH|EC|DSA) PRIVATE KEY|ghp_[A-Za-z0-9]{30,}|sk-[A-Za-z0-9]{20,}' -- . >/dev/null 2>&1; then
	fail "possible secret pattern found in tracked files"
else
	ok "no secret patterns in tracked files"
fi

# ---- B. auto-fixable housekeeping ---------------------------------------

say "B. Version pins and personal defaults (--fix applies)"

fix_installer_pin() {
	file=$1
	pin=$(sed -n 's|.*raw\.githubusercontent\.com/jjspscl/my/\(v[0-9][^/]*\)/scripts/install-mcp\.sh.*|\1|p' "$file" | head -n 1)
	if [ -z "$pin" ]; then
		warn "$file: no installer pin found to bump"
	elif [ "$pin" = "$VERSION" ]; then
		ok "$file: installer already pinned to $VERSION"
	else
		if [ "$FIX" = 1 ]; then
			sed "s|/jjspscl/my/$pin/scripts/install-mcp.sh|/jjspscl/my/$VERSION/scripts/install-mcp.sh|" "$file" > "$file.tmp" && mv "$file.tmp" "$file"
			fixed "$file: installer pin $pin -> $VERSION"
		else
			fail "$file: installer pin $pin != $VERSION (run with --fix)"
		fi
	fi
}

fix_installer_pin README.md
fix_installer_pin docs/mcp.md

fix_email_default() {
	file=$1
	if grep -q "jjspscl@gmail.com" "$file"; then
		if [ "$FIX" = 1 ]; then
			sed 's/jjspscl@gmail.com/you@example.com/g' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
			fixed "$file: e2e default email -> you@example.com"
		else
			fail "$file: contains jjspscl@gmail.com default (run with --fix)"
		fi
	else
		ok "$file: no personal email default"
	fi
}

fix_email_default apps/web/e2e/helpers/env.ts
fix_email_default docs/frontend.md

# ---- C. reported, never auto-fixed --------------------------------------

say "C. Reported only (deliberately not auto-fixed)"

if git grep -l 'jjspscl@gmail.com' -- apps/api/scripts/seed_dev.sql >/dev/null 2>&1; then
	warn "seed_dev.sql contains the maintainer email (35x). Dev fixture only — never shipped (no embed, no archive files:, no image COPY). Author email is in commit headers regardless; scrubbing here buys nothing. Real remedy: switch git config user.email to @users.noreply.github.com + git filter-repo, as a separate task."
fi

if git ls-files | grep -q '^HANDOFF.md$'; then
	warn "HANDOFF.md is tracked (plan says drop it for this release)"
fi
HANDOFF_REFS=$(git grep -l HANDOFF -- . 2>/dev/null | grep -v '^HANDOFF.md$' || true)
if [ -n "$HANDOFF_REFS" ]; then
	warn "HANDOFF.md referenced by: $(echo "$HANDOFF_REFS" | tr '\n' ' ')"
fi

if grep -q "$VERSION" ROADMAP.md 2>/dev/null; then
	ok "ROADMAP.md mentions $VERSION"
else
	warn "ROADMAP.md has no entry mentioning $VERSION"
fi

if [ -n "$LAST" ]; then
	say "Commits since $LAST (changelog shape):"
	git log "$LAST..HEAD" --format='%s' | sed 's/^[^a-z]*//; s/(.*//; s/:.*//' | sort | uniq -c | sort -rn | sed 's/^/    /'
fi

# ---- D. validation gates -------------------------------------------------

say "D. Validation gates"

if [ "$SKIP_BUILD" = 1 ]; then
	warn "build gates skipped (--skip-build)"
else
	for task in lint test typecheck build build:mcp; do
		if mise run "$task" >/tmp/release-guard-$task.log 2>&1; then
			ok "mise run $task"
		else
			fail "mise run $task failed (log: /tmp/release-guard-$task.log)"
		fi
	done
	if mise run release:check >/tmp/release-guard-release-check.log 2>&1; then
		ok "mise run release:check (goreleaser snapshot)"
	else
		fail "mise run release:check failed (log: /tmp/release-guard-release-check.log)"
	fi
fi

# ---- summary -------------------------------------------------------------

say ""
say "Summary: $FAIL failure(s), $WARN warning(s)"
[ "$FAIL" = 0 ] || exit 1
