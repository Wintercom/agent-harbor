#!/usr/bin/env bash
# 将本地 agent-harbor 推送到 GitHub
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# 优先环境变量；否则沿用已有 origin；默认 HTTPS（可用 SSH：HARBOR_REMOTE=git@github.com:Wintercom/agent-harbor.git）
BRANCH="${HARBOR_BRANCH:-main}"
if [[ -n "${HARBOR_REMOTE:-}" ]]; then
  REMOTE_URL="$HARBOR_REMOTE"
elif git remote get-url origin >/dev/null 2>&1; then
  REMOTE_URL="$(git remote get-url origin)"
else
  REMOTE_URL="https://github.com/Wintercom/agent-harbor.git"
fi

if [[ ! -d .git ]]; then
  echo "→ 尚未初始化，先执行: ./scripts/github-pull.sh"
  exit 1
fi

git remote set-url origin "$REMOTE_URL" 2>/dev/null || git remote add origin "$REMOTE_URL"
echo "→ remote: $REMOTE_URL"

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
if ! git push -u origin "$BRANCH"; then
  echo ""
  echo "✗ 推送失败：当前环境未登录 GitHub。"
  echo "  任选其一后重跑本脚本："
  echo "  1) gh auth login"
  echo "  2) 将本机 SSH 公钥加到 GitHub，然后："
  echo "       HARBOR_REMOTE=git@github.com:Wintercom/agent-harbor.git ./scripts/github-push.sh"
  echo "  3) 在「终端.app」用已登录的 GitHub Desktop / HTTPS 凭据执行本脚本"
  exit 1
fi

echo "✓ 已推送"
git status -sb
git log -1 --oneline
echo "  https://github.com/Wintercom/agent-harbor"
