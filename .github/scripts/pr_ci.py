#!/usr/bin/env python3
"""Cancel only existing PR runs belonging to the current close event."""

from datetime import datetime
import json
import os
from urllib.error import HTTPError
from urllib.parse import urlencode, urlsplit
from urllib.request import Request, urlopen


ACTIVE_STATUSES = ("queued", "in_progress", "waiting", "pending", "requested")


class GitHub:
    def __init__(self):
        self.base = os.environ.get("GITHUB_API_URL", "https://api.github.com")
        if urlsplit(self.base).scheme != "https":
            raise ValueError("GITHUB_API_URL must use HTTPS")
        self.repository = os.environ["GITHUB_REPOSITORY"]

    def request(self, path, method="GET", **params):
        query = "?" + urlencode(params) if params else ""
        request = Request(f"{self.base}/repos/{self.repository}/{path}{query}",
                          method=method, headers={
                              "Authorization": f"Bearer {os.environ['GH_TOKEN']}",
                              "Accept": "application/vnd.github+json",
                          })
        with urlopen(request, timeout=30) as response:
            body = response.read()
            return json.loads(body) if body else None

    def pages(self, path, **params):
        page = 1
        while True:
            result = self.request(path, per_page=100, page=page, **params)
            items = result["workflow_runs"] if isinstance(result, dict) else (result or [])
            yield from items
            if len(items) < 100:
                return
            page += 1


def timestamp(value):
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def cleanup(api, event):
    pull = event.get("pull_request", {})
    if event.get("action") != "closed" or pull.get("state") != "closed":
        return
    number, closed = pull["number"], pull.get("closed_at")
    if not closed:
        return
    cutoff = timestamp(closed)

    def still_closed():
        current = api.request(f"pulls/{number}")
        # A reopen followed by another close belongs to a different event too.
        return current.get("state") == "closed" and current.get("closed_at") == closed

    if not still_closed():
        return
    head = pull.get("head") or {}
    repository = (head.get("repo") or {}).get("id")
    branch = head.get("ref")
    unique_head = False
    if repository and branch and head.get("label"):
        # Fork runs can omit pull_requests. A repository/branch fallback is
        # safe only if exactly one historical PR uses that head in this repo.
        peers = {p["number"] for p in api.pages("pulls", state="all", head=head["label"])
                 if (p.get("head", {}).get("repo") or {}).get("id") == repository
                 and p.get("head", {}).get("ref") == branch}
        unique_head = peers == {number}

    def eligible(run):
        if run.get("event") != "pull_request" or run.get("status") not in ACTIVE_STATUSES:
            return False
        # Preserve reruns, even when created_at still refers to the first run.
        # Missing timestamps/attempts and the cutoff's whole second fail closed.
        if run.get("run_attempt") != 1:
            return False
        for field in ("created_at", "run_started_at"):
            if not run.get(field) or timestamp(run[field]) >= cutoff:
                return False
        associations = {p["number"] for p in run.get("pull_requests", [])}
        return associations == {number} or (not associations and unique_head
            and (run.get("head_repository") or {}).get("id") == repository
            and run.get("head_branch") == branch)

    # Snapshot pagination before mutating list membership by cancelling runs.
    candidates = {}
    for status in ACTIVE_STATUSES:
        for run in api.pages("actions/runs", event="pull_request", status=status):
            if eligible(run):
                candidates[run["id"]] = run
    for run_id in candidates:
        current = api.request(f"actions/runs/{run_id}")
        if not eligible(current):
            continue
        if not still_closed():
            return
        try:
            api.request(f"actions/runs/{run_id}/cancel", method="POST")
        except HTTPError as error:
            if error.code != 409:  # Completion can race the cancellation request.
                raise
        else:
            print(f"Cancelled PR #{number} workflow run {run_id}")


if __name__ == "__main__":
    with open(os.environ["GITHUB_EVENT_PATH"]) as stream:
        cleanup(GitHub(), json.load(stream))
