#!/usr/bin/env python3
"""在临时非敏感目录探测 Seatbelt；仅作诊断，不是生产 Runtime 或完整验收。"""

import json
import os
from pathlib import Path
import platform
import subprocess
import sys
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"harbor-probe")

    def log_message(self, *_args):
        pass


def quote(value):
    # SBPL 字符串按原始 UTF-8 写入，只转义引号和反斜线。
    if any(ord(char) < 32 for char in value):
        raise ValueError("探测路径含控制字符")
    return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'


def main():
    if sys.platform != "darwin" or os.geteuid() == 0:
        print("仅支持 macOS 非 root 探测", file=sys.stderr)
        return 2
    results = []
    with tempfile.TemporaryDirectory(prefix="harbor-seatbelt-probe-") as temporary:
        root = Path(temporary).resolve()
        workspace = root / 'workspace "引号"'
        home = root / "private-home"
        workspace.mkdir(mode=0o700)
        home.mkdir(mode=0o700)
        (workspace / "allowed.txt").write_text("harbor-fixture", encoding="utf-8")
        outside = root / "outside.txt"
        outside.write_text("outside-fixture", encoding="utf-8")
        (workspace / "escape").symlink_to(outside)
        servers = [ThreadingHTTPServer(("127.0.0.1", 0), Handler) for _ in range(2)]
        for server in servers:
            threading.Thread(target=server.serve_forever, daemon=True).start()
        try:
            allowed_port, denied_port = (server.server_address[1] for server in servers)
            # system.sb 包含系统 IPC 等较宽授权，仅帮助定位基础运行需求。
            # 生产 profile 须继续收敛并验证，不能把此探测模板用于真实 Agent。
            profile = f'''(version 1)
(deny default)
(import "system.sb")
(allow process-exec)
(allow file-read* file-map-executable
    (literal "/bin/cat") (literal "/usr/bin/touch") (literal "/usr/bin/curl"))
(allow file-read* (literal "/private/etc/ssl/openssl.cnf"))
(allow file-read* file-write* (subpath {quote(str(workspace))}) (subpath {quote(str(home))}))
(allow network-outbound (remote tcp "localhost:{allowed_port}"))
'''
            env = {"HOME": str(home), "PATH": "/usr/bin:/bin", "LC_ALL": "C"}

            def check(name, command, expect_success, expected=None):
                result = subprocess.run(
                    ["/usr/bin/sandbox-exec", "-p", profile, *command],
                    cwd=workspace, env=env, capture_output=True, timeout=5,
                )
                passed = (result.returncode == 0) == expect_success
                if expected is not None:
                    passed = passed and result.stdout == expected
                # 拒绝必须是命令报告的普通失败，崩溃不能记作边界通过。
                if not expect_success:
                    passed = passed and result.returncode == (7 if command[0] == "/usr/bin/curl" else 1)
                results.append({"check": name, "passed": passed, "exitCode": result.returncode})
                if not passed:
                    print(result.stderr.decode("utf-8", errors="replace")[:2000], file=sys.stderr)

            check("workspace_read", ["/bin/cat", str(workspace / "allowed.txt")], True, b"harbor-fixture")
            check("workspace_write", ["/usr/bin/touch", str(workspace / "created.txt")], True)
            check("outside_read_denied", ["/bin/cat", str(outside)], False)
            check("symlink_escape_denied", ["/bin/cat", str(workspace / "escape")], False)
            check("outside_write_denied", ["/usr/bin/touch", str(root / "forbidden.txt")], False)
            for name, port, permitted in [("gate_port", allowed_port, True), ("other_loopback_denied", denied_port, False)]:
                url = f"http://127.0.0.1:{port}/"
                check(name, ["/usr/bin/curl", "--silent", "--show-error", "--max-time", "2", "--noproxy", "*", url], permitted, b"harbor-probe" if permitted else None)
        finally:
            for server in servers:
                server.shutdown()
                server.server_close()
    print(json.dumps({"os": platform.mac_ver()[0], "cpu": platform.machine(), "profile": "diagnostic-system.sb-not-production", "results": results}, ensure_ascii=False, indent=2))
    return 0 if all(item["passed"] for item in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
