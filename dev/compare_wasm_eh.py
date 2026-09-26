#!/usr/bin/env python3

"""Compare legacy EH, direct exnref EH, and Binaryen EH translation.

Requires an Emscripten SDK supporting WASM_LEGACY_EXCEPTIONS=0, Node,
wasm-tools, llvm-dwarfdump, and a Binaryen wasm-opt. Override individual
tools with EMXX, NODE, WASM_TOOLS, LLVM_DWARFDUMP, or WASMOPT; alternatively
set EM_BINARYEN_ROOT to select a complete Binaryen installation. Set LLGO to
add the current Go panic/recover baseline; it is not translated here.
"""

import argparse
import os
import pathlib
import shutil
import subprocess
import tempfile


ROOT = pathlib.Path(__file__).resolve().parent.parent
CPP = ROOT / "dev/testdata/wasm-eh/exception.cpp"
GO = ROOT / "internal/build/testdata/wasm-runtime"
GO_CPP = ROOT / "dev/testdata/wasm-eh/go-cpp-boundary"
EXPECTED = "cpp catch and sjlj ok"


def tool(name, variable):
    value = os.environ.get(variable, name)
    resolved = shutil.which(value)
    if resolved is None:
        raise SystemExit(f"missing {variable}: {value}")
    return resolved


def run(args, *, env=None, timeout=180):
    result = subprocess.run(args, capture_output=True, text=True, env=env, timeout=timeout)
    if result.returncode != 0:
        raise RuntimeError(
            f"{' '.join(map(str, args))} failed ({result.returncode}):\n"
            f"{result.stdout}{result.stderr}"
        )
    return result.stdout + result.stderr


def verify_module(module, wasm_tools, dwarfdump):
    run([wasm_tools, "validate", "--features", "all", str(module)])
    verification = run([dwarfdump, "--verify", str(module)])
    if "No errors." not in verification:
        raise RuntimeError(f"DWARF verification did not pass for {module}:\n{verification}")
    return run([wasm_tools, "print", str(module)])


def run_cpp(module, node):
    output = run([node, "--input-type=module", "-e",
                  "import(process.argv[1]).then(m => m.default())", str(module)])
    if EXPECTED not in output:
        raise RuntimeError(f"unexpected C++ result from {module}:\n{output}")


def run_browser(script, expected, node, env):
    output = run([node, str(ROOT / "dev/test_wasm_browser.mjs"),
                  str(script), expected], env=env, timeout=90)
    if expected not in output:
        raise RuntimeError(f"browser did not report {expected!r} from {script}:\n{output}")


def compare_optimization_level(directory, level, emxx, wasm_opt, wasm_tools, dwarfdump, node, env, browser):
    variants = {}
    for name, legacy in (("legacy", "1"), ("direct", "0")):
        script = directory / f"{name}-O{level}.mjs"
        run([emxx, f"-O{level}", "-g", "-fwasm-exceptions",
             f"-sWASM_LEGACY_EXCEPTIONS={legacy}", "-sSUPPORT_LONGJMP=wasm",
             "-sEXIT_RUNTIME=1", str(CPP), "-o", str(script)], env=env)
        variants[name] = script

    legacy_module = variants["legacy"].with_suffix(".wasm")
    translated_module = directory / f"translated-O{level}.wasm"
    run([wasm_opt, str(legacy_module), "--translate-to-exnref",
         "--enable-exception-handling", "--enable-bulk-memory",
         "--enable-bulk-memory-opt", "--enable-reference-types",
         "--enable-multivalue", "-g",
         "-o", str(translated_module)], env=env)
    legacy_source = variants["legacy"].read_text()
    # Emscripten glue refers to its companion Wasm module by bare filename.
    translated_source = legacy_source.replace(legacy_module.name, translated_module.name)
    if translated_source == legacy_source:
        raise RuntimeError("Emscripten glue did not name its companion wasm module")
    translated_script = translated_module.with_suffix(".mjs")
    translated_script.write_text(translated_source)
    variants["translated"] = translated_script

    sizes = {}
    for name, script in variants.items():
        module = script.with_suffix(".wasm")
        wat = verify_module(module, wasm_tools, dwarfdump)
        if ("try_table" in wat) != (name != "legacy"):
            raise RuntimeError(f"{name} emitted the wrong EH instruction family")
        run_cpp(script, node)
        if browser:
            run_browser(script, EXPECTED, node, env)
        sizes[name] = module.stat().st_size
    print(f"O{level}: " + ", ".join(f"{name}={size} bytes" for name, size in sizes.items()))


def run_go_baseline(directory, llgo, node, env):
    script = directory / "go-panic-recover.mjs"
    go_env = env.copy()
    go_env["LLGO_ROOT"] = str(ROOT)
    run([llgo, "build", "-target", "emscripten", "-o", str(script), str(GO)], env=go_env)
    output = run([node, str(ROOT / "targets/emscripten-runner.mjs"), str(script)], env=go_env)
    if "js" not in output.splitlines():
        raise RuntimeError(f"Go panic/recover baseline failed:\n{output}")
    print("Go panic/recover baseline: passed")


def run_go_cpp_boundary(directory, llgo, node, env, browser):
    go_env = env.copy()
    go_env["LLGO_ROOT"] = str(ROOT)
    # LLGoFiles expands one env-provided compiler argument. Keep C++ EH
    # inside the wrapper and use JS EH only for that C++ translation unit.
    go_env["LLGO_EH_CFLAGS"] = "-fexceptions"
    for level in (0, 2):
        script = directory / f"go-cpp-boundary-O{level}.mjs"
        run([llgo, "build", f"-O={level}", "-target", "emscripten", "-o",
             str(script), str(GO_CPP)], env=go_env)
        output = run([node, str(ROOT / "targets/emscripten-runner.mjs"),
                      str(script)], env=go_env)
        if "go cpp boundary ok" not in output.splitlines():
            raise RuntimeError(f"Go/C++ wrapper at O{level} failed:\n{output}")
        if browser:
            run_browser(script, "go cpp boundary ok", node, go_env)
    print("Go/C++ catch-status-panic wrapper: passed at O0 and O2")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--browser", action="store_true",
                        help="also execute every EH variant and Go/C++ wrapper in Chrome")
    args = parser.parse_args()
    emxx = tool("em++", "EMXX")
    node = tool("node", "NODE")
    wasm_tools = tool("wasm-tools", "WASM_TOOLS")
    dwarfdump = tool("llvm-dwarfdump", "LLVM_DWARFDUMP")
    wasm_opt = os.environ.get("WASMOPT")
    if not wasm_opt and os.environ.get("EM_BINARYEN_ROOT"):
        wasm_opt = str(pathlib.Path(os.environ["EM_BINARYEN_ROOT"]) / "bin/wasm-opt")
    wasm_opt = tool(wasm_opt or "wasm-opt", "WASMOPT")
    env = os.environ.copy()
    with tempfile.TemporaryDirectory(prefix="llgo-wasm-eh-") as temporary:
        directory = pathlib.Path(temporary)
        for level in (0, 2):
            compare_optimization_level(directory, level, emxx, wasm_opt,
                                       wasm_tools, dwarfdump, node, env, args.browser)
        if llgo := os.environ.get("LLGO"):
            run_go_baseline(directory, llgo, node, env)
            run_go_cpp_boundary(directory, llgo, node, env, args.browser)


if __name__ == "__main__":
    main()
