from unittest import mock
import unittest
from urllib.error import HTTPError

from pr_ci import ACTIVE_STATUSES, GitHub, cleanup


def pull(**changes):
    return {"number": 12, "state": "closed", "closed_at": "2026-10-20T10:00:00Z",
            "head": {"repo": {"id": 30}, "ref": "feature", "label": "fork:feature"},
            **changes}


def run(**changes):
    return {"id": 45, "event": "pull_request", "status": "queued", "run_attempt": 1,
            "created_at": "2026-10-20T09:59:00Z",
            "run_started_at": "2026-10-20T09:59:00Z",
            "pull_requests": [{"number": 12}], "head_repository": {"id": 30},
            "head_branch": "feature", **changes}


class FakeGitHub:
    def __init__(self, runs=None, states=None, peers=None, current=None):
        self.runs = runs if runs is not None else [run()]
        self.states = states if states is not None else [pull()]
        self.peers = peers if peers is not None else [pull()]
        self.current = current or {r["id"]: r for r in self.runs}
        self.calls = []
        self.cancelled = []
        self.cancel_error = None

    def request(self, path, method="GET", **params):
        self.calls.append((path, method))
        if path == "pulls/12":
            return self.states.pop(0) if len(self.states) > 1 else self.states[0]
        if path.endswith("/cancel"):
            if self.cancel_error:
                raise self.cancel_error
            self.cancelled.append(int(path.split("/")[-2]))
            return None
        return self.current[int(path.split("/")[-1])]

    def pages(self, path, **params):
        self.calls.append((path, params))
        if path == "pulls":
            yield from self.peers
        else:
            yield from (r for r in self.runs if r["status"] == params["status"])

    def cleanup(self):
        cleanup(self, {"action": "closed", "pull_request": pull()})


class CleanupTests(unittest.TestCase):
    def test_active_statuses_cancel_individual_runs_after_listing(self):
        api = FakeGitHub([run(id=i+1, status=status)
                          for i, status in enumerate(ACTIVE_STATUSES)])
        api.cleanup()
        self.assertEqual(api.cancelled, [1, 2, 3, 4, 5])
        last_list = max(i for i, call in enumerate(api.calls) if call[0] == "actions/runs")
        first_cancel = next(i for i, call in enumerate(api.calls) if call[1] == "POST")
        self.assertLess(last_list, first_cancel)
        for i, call in enumerate(api.calls):
            if call[1] == "POST":
                self.assertEqual(api.calls[i-1], ("pulls/12", "GET"))

    def test_open_or_different_close_event_stops_before_listing(self):
        for current in (pull(state="open", closed_at=None),
                        pull(closed_at="2026-10-20T10:02:00Z")):
            with self.subTest(current=current):
                api = FakeGitHub(states=[current])
                api.cleanup()
                self.assertEqual(api.calls, [("pulls/12", "GET")])

    def test_reopen_during_pagination_preserves_all_runs(self):
        api = FakeGitHub(states=[pull(), pull(state="open", closed_at=None)])
        api.cleanup()
        self.assertEqual(api.cancelled, [])

    def test_reopen_between_cancellations_stops_remaining_runs(self):
        api = FakeGitHub(runs=[run(id=45), run(id=46)],
                         states=[pull(), pull(), pull(state="open", closed_at=None)])
        api.cleanup()
        self.assertEqual(api.cancelled, [45])

    def test_delayed_close_never_targets_new_run_ids(self):
        # Even if a reopen's state change has not reached the API, the time
        # boundary excludes the new run before it can reach the POST path.
        api = FakeGitHub(runs=[run(), run(id=46, created_at="2026-10-20T10:00:01Z")])
        api.cleanup()
        self.assertEqual(api.cancelled, [45])
        self.assertNotIn(("actions/runs/46", "GET"), api.calls)

    def test_unsafe_or_ambiguous_run_metadata_is_preserved(self):
        for changes in (
            {"event": "push"}, {"event": "pull_request_target"},
            {"pull_requests": [{"number": 13}]},
            {"pull_requests": [{"number": 12}, {"number": 13}]},
            {"run_attempt": 2}, {"run_attempt": None},
            {"created_at": None}, {"run_started_at": None},
            {"created_at": "2026-10-20T10:00:00Z"},
            {"run_started_at": "2026-10-20T10:00:01Z"},
        ):
            with self.subTest(changes=changes):
                api = FakeGitHub(runs=[run(**changes)])
                api.cleanup()
                self.assertEqual(api.cancelled, [])

    def test_run_is_rechecked_before_post(self):
        for current in (run(status="completed"), run(run_attempt=2),
                        run(run_started_at="2026-10-20T10:00:02Z"),
                        run(pull_requests=[{"number": 13}])):
            with self.subTest(current=current):
                api = FakeGitHub(current={45: current})
                api.cleanup()
                self.assertEqual(api.cancelled, [])

    def test_fork_run_without_associations_requires_unique_matching_head(self):
        api = FakeGitHub(runs=[run(pull_requests=[])])
        api.cleanup()
        self.assertEqual(api.cancelled, [45])
        self.assertIn(("pulls", {"state": "all", "head": "fork:feature"}), api.calls)
        for changes, peers in (
            ({"head_repository": {"id": 31}}, [pull()]),
            ({"head_branch": "other"}, [pull()]),
            ({}, [pull(), pull(number=13)]), ({}, []),
            ({}, [pull(head={"repo": None, "ref": "feature"})]),
        ):
            with self.subTest(changes=changes, peers=peers):
                api = FakeGitHub(runs=[run(pull_requests=[], **changes)], peers=peers)
                api.cleanup()
                self.assertEqual(api.cancelled, [])

    def test_associated_run_still_works_after_fork_deletion(self):
        event_pull = pull(head={"repo": None, "ref": "feature"})
        api = FakeGitHub()
        cleanup(api, {"action": "closed", "pull_request": event_pull})
        self.assertEqual(api.cancelled, [45])
        self.assertFalse(any(call[0] == "pulls" for call in api.calls))

    def test_merge_is_a_close_event_too(self):
        api = FakeGitHub(states=[pull(merged=True)])
        api.cleanup()
        self.assertEqual(api.cancelled, [45])

    def test_non_close_or_incomplete_events_do_nothing(self):
        for event in ({}, {"action": "opened", "pull_request": pull()},
                      {"action": "closed", "pull_request": pull(state="open")},
                      {"action": "closed", "pull_request": pull(closed_at=None)}):
            with self.subTest(event=event):
                api = FakeGitHub()
                cleanup(api, event)
                self.assertEqual(api.calls, [])

    def test_completion_conflict_is_benign_other_errors_fail(self):
        for code in (409, 403, 500):
            with self.subTest(code=code):
                api = FakeGitHub()
                api.cancel_error = HTTPError("https://api.github.com", code, "failure", {}, None)
                if code == 409:
                    api.cleanup()
                else:
                    with self.assertRaises(HTTPError):
                        api.cleanup()

    def test_read_failure_never_cancels(self):
        api = FakeGitHub()
        with mock.patch.object(api, "pages", side_effect=RuntimeError("API unavailable")):
            with self.assertRaises(RuntimeError):
                api.cleanup()
        self.assertEqual(api.cancelled, [])


class GitHubTests(unittest.TestCase):
    @mock.patch.dict("os.environ", {"GITHUB_REPOSITORY": "owner/repo"})
    def test_rejects_non_https_api_urls(self):
        for value in ("http://api.github.com", "file:///tmp/api", "ftp://api.github.com", "api.github.com"):
            with self.subTest(value=value), mock.patch.dict("os.environ", {"GITHUB_API_URL": value}):
                with self.assertRaises(ValueError):
                    GitHub()

    @mock.patch.dict("os.environ", {"GITHUB_REPOSITORY": "owner/repo", "GH_TOKEN": "secret",
                                   "GITHUB_API_URL": "https://api.github.com"})
    def test_requests_encode_parameters_and_handle_empty_post_response(self):
        api = GitHub()
        with mock.patch("pr_ci.urlopen") as open_url:
            open_url.return_value.__enter__.return_value.read.return_value = b""
            self.assertIsNone(api.request("actions/runs/45/cancel", method="POST"))
            request = open_url.call_args.args[0]
            self.assertEqual(request.get_method(), "POST")
            self.assertEqual(request.get_header("Authorization"), "Bearer secret")
            self.assertEqual(open_url.call_args.kwargs["timeout"], 30)
            open_url.return_value.__enter__.return_value.read.return_value = b'{"state":"closed"}'
            self.assertEqual(api.request("pulls", head="fork:a/b"), {"state": "closed"})
            self.assertTrue(open_url.call_args.args[0].full_url.endswith("?head=fork%3Aa%2Fb"))

    @mock.patch.dict("os.environ", {"GITHUB_REPOSITORY": "owner/repo"})
    def test_pagination_handles_full_and_empty_pages(self):
        api = GitHub()
        for pages, expected in (([None], []), ([[]], []),
                                ([list(range(100)), None], list(range(100))),
                                ([{"workflow_runs": list(range(100))},
                                  {"workflow_runs": [100]}], list(range(101)))):
            with self.subTest(pages=pages), mock.patch.object(api, "request", side_effect=pages) as request:
                self.assertEqual(list(api.pages("actions/runs", status="queued")), expected)
                self.assertEqual(request.call_args_list[0].kwargs, {"page": 1, "per_page": 100, "status": "queued"})


if __name__ == "__main__":
    unittest.main()
