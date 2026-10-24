#!/usr/bin/env bash
# 从 GitHub 拉取 agent-harbor 最新代码
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

REMOTE_URL="${HARBOR_REMOTE:-https://github.com/liyufei916/agent-harbor.git}"
BRANCH="${HARBOR_BRANCH:-main}"

if [[ ! -d .git ]]; then
  echo "→ 初始化本地仓库并关联远程"
  git init
  git remote add origin "$REMOTE_URL" 2>/dev/null || git remote set-url origin "$REMOTE_URL"
  git fetch origin
  # 若本地尚无提交，直接检出远程分支
  if ! git rev-parse HEAD >/dev/null 2>&1; then
    git checkout -B "$BRANCH" "origin/$BRANCH"
  else
    git checkout "$BRANCH" 2>/dev/null || git checkout -b "$BRANCH"
    git pull --ff-only origin "$BRANCH"
  fi
else
  git remote set-url origin "$REMOTE_URL" 2>/dev/null || git remote add origin "$REMOTE_URL"
  echo "→ fetch + pull origin/$BRANCH"
  git fetch origin
  git checkout "$BRANCH" 2>/dev/null || git checkout -b "$BRANCH" "origin/$BRANCH"
  git pull --ff-only origin "$BRANCH"
fi

echo "✓ 已与 origin/$BRANCH 同步"
git status -sb
git log -1 --oneline
