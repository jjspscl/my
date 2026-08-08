#!/usr/bin/env bash
set -euo pipefail

rm -rf apps/api/internal/platform/web/static
mkdir -p apps/api/internal/platform/web/static
cp -R apps/web/dist/* apps/api/internal/platform/web/static/
touch apps/api/internal/platform/web/static/.gitkeep
