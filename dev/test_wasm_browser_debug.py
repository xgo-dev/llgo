#!/usr/bin/env python3
"""Build current J32/J64 artifacts and exercise the actual Chrome debugger.

Requires LLGO, LLGO_BROWSER_CHROME (Chrome for Testing or Chromium), LLVM,
Emscripten and the pinned LLGo Binaryen. Tests set a source breakpoint through
the installed extension, inspect real paused locals and parked logical Go
stacks after GC, and observe the Go completion output.
"""

import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent.parent


def run(command, env, cwd=ROOT):
    print("+", " ".join(map(str, command)), flush=True)
    subprocess.run(command, cwd=cwd, env=env, check=True, timeout=300)


def main():
    env = os.environ.copy()
    env["LLGO_ROOT"] = str(ROOT)
    if not env.get("LLGO_BROWSER_CHROME"):
        raise SystemExit("LLGO_BROWSER_CHROME is required; no skipped browser acceptance")
    llgo = env.get("LLGO", "llgo")
    coverage_dir = env.get("LLGO_BROWSER_COVERAGE_DIR")
    if coverage_dir:
        coverage_dir = Path(coverage_dir).resolve()
        coverage_dir.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="llgo-browser-debug-") as directory:
        for profile, target in (("j32", "wasm"), ("j64", "emscripten-memory64")):
            runtime_stem = Path(directory) / f"{profile}-runtime"
            run([llgo, "build", "-target", target, "-O0",
                 "-debug-artifact=embedded", "-o", str(runtime_stem.with_suffix(".mjs")),
                 "./internal/browserdebug/testdata/runtime"], env)
            env["LLGO_BROWSER_DEBUG_RUNTIME_ARTIFACT"] = str(runtime_stem.with_suffix(".wasm"))
            for mode in ("embedded", "external"):
                stem = Path(directory) / f"{profile}-{mode}"
                run([llgo, "build", "-target", target, "-O0",
                     f"-debug-artifact={mode}", "-o", str(stem.with_suffix(".mjs")),
                     "./internal/build/testdata/wasm-debug"], env)
                env["LLGO_BROWSER_DEBUG_ARTIFACT"] = str(stem.with_suffix(".wasm"))
                test = ["go", "test", "-count=1", "-timeout=3m", "-v"]
                if coverage_dir:
                    test += ["-covermode=atomic",
                             "-coverpkg=./internal/browserdebug,./internal/wasmdebug,./internal/debugabi",
                             f"-coverprofile={coverage_dir / f'{profile}-{mode}.out'}"]
                run(test + ["./internal/browserdebug", "./cmd/internal/browser"], env)
            for workers in (1, 2, 4):
                mode = "external" if workers == 4 else "embedded"
                stem = Path(directory) / f"{profile}-goroutines-{workers}"
                env["LLGO_WASM_WORKERS"] = str(workers)
                run([llgo, "build", "-target", target, "-O0",
                     "-tags=llgo.wasm.debugger", f"-debug-artifact={mode}",
                     "-o", str(stem.with_suffix(".mjs")), "."], env, ROOT / "test/debug/wasm")
                env["LLGO_BROWSER_DEBUG_ARTIFACT"] = str(stem.with_suffix(".wasm"))
                run(["go", "test", "-count=1", "-timeout=3m", "-v",
                     "./cmd/internal/browser", "-run=^TestChromeLanguageExtension$"], env)
            env.pop("LLGO_WASM_WORKERS", None)


if __name__ == "__main__":
    main()
