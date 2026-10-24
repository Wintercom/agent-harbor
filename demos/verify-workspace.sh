#!/usr/bin/env bash
# 校验专用演示工作区可写、可重复覆盖 hello.txt（非敏感）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WS="$ROOT/demos/workspace"
EXPECTED="Hello from Agent Harbor mock"

if [[ ! -d "$WS" ]]; then
  echo "FAIL: missing $WS"
  exit 1
fi
if [[ ! -w "$WS" ]]; then
  echo "FAIL: not writable $WS"
  exit 1
fi

printf '%s\n' "$EXPECTED" >"$WS/hello.txt"
got="$(tr -d '\r' <"$WS/hello.txt" | head -n1)"
if [[ "$got" != "$EXPECTED" ]]; then
  echo "FAIL: hello.txt content mismatch: [$got]"
  exit 1
fi

# 二次覆盖（演示可重复）
printf '%s\n' "$EXPECTED" >"$WS/hello.txt"
got2="$(tr -d '\r' <"$WS/hello.txt" | head -n1)"
if [[ "$got2" != "$EXPECTED" ]]; then
  echo "FAIL: overwrite mismatch"
  exit 1
fi

# 拒绝把明显敏感文件名当作「已准备好」
for bad in .env credentials.json id_rsa .ssh; do
  if [[ -e "$WS/$bad" ]]; then
    echo "FAIL: sensitive name present: $bad"
    exit 1
  fi
done

echo "PASS: workspace ok"
echo "abspath: $(python3 -c "import os; print(os.path.realpath('$WS'))")"
echo "relative: demos/workspace"
echo "note: mock UI still uses /demo/project; use this path for live/I2"
