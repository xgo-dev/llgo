#!/usr/bin/env python3
"""Decide whether a heavy LLGo workflow may use a runner."""

from __future__ import annotations

import os
from dataclasses import dataclass


UPSTREAM_REPOSITORY = "xgo-dev/llgo"
REVIEW_LABEL = "need-review"


@dataclass(frozen=True)
class Event:
    name: str
    repository: str
    repository_is_fork: bool = False
    ref_type: str = ""
    ref_name: str = ""
    default_branch: str = "main"
    action: str = ""
    event_label: str = ""
    pr_labels: frozenset[str] = frozenset()
    review_ready: bool = False


def decide(event: Event) -> tuple[bool, str]:
    """Return the gate decision and a diagnostic reason."""
    if event.name in {"workflow_dispatch", "schedule"}:
        return True, f"running heavy CI for {event.name}"

    if event.name == "push":
        if event.ref_type == "tag":
            return True, "running tag validation/release CI"
        if (
            event.repository_is_fork
            and event.repository != UPSTREAM_REPOSITORY
            and event.ref_name != event.default_branch
        ):
            return True, "running heavy CI on a fork feature branch"
        return False, "upstream and fork-default-branch pushes do not run heavy CI"

    if event.name != "pull_request":
        return False, f"unsupported heavy-CI event: {event.name}"

    if event.repository != UPSTREAM_REPOSITORY:
        return False, "fork pull requests reuse the feature-branch push CI"

    if not event.review_ready:
        return False, "pull request template and matching fork CI evidence are not review-ready"

    if REVIEW_LABEL not in event.pr_labels:
        return False, f"waiting for the {REVIEW_LABEL} label"

    if event.action == "labeled" and event.event_label != REVIEW_LABEL:
        return False, "an unrelated label must not restart full CI"

    return True, f"{REVIEW_LABEL} authorizes full upstream CI for this SHA"


def _event_from_environment() -> Event:
    labels = frozenset(
        label.strip()
        for label in os.environ.get("CI_PR_LABELS", "").split(",")
        if label.strip()
    )
    return Event(
        name=os.environ.get("CI_EVENT_NAME", ""),
        repository=os.environ.get("CI_REPOSITORY", ""),
        repository_is_fork=os.environ.get("CI_REPOSITORY_IS_FORK", "").lower()
        == "true",
        ref_type=os.environ.get("CI_REF_TYPE", ""),
        ref_name=os.environ.get("CI_REF_NAME", ""),
        default_branch=os.environ.get("CI_DEFAULT_BRANCH", "main"),
        action=os.environ.get("CI_EVENT_ACTION", ""),
        event_label=os.environ.get("CI_EVENT_LABEL", ""),
        pr_labels=labels,
        review_ready=os.environ.get("CI_REVIEW_READY", "").lower() == "true",
    )


if __name__ == "__main__":
    run, reason = decide(_event_from_environment())
    print(f"run={'true' if run else 'false'}")
    print(f"reason={reason}")
