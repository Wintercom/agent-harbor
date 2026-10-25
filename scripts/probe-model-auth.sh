#!/usr/bin/env bash
# 本机验证模型上游 Key（不经过 Harbor）。勿把 Key 贴到聊天或提交仓库。
# 用法：
#   export KEY='sk-…'
#   ./scripts/probe-model-auth.sh https://llmapi.qiniu.io
#   ./scripts/probe-model-auth.sh https://api.qnaigc.com
set -euo pipefail
BASE="${1:-https://llmapi.qiniu.io}"
BASE="${BASE%/}"
if [[ -z "${KEY:-}" ]]; then
  echo "请先: export KEY='你的完整 Key'" >&2
  exit 1
fi
echo "→ GET $BASE/v1/models  (Key 长度=${#KEY}, 前缀=${KEY:0:3}…)"
code="$(curl -sS -o /tmp/harbor-probe-models.json -w "%{http_code}" --max-time 15 \
  "$BASE/v1/models" -H "Authorization: Bearer $KEY" -H "Accept: application/json" || true)"
echo "HTTP $code"
head -c 400 /tmp/harbor-probe-models.json 2>/dev/null; echo
if [[ "$code" != "200" ]]; then
  echo "✗ Key 未被该地址接受（与 Harbor 报 UPSTREAM_AUTH_FAILED 一致）。请换对应控制台签发的 Key，或换地址。" >&2
  exit 2
fi
MODEL="${MODEL:-GPT-5.6}"
echo "→ POST $BASE/v1/messages  model=$MODEL"
code="$(curl -sS -o /tmp/harbor-probe-msg.json -w "%{http_code}" --max-time 15 \
  -X POST "$BASE/v1/messages" \
  -H "Authorization: Bearer $KEY" -H "content-type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d "{\"model\":\"$MODEL\",\"max_tokens\":1,\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}]}" || true)"
echo "HTTP $code"
head -c 400 /tmp/harbor-probe-msg.json 2>/dev/null; echo
