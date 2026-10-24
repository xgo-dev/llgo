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
    wasm_opt = env.get("WASMOPT")
    if not wasm_opt and env.get("EM_BINARYEN_ROOT"):
        wasm_opt = str(Path(env["EM_BINARYEN_ROOT"]) / "bin" /
                       ("wasm-opt.exe" if os.name == "nt" else "wasm-opt"))
    for command in (
        ["git", "rev-parse", "HEAD"], [llgo, "version"], ["go", "version"], ["emcc", "--version"],
        [wasm_opt or "wasm-opt", "--version"], ["node", "--version"],
        [env["LLGO_BROWSER_CHROME"], "--version"],
    ):
        run(command, env)
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
                             "-coverpkg=./cmd/internal/browser,./internal/browserdebug,./internal/wasmdebug,./internal/debugabi",
                             f"-coverprofile={coverage_dir / f'{profile}-{mode}.out'}"]
                run(test + ["./internal/browserdebug", "./cmd/internal/browser"], env)
            for workers, optimization in ((1, "-O0"), (2, "-O0"), (4, "-O0"), (2, "-Os")):
                mode = "external" if workers == 4 else "embedded"
                variant = f"{profile}-goroutines-{workers}" + ("-optimized" if optimization == "-Os" else "")
                stem = Path(directory) / variant
                env["LLGO_WASM_WORKERS"] = str(workers)
                env["LLGO_BROWSER_DEBUG_OPTIMIZED"] = "1" if optimization == "-Os" else "0"
                run([llgo, "build", "-target", target, optimization,
                     "-tags=llgo.wasm.debugger", f"-debug-artifact={mode}",
                     "-o", str(stem.with_suffix(".mjs")), "."], env, ROOT / "test/debug/wasm")
                env["LLGO_BROWSER_DEBUG_ARTIFACT"] = str(stem.with_suffix(".wasm"))
                test = ["go", "test", "-count=1", "-timeout=3m", "-v"]
                if coverage_dir:
                    test += ["-covermode=atomic", "-coverpkg=./cmd/internal/browser",
                             f"-coverprofile={coverage_dir / f'{variant}.out'}"]
                run(test + ["./cmd/internal/browser", "-run=^TestChromeLanguageExtension$"], env)
            env.pop("LLGO_WASM_WORKERS", None)
            env.pop("LLGO_BROWSER_DEBUG_OPTIMIZED", None)


if __name__ == "__main__":
    main()
