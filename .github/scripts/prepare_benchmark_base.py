"""Keep a benchmark base's dependency actions relative to its own checkout."""
from pathlib import Path
import os
import sys


def prepare(base):
    # Local actions resolve paths from GITHUB_WORKSPACE, even when action.yml
    # lives in another checkout. Preserve the base's dependency recipes while
    # redirecting their nested actions, scripts and cache hashes to that base.
    # Separate temporary install trees so the head can set up a newer toolchain
    # without colliding with the base's clones, extraction or wrapper paths.
    root = Path(base)
    if root.is_absolute() or ".." in root.parts:
        raise ValueError("benchmark base must be a workspace-relative checkout")
    for path in (root / ".github/actions").rglob("action.yml"):
        source = path.read_text()
        prefix = root.as_posix() + "/.github"
        source = source.replace(".github/", prefix + "/").replace(".github\\", prefix + "\\")
        source = source.replace("RUNNER_TEMP", "LLGO_BENCHMARK_BASE_TEMP")
        path.write_text(source.replace("runner.temp", "env.LLGO_BENCHMARK_BASE_TEMP"))
    if "GITHUB_ENV" in os.environ:
        temporary = Path(os.environ["RUNNER_TEMP"]) / "llgo-benchmark-base"
        temporary.mkdir(parents=True, exist_ok=True)
        with open(os.environ["GITHUB_ENV"], "a") as environment:
            environment.write(f"LLGO_BENCHMARK_BASE_TEMP={temporary}\n")


if __name__ == "__main__":
    prepare(sys.argv[1])
