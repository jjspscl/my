#!/bin/sh
set -eu

repo="jjspscl/my"
base_url="https://github.com/$repo/releases/download"
install_dir="${MY_MCP_INSTALL_DIR:-$HOME/.local/bin}"
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/my-mcp-install.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

fail() {
	printf 'my-mcp install failed: %s\n' "$1" >&2
	exit 1
}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
	darwin|linux) ;;
	*) fail "unsupported OS: $os (supported: darwin, linux)" ;;
esac

machine=$(uname -m)
case "$machine" in
	x86_64|amd64) arch=amd64 ;;
	aarch64|arm64) arch=arm64 ;;
	*) fail "unsupported architecture: $machine (supported: amd64, arm64)" ;;
esac

if [ -n "${MY_MCP_VERSION:-}" ]; then
	tag=$MY_MCP_VERSION
else
	tag=$(curl -fsSL -H 'Accept: application/vnd.github+json' "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)
	[ -n "$tag" ] || fail "GitHub latest-release response had no tag_name"
fi

version=${tag#v}
archive="my-mcp_${version}_${os}_${arch}.tar.gz"
release_url="$base_url/$tag"

curl -fsSL "$release_url/$archive" -o "$tmp_dir/$archive" || fail "download failed: $archive"
curl -fsSL "$release_url/checksums.txt" -o "$tmp_dir/checksums.txt" || fail "download failed: checksums.txt"

expected=$(sed -n "s/^\([0-9a-fA-F]*\)  *$archive$/\1/p" "$tmp_dir/checksums.txt" | head -n 1)
[ -n "$expected" ] || fail "checksum entry missing for $archive"

if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp_dir/$archive" | awk '{print $1}')
else
	actual=$(shasum -a 256 "$tmp_dir/$archive" | awk '{print $1}')
fi
[ "$actual" = "$expected" ] || fail "checksum mismatch for $archive"

if command -v gh >/dev/null 2>&1; then
	if ! gh attestation verify "$tmp_dir/$archive" --repo "$repo" >/dev/null 2>&1; then
		fail "GitHub attestation verification failed for $archive"
	fi
else
	printf 'warning: gh unavailable; provenance attestation not verified\n' >&2
fi

mkdir -p "$install_dir"
tar -xzf "$tmp_dir/$archive" -C "$tmp_dir"
binary="$tmp_dir/my-mcp"
[ -x "$binary" ] || fail "archive did not contain executable my-mcp"
install -m 0755 "$binary" "$install_dir/my-mcp"

printf 'installed my-mcp %s at %s/my-mcp\n' "$tag" "$install_dir"
case ":${PATH:-}:" in
	*":$install_dir:"*) ;;
	*) printf 'warning: add %s to PATH\n' "$install_dir" >&2 ;;
esac
printf '\nstdio command: %s/my-mcp\n' "$install_dir"
printf 'Requires MY_DATABASE_URL, MY_REDIS_URL, and MY_USER_EMAIL, plus an already-migrated database.\n'
printf 'HTTP transport (dashboard running): http://127.0.0.1:8081/mcp with MY_MCP_ENABLED=true and MY_MCP_TOKEN set.\n'
printf 'Client configuration examples: https://github.com/%s/blob/main/docs/mcp.md\n' "$repo"
