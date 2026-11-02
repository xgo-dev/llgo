#!/usr/bin/env python3
"""Stream a command to stdout and a log through a native subprocess pipe."""

from pathlib import Path
import subprocess
import sys


def run(log, command):
    # Git Bash's tee pipe can deadlock a native Windows writer after sustained
    # output. Create the child pipe in Python, and forward bytes without a shell
    # pipe or text conversion so quarantine matching sees the original output.
    with Path(log).open("wb") as output:
        with subprocess.Popen(command, stdout=subprocess.PIPE,
                              stderr=subprocess.STDOUT, bufsize=0) as process:
            while chunk := process.stdout.read(16384):
                output.write(chunk)
                output.flush()
                sys.stdout.buffer.write(chunk)
                sys.stdout.buffer.flush()
            status = process.wait()
    return status if status >= 0 else 128 - status


if __name__ == "__main__":
    sys.exit(run(sys.argv[1], sys.argv[2:]))
