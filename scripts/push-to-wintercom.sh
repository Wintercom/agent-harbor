#!/usr/bin/env bash
# 接受邀请后，把本地 main 推到 Wintercom/agent-harbor
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

gh auth setup-git
git remote set-url origin https://github.com/Wintercom/agent-harbor.git
git fetch origin
git push -u origin main
echo "✓ https://github.com/Wintercom/agent-harbor"
git status -sb
