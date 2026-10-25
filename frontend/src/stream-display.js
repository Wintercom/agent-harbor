/**
 * 把 Claude --print stream-json 刷屏压成可读行；非 JSON 原样透出。
 */

function contentParts(message) {
  const parts = message?.content;
  if (!Array.isArray(parts)) return [];
  return parts;
}

function formatEvent(obj) {
  if (!obj || typeof obj !== "object") return "";
  const type = obj.type;

  if (type === "system" && obj.subtype === "init") {
    const tools = Array.isArray(obj.tools) ? obj.tools.join(", ") : "";
    return `● 会话就绪 · ${obj.model || "?"} · 工具[${tools || "无"}]\r\n`;
  }

  if (type === "assistant" && obj.message) {
    let out = "";
    for (const part of contentParts(obj.message)) {
      if (part?.type === "text" && part.text) {
        out += `${part.text}\r\n`;
      } else if (part?.type === "tool_use") {
        const name = part.name || "tool";
        const input = part.input && typeof part.input === "object" ? part.input : {};
        const hint =
          input.file_path ||
          input.path ||
          input.command ||
          input.pattern ||
          "";
        out += `⚙ ${name}${hint ? ` · ${String(hint).slice(0, 80)}` : ""}\r\n`;
      }
      // thinking 不展示
    }
    return out;
  }

  if (type === "user" && obj.message?.content) {
    let out = "";
    for (const part of contentParts(obj.message)) {
      if (part?.type !== "tool_result") continue;
      if (part.is_error) {
        const err = String(part.content || "error").split("\n")[0].slice(0, 120);
        out += `✗ ${err}\r\n`;
      }
      // 成功 tool_result 默认折叠，避免大段文件内容刷屏
    }
    return out;
  }

  if (type === "result") {
    const text = obj.result != null ? String(obj.result).trim() : "";
    if (!text) {
      return obj.is_error ? `✗ 本轮失败\r\n` : "";
    }
    // result 常与最后一条 assistant text 重复；只在较短时展示
    if (text.length > 400) {
      return `—— 本轮结束\r\n`;
    }
    return `—— ${text}\r\n`;
  }

  // 其它协议事件静默
  return "";
}

export function createStreamDisplay() {
  let buf = "";
  return {
    push(chunk) {
      if (!chunk) return "";
      buf += chunk;
      let out = "";
      for (;;) {
        const nl = buf.indexOf("\n");
        if (nl < 0) break;
        const line = buf.slice(0, nl).replace(/\r$/, "");
        buf = buf.slice(nl + 1);
        const trimmed = line.trim();
        if (!trimmed) continue;
        if (trimmed.startsWith("{")) {
          try {
            out += formatEvent(JSON.parse(trimmed));
            continue;
          } catch {
            // 半截或非协议 JSON：降级显示截断
            out += `${trimmed.slice(0, 160)}${trimmed.length > 160 ? "…" : ""}\r\n`;
            continue;
          }
        }
        out += `${line}\r\n`;
      }
      return out;
    },
    reset() {
      buf = "";
    },
  };
}
