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

# Homebrew Go + `go install` 产物常见路径（裸跑 `wails3` 常缺这个）
export PATH="/opt/homebrew/bin:${HOME}/go/bin:${PATH}"
WAILS3_BIN="${WAILS3:-wails3}"
VITE_PORT="${WAILS_VITE_PORT:-9245}"

echo "→ 仓库: $ROOT"
echo "→ 分支: $(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo '?')"
echo "→ 口径: 全量 PAGE（页面配置上游；mock 请用 frontend 下 npm run dev）"

if ! command -v go >/dev/null 2>&1; then
  echo "✗ 未找到 go"
  echo "  macOS: brew install go"
  exit 1
fi

if ! command -v "$WAILS3_BIN" >/dev/null 2>&1; then
  if [[ -x "${HOME}/go/bin/wails3" ]]; then
    WAILS3_BIN="${HOME}/go/bin/wails3"
  else
    echo "✗ 未找到 wails3"
    echo "  安装: go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.34"
    echo "  并确保 \$HOME/go/bin 在 PATH（本脚本已尝试加入）"
    exit 1
  fi
fi

if ! command -v node >/dev/null 2>&1; then
  echo "✗ 未找到 node（需要 Node 20.19+ 或 22+）"
  exit 1
fi

echo "→ go: $(go version)"
echo "→ wails3: $WAILS3_BIN ($("$WAILS3_BIN" version 2>/dev/null | head -1 || echo ok))"

# 清掉上次残留的桌面/Vite，避免 listen tcp :9245 address already in use
free_dev_port() {
  local port="$1"
  local pids
  pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -z "$pids" ]]; then
    return 0
  fi
  echo "→ 端口 $port 占用中，结束上次 Agent Harbor / Vite…"
  pkill -f 'bin/AgentHarbor.dev.app/Contents/MacOS/AgentHarbor' 2>/dev/null || true
  pkill -f 'wails3 task' 2>/dev/null || true
  pkill -f 'wails3 dev' 2>/dev/null || true
  # 仅杀本仓库 frontend 的 vite
  pkill -f "${ROOT}/frontend/node_modules/.bin/vite" 2>/dev/null || true
  sleep 0.5
  pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    # shellcheck disable=SC2086
    kill $pids 2>/dev/null || true
    sleep 0.5
  fi
  if lsof -tiTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "✗ 端口 $port 仍被占用，请手动: lsof -nP -iTCP:$port -sTCP:LISTEN"
    exit 1
  fi
  echo "→ 端口 $port 已空闲"
}

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

free_dev_port "$VITE_PORT"

echo "→ wails3 task dev (Vite http://127.0.0.1:${VITE_PORT}) — 顶栏应为「真实绑定」"
echo "→ 配置账号：侧栏「模型」「工具」→ 再 HOME「新会话」"
echo "→ 勿裸跑 wails3（PATH 常缺）；请用本脚本，或先: export PATH=\"\$HOME/go/bin:\$PATH\""
exec "$WAILS3_BIN" task dev
