#!/usr/bin/env bash
# 启动全量 PAGE 桌面窗（Wails · 真实绑定）
# 上游在「模型 / 工具」页配置（名+地址+Key 或 OAuth），不以 HARBOR_* + demo-* 为全量用法。
#
# 用法（仓库根或任意目录）：
#   ./scripts/dev-desktop.sh
#
# 可选环境变量：
#   SKIP_GENERATE=1   跳过 wails3 generate bindings
#   SKIP_NPM=1        跳过 npm ci
#   WAILS3=wails3     自定义 CLI 路径
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export PATH="${HOME}/go/bin:${PATH}"
WAILS3_BIN="${WAILS3:-wails3}"

echo "→ 仓库: $ROOT"
echo "→ 分支: $(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo '?')"
echo "→ 口径: 全量 PAGE（页面配置上游；mock 请用 frontend 下 npm run dev）"

if ! command -v "$WAILS3_BIN" >/dev/null 2>&1; then
  echo "✗ 未找到 $WAILS3_BIN"
  echo "  安装: go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.34"
  echo "  并确保 \$HOME/go/bin 在 PATH 中"
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "✗ 未找到 node（需要 Node 20.19+ 或 22+）"
  exit 1
fi

if [[ "${SKIP_GENERATE:-0}" != "1" ]]; then
  echo "→ wails3 generate bindings ./..."
  "$WAILS3_BIN" generate bindings ./...
else
  echo "→ 跳过 generate（SKIP_GENERATE=1）"
fi

if [[ "${SKIP_NPM:-0}" != "1" ]]; then
  if [[ ! -d frontend/node_modules ]]; then
    echo "→ npm ci --prefix frontend"
    npm ci --prefix frontend
  else
    echo "→ frontend/node_modules 已存在（跳过 npm ci；强制重装: rm -rf frontend/node_modules）"
  fi
else
  echo "→ 跳过 npm（SKIP_NPM=1）"
fi

echo "→ wails3 dev（顶栏应为「真实绑定」；新方法未接线可能 UNIMPLEMENTED）"
echo "→ 配置账号：侧栏「模型」「工具」→ 再 HOME「新会话」"
exec "$WAILS3_BIN" dev
