#!/usr/bin/env python3
"""Run optimized Emscripten products through their actual host boundaries."""

import argparse
import os
import pathlib
import subprocess
import tempfile


ROOT = pathlib.Path(__file__).resolve().parents[3]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--browser", action="store_true", help="also run each artifact in Chrome")
    args = parser.parse_args()
    env = dict(os.environ, LLGO_ROOT=str(ROOT))
    llgo = env.get("LLGO", "llgo")
    node = env.get("NODE", "node")
    # O3/Os/Oz all run MetaDCE. Exercise real suspension, C ABI/EH, and
    # shared Go/C filesystem state, including the pthread Module contract.
    cases = (
        ("O3", 1, "wasm-callback", "wasm callback-only wake ok"),
        ("Os", 1, "wasm-profile", "wasm ABI profile ok"),
        ("Oz", 1, "wasm-browser-fs", "wasm filesystem ok"),
        ("Oz", 2, "wasm-browser-fs", "wasm filesystem ok"),
    )
    with tempfile.TemporaryDirectory(prefix="llgo-wasm-optimization-") as directory:
        if args.browser:
            negative = pathlib.Path(directory) / "exit-after-marker.mjs"
            negative.write_text(pathlib.Path(__file__).with_name(negative.name).read_text())
            rejected = subprocess.run(
                [node, str(ROOT / "dev/test_wasm_browser.mjs"), str(negative),
                 "wasm deferred marker", "workers"],
                env=dict(env, LLGO_BROWSER_REQUIRE_EXIT="1"),
                capture_output=True, text=True, timeout=90,
            )
            if rejected.returncode == 0 or "exit 2: wasm deferred marker" not in rejected.stderr:
                raise RuntimeError(f"browser accepted a failed program or lost its arguments:\n{rejected.stdout}\n{rejected.stderr}")
        for target in ("emscripten", "emscripten-memory64"):
            for level, workers, fixture, marker in cases:
                name = f"{target}-{level}-{workers}-workers"
                output = pathlib.Path(directory) / f"{name}.mjs"
                build_env = dict(env, LLGO_WASM_WORKERS=str(workers))
                print(f"Checking {name}", flush=True)
                subprocess.run(
                    [llgo, "build", "-target", target, f"-{level}", "-o", str(output),
                     str(ROOT / "internal/build/testdata" / fixture)],
                    env=build_env, check=True, timeout=300,
                )
                result = subprocess.run(
                    [node, str(ROOT / "targets" / f"{target}-runner.mjs"),
                     str(output), "workers" if workers > 1 else "single"],
                    env=build_env, capture_output=True, text=True, timeout=120,
                )
                if result.returncode or marker not in (result.stdout + result.stderr).splitlines():
                    raise RuntimeError(f"{name}: exit {result.returncode}\n{result.stdout}\n{result.stderr}")
                print(marker, flush=True)
                if args.browser:
                    # The FS marker is deferred and also prints on panic.
                    # Pthread programs must actually exit successfully after
                    # the fixture has checked that multiple Ms ran.
                    browser_env = dict(build_env, LLGO_BROWSER_REQUIRE_EXIT="1" if workers > 1 else "0")
                    subprocess.run(
                        [node, str(ROOT / "dev/test_wasm_browser.mjs"), str(output), marker,
                         "workers" if workers > 1 else "single"],
                        env=browser_env, check=True, timeout=90,
                    )
        # Raw js/wasm uses the GoJS provider but shares Emscripten linking.
        output = pathlib.Path(directory) / "raw-js-Oz.mjs"
        raw_env = dict(env, GOOS="js", GOARCH="wasm", LLGO_WASM_WORKERS="1")
        subprocess.run(
            [llgo, "build", "-Oz", "-o", str(output),
             str(ROOT / "internal/build/testdata/wasm-callback")],
            env=raw_env, check=True, timeout=300,
        )
        marker = "wasm callback-only wake ok"
        result = subprocess.run(
            [node, str(ROOT / "targets/emscripten-runner.mjs"), "--browser-only", str(output)],
            env=raw_env, capture_output=True, text=True, timeout=120,
        )
        if result.returncode or marker not in (result.stdout + result.stderr).splitlines():
            raise RuntimeError(f"raw js/wasm: exit {result.returncode}\n{result.stdout}\n{result.stderr}")
        if args.browser:
            subprocess.run(
                [node, str(ROOT / "dev/test_wasm_browser.mjs"), str(output), marker],
                env=raw_env, check=True, timeout=90,
            )


if __name__ == "__main__":
    main()
