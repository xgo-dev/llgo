#!/usr/bin/env python3

import unittest

from ci_gate import Event, decide


class GateTests(unittest.TestCase):
    def assert_gate(self, expected: bool, **kwargs: object) -> None:
        actual, _ = decide(Event(**kwargs))
        self.assertEqual(expected, actual)

    def test_fork_feature_push_runs(self) -> None:
        self.assert_gate(
            True,
            name="push",
            repository="contributor/llgo",
            repository_is_fork=True,
            ref_type="branch",
            ref_name="feature",
        )

    def test_upstream_and_fork_default_pushes_skip(self) -> None:
        self.assert_gate(
            False,
            name="push",
            repository="xgo-dev/llgo",
            ref_type="branch",
            ref_name="feature",
        )
        self.assert_gate(
            False,
            name="push",
            repository="contributor/llgo",
            repository_is_fork=True,
            ref_type="branch",
            ref_name="main",
        )

    def test_tags_and_manual_events_run(self) -> None:
        self.assert_gate(
            True,
            name="push",
            repository="xgo-dev/llgo",
            ref_type="tag",
            ref_name="v1.0.0",
        )
        self.assert_gate(True, name="workflow_dispatch", repository="xgo-dev/llgo")
        self.assert_gate(True, name="schedule", repository="xgo-dev/llgo")

    def test_upstream_pr_requires_need_review(self) -> None:
        self.assert_gate(
            False,
            name="pull_request",
            repository="xgo-dev/llgo",
            action="synchronize",
        )
        self.assert_gate(
            True,
            name="pull_request",
            repository="xgo-dev/llgo",
            action="synchronize",
            pr_labels=frozenset({"need-review"}),
            review_ready=True,
        )

    def test_upstream_pr_requires_review_ready_metadata(self) -> None:
        self.assert_gate(
            False,
            name="pull_request",
            repository="xgo-dev/llgo",
            action="synchronize",
            pr_labels=frozenset({"need-review"}),
        )

    def test_only_need_review_label_restarts_full_ci(self) -> None:
        labels = frozenset({"need-review", "go-test-compat"})
        self.assert_gate(
            True,
            name="pull_request",
            repository="xgo-dev/llgo",
            action="labeled",
            event_label="need-review",
            pr_labels=labels,
            review_ready=True,
        )
        self.assert_gate(
            False,
            name="pull_request",
            repository="xgo-dev/llgo",
            action="labeled",
            event_label="go-test-compat",
            pr_labels=labels,
            review_ready=True,
        )

    def test_fork_pr_does_not_duplicate_push_ci(self) -> None:
        self.assert_gate(
            False,
            name="pull_request",
            repository="contributor/llgo",
            repository_is_fork=True,
            action="synchronize",
            pr_labels=frozenset({"need-review"}),
        )


if __name__ == "__main__":
    unittest.main()
