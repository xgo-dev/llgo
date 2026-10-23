#!/usr/bin/env python3
"""Render go test -json output and retain per-case timings, including failures."""

import argparse
import html
import json
import os
from pathlib import Path
import subprocess
import sys


def cell(value):
    return html.escape(str(value)).replace("|", "&#124;").replace("\n", " ")


def run(command, output, suite, summary=None):
    output.mkdir(parents=True, exist_ok=True)
    cases = []
    print(f"Starting Go test suite {suite}: {command!r}", file=sys.stderr, flush=True)
    with (output / f"{suite}.jsonl").open("w", encoding="utf-8", buffering=1) as log:
        with subprocess.Popen(command, stdout=subprocess.PIPE, text=True,
                              encoding="utf-8", errors="replace", bufsize=1) as process:
            for line in process.stdout:
                log.write(line)
                try:
                    event = json.loads(line)
                except ValueError:
                    # Preserve unexpected tool output so diagnostics never disappear.
                    print(line, end="", flush=True)
                    continue
                if not isinstance(event, dict):
                    print(line, end="", flush=True)
                    continue
                if isinstance(event.get("Output"), str):
                    print(event["Output"], end="", flush=True)
                if event.get("Test") and event.get("Action") in {"pass", "fail", "skip"}:
                    cases.append({key: event.get(key) for key in
                                  ("Package", "Test", "Action", "Elapsed")})
            status = process.wait()
    cases.sort(key=lambda case: case.get("Elapsed") or 0, reverse=True)
    (output / f"{suite}-times.json").write_text(
        json.dumps(cases, indent=2) + "\n", encoding="utf-8")
    # Keep every case in a plain-text artifact and the job log. The summary
    # remains bounded because GitHub limits its size. Parent elapsed times
    # include their subtests and must not be added to child elapsed times.
    with (output / f"{suite}-times.txt").open("w", encoding="utf-8") as report:
        report.write(f"Go test timings: {suite} ({len(cases)} cases, exit {status})\n")
        report.write("Parent times include subtests; do not sum both.\n")
        report.write("Seconds\tResult\tPackage\tCase\n")
        for case in cases:
            report.write(f"{case['Elapsed'] or 0:.2f}\t{case['Action']}\t"
                         f"{case['Package']}\t{case['Test']}\n")
    print()
    # A single full-suite write can hang in a Windows Git Bash/tee pipe even
    # after the reader has received every line. Keep the report writes bounded.
    with (output / f"{suite}-times.txt").open(encoding="utf-8") as timings:
        for line in timings:
            print(line, end="", flush=True)
    if summary:
        with Path(summary).open("a", encoding="utf-8") as report:
            report.write(f"\n### Go tests: {cell(suite)} (exit {status})\n\n")
            for title, rows in (
                ("Slowest tests", [case for case in cases if "/" not in case["Test"]]),
                ("Slowest subtests", [case for case in cases if "/" in case["Test"]]),
            ):
                report.write(f"{title}\n\n| Seconds | Package | Case | Result |\n")
                report.write("| ---: | --- | --- | --- |\n")
                for case in rows[:15]:
                    report.write(f"| {case['Elapsed'] or 0:.2f} | {cell(case['Package'])} | "
                                 f"{cell(case['Test'])} | {case['Action']} |\n")
                report.write("\n")
    return status if status >= 0 else 128 - status


def main():
    # Redirected Windows streams may default to cp1252, but Go diagnostics and
    # test names are UTF-8. Keep them intact on both the live log and stderr.
    sys.stdout.reconfigure(encoding="utf-8", errors="backslashreplace")
    sys.stderr.reconfigure(encoding="utf-8", errors="backslashreplace")
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-dir", type=Path, default=Path("go-test-timings"))
    parser.add_argument("suite", choices=("main", "build", "compiler", "language",
                                          "traceback", "embedded"))
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if not args.command:
        parser.error("a go test -json command is required")
    return run(args.command, args.output_dir, args.suite, os.getenv("GITHUB_STEP_SUMMARY"))


if __name__ == "__main__":
    sys.exit(main())
