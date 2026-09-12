"""Regression coverage for the exported history, without rendering a movie."""

import tempfile
import unittest
from pathlib import Path

from movie import custom_field, prepare, read_history, timing


def commit(sha, timestamp, changes=(), subject="A change"):
    fields = [
        "",
        "PICFETCH-COMMIT",
        sha,
        str(timestamp),
        "2026-09-12T16:00:00+02:00",
        "Ronin",
        subject,
    ]
    for action, path in changes:
        fields.extend(["\n" + action, path])
    return "\0".join(fields).encode() + b"\0"


class HistoryTests(unittest.TestCase):
    def test_captions_prefer_features_and_include_late_milestones(self):
        raw = (
            commit("a", 100, [("A", "a.go")])
            + commit("b", 800, [("A", "notes.md"), ("A", "more.md")], "Housekeeping notes")
            + commit("c", 1000, [("A", "feature.go")], "feat: add a new explorer")
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "merge-trees").mkdir()
            (root / "source-head.txt").write_text("c\n")
            (root / "history.git").write_bytes(raw)
            (root / "expected-tree.paths").write_bytes(b"a.go\0notes.md\0more.md\0feature.go\0")
            _, summary = prepare(root, "30")
            captions = (root / "captions.txt").read_text()
        self.assertEqual(summary["commits"], 3)
        self.assertIn("feat: add a new explorer", captions)
        self.assertNotIn("Housekeeping", captions)

    def test_history_preserves_edits_deletions_and_empty_commits(self):
        raw = (
            commit("a", 100, [("A", "a.go"), ("A", "old.txt")])
            + commit("b", 110, [("M", "a.go"), ("D", "old.txt")])
            + commit("c", 120)
        )
        history = read_history(raw, {}, {"a.go"})
        self.assertEqual([row["files"] for row in history["commits"]], [2, 1, 1])
        self.assertEqual([event[2] for event in history["events"]], ["A", "A", "M", "D"])
        self.assertEqual(history["head"], "c")

    def test_backward_clock_keeps_causal_order_and_original_time(self):
        raw = commit("a", 110, [("A", "a.go")]) + commit("b", 100, [("M", "a.go")])
        history = read_history(raw, {}, {"a.go"})
        self.assertEqual([c["timestamp"] for c in history["commits"]], [110, 110])
        self.assertEqual(history["commits"][1]["original_timestamp"], 100)
        self.assertEqual(history["timestamp_clamps"], 1)

    def test_merge_removes_a_file_resurrected_by_side_branch_history(self):
        raw = (
            commit("a", 100, [("A", "a.go")])
            + commit("main", 110, [("D", "a.go")])
            + commit("side", 120, [("M", "a.go")])
            + commit("merge", 130)
        )
        history = read_history(raw, {"merge": set()}, set())
        self.assertEqual(history["commits"][-1]["files"], 0)
        self.assertEqual(history["events"][-1][2:4], ("D", "a.go"))
        self.assertEqual(history["merge_reconciliations"], 1)

    def test_unusual_paths_and_author_text_remain_distinct(self):
        paths = {
            "space name.go",
            "tab\tfile.go",
            "new\nline.go",
            "pipe|name.go",
            "pipe%7Cname.go",
            "café.go",
        }
        history = read_history(commit("a", 100, [("A", p) for p in sorted(paths)]), {}, paths)
        encoded = [custom_field(event[3]) for event in history["events"]]
        self.assertEqual(len(set(encoded)), len(paths))
        self.assertTrue(
            all("|" not in name and "\n" not in name and "\t" not in name for name in encoded)
        )
        self.assertEqual(custom_field("Ronin|Pico\n"), "Ronin%7CPico%0A")

    def test_incomplete_or_mismatched_history_fails_before_render(self):
        with self.assertRaisesRegex(ValueError, "empty"):
            read_history(b"", {}, set())
        with self.assertRaisesRegex(ValueError, "final tree"):
            read_history(commit("a", 100, [("A", "a.go")]), {}, {"b.go"})
        with self.assertRaises(ValueError):
            read_history(b"\0PICFETCH-COMMIT\0truncated", {}, set())

    def test_timing_bounds_and_single_timestamp(self):
        result = timing(100, 100, "30")
        self.assertGreater(result["seconds_per_day"], 0)
        self.assertEqual(result["body_seconds"], 19)
        self.assertAlmostEqual(timing(100, 86500, "180")["seconds_per_day"], 161)
        for invalid in ["0", "29", "901", "nan", "inf", "hello"]:
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                timing(100, 200, invalid)


if __name__ == "__main__":
    unittest.main()
