#!/usr/bin/env python3
"""校验脱敏后的 Agent Harbor 验收证据；不启动进程、不访问网络、不读取凭据。"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

ALLOWED_STATUS = {"待执行", "通过", "失败", "阻断"}
REQUIRED = {"id", "status", "time", "environment", "operation", "expected", "actual"}
SECRET_PATTERNS = (
    re.compile(r"(?i)(authorization|api[_-]?key|access[_-]?token|secret)\s*[:=]\s*[^\s,;]+"),
    re.compile(r"\b(sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9_]{20,})\b"),
)


def check_report(document: object) -> list[str]:
    if not isinstance(document, dict):
        return ["报告顶层必须是 JSON 对象"]
    records = document.get("checks")
    if not isinstance(records, list) or not records:
        return ["checks 必须是非空数组"]
    errors: list[str] = []
    seen: set[str] = set()
    serialized = json.dumps(document, ensure_ascii=False)
    for pattern in SECRET_PATTERNS:
        if pattern.search(serialized):
            errors.append("报告疑似包含凭据或认证值")
            break
    for index, record in enumerate(records):
        prefix = f"checks[{index}]"
        if not isinstance(record, dict):
            errors.append(f"{prefix} 必须是对象")
            continue
        missing = REQUIRED - record.keys()
        if missing:
            errors.append(f"{prefix} 缺少字段：{','.join(sorted(missing))}")
        check_id = record.get("id")
        if not isinstance(check_id, str) or not check_id:
            errors.append(f"{prefix}.id 必须是非空字符串")
        elif check_id in seen:
            errors.append(f"重复检查编号：{check_id}")
        else:
            seen.add(check_id)
        if record.get("status") not in ALLOWED_STATUS:
            errors.append(f"{prefix}.status 不是允许值")
        for field in ("time", "operation", "expected", "actual"):
            if not isinstance(record.get(field), str) or not record[field].strip():
                errors.append(f"{prefix}.{field} 必须是非空字符串")
        environment = record.get("environment")
        if not isinstance(environment, dict) or not environment.get("os") or not environment.get("arch"):
            errors.append(f"{prefix}.environment 必须包含 os 和 arch")
        if record.get("status") == "通过" and not record.get("evidence"):
            errors.append(f"{prefix} 标记通过时必须提供 evidence")
    return errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path, help="脱敏 JSON 验收报告")
    args = parser.parse_args(argv)
    try:
        document = json.loads(args.report.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        print(f"读取报告失败：{error}", file=sys.stderr)
        return 2
    errors = check_report(document)
    if errors:
        for error in errors:
            print(f"FAIL: {error}")
        return 1
    print(f"PASS: {len(document['checks'])} 条验收记录格式正确，未发现明显凭据模式")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
