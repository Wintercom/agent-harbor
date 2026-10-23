#!/usr/bin/env bash
# 将本地 agent-harbor 推送到 GitHub
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

REMOTE_URL="${HARBOR_REMOTE:-https://github.com/Wintercom/agent-harbor.git}"
BRANCH="${HARBOR_BRANCH:-main}"

if [[ ! -d .git ]]; then
  echo "→ 尚未初始化，先执行 scripts/github-pull.sh 或自行 git init"
  exit 1
fi

git remote set-url origin "$REMOTE_URL" 2>/dev/null || git remote add origin "$REMOTE_URL"

# 有未提交改动则先提示（默认自动 add+commit，可用 HARBOR_NO_COMMIT=1 跳过）
if [[ -n "$(git status --porcelain)" ]]; then
  if [[ "${HARBOR_NO_COMMIT:-0}" == "1" ]]; then
    echo "✗ 有未提交改动且设置了 HARBOR_NO_COMMIT=1，中止"
    git status -sb
    exit 1
  fi
  MSG="${HARBOR_COMMIT_MSG:-chore: sync agent-harbor docs and prototype}"
  echo "→ 提交本地改动: $MSG"
  git add -A
  git commit -m "$MSG"
fi

git checkout "$BRANCH" 2>/dev/null || git checkout -b "$BRANCH"
echo "→ push origin/$BRANCH"
git push -u origin "$BRANCH"

echo "✓ 已推送"
git status -sb
git log -1 --oneline
echo "  https://github.com/Wintercom/agent-harbor"
