import unittest
from pathlib import Path

from checks import check_report


def record(status="待执行", **extra):
    value = {
        "id": "I1",
        "status": status,
        "time": "2026-10-24T00:00:00Z",
        "environment": {"os": "macOS", "arch": "arm64"},
        "operation": "受控测试",
        "expected": "拒绝未授权目标",
        "actual": "待执行",
    }
    value.update(extra)
    if status == "通过":
        value.setdefault("evidence", "脱敏退出码")
    return value


class ChecksTest(unittest.TestCase):
    def test_example_report_is_valid(self):
        document = __import__("json").loads((Path(__file__).parent / "fixtures" / "acceptance.example.json").read_text())
        self.assertEqual(check_report(document), [])

    def test_valid_report(self):
        self.assertEqual(check_report({"checks": [record()]}), [])

    def test_pass_requires_evidence(self):
        errors = check_report({"checks": [record("通过", evidence="")]})
        self.assertTrue(any("evidence" in error for error in errors))

    def test_duplicate_and_secret_rejected(self):
        errors = check_report({"checks": [record(actual="api_key=sk-abcdefghijklmnop"), record()]})
        self.assertTrue(any("凭据" in error for error in errors))
        self.assertTrue(any("重复" in error for error in errors))

    def test_invalid_status_and_environment(self):
        errors = check_report({"checks": [record(status="完成", environment={})]})
        self.assertTrue(any("status" in error for error in errors))
        self.assertTrue(any("environment" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
