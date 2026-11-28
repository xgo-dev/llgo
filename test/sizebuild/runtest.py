#!/usr/bin/env python3
"""Build the size proposal's existing regression cases with a dev LLGo."""

import argparse
import json
import os
import pathlib
import platform
import re
import shutil
import signal
import subprocess
import tempfile


ROOT = pathlib.Path(__file__).resolve().parents[2]


def command(args, env, timeout=300, *, stdout_only=False, cwd=ROOT):
    try:
        result = subprocess.run(args, cwd=cwd, env=env, capture_output=True,
                                text=True, timeout=timeout)
    except subprocess.TimeoutExpired as error:
        # TimeoutExpired may hold bytes even with text=True, or None when the
        # process did not write to a stream before the deadline.
        output = "".join(part.decode(errors="replace") if isinstance(part, bytes)
                         else part or "" for part in (error.stdout, error.stderr))
        raise RuntimeError(f"{args!r} timed out after {timeout}s\n{output}") from error
    if result.returncode:
        raise RuntimeError(f"{args!r} failed ({result.returncode})\n"
                           f"{result.stdout}{result.stderr}")
    return result.stdout if stdout_only else result.stdout + result.stderr


def runtime_free_cache_probe(llgo, wasmer, directory, env):
    marker = "llgo: runtime startup omitted: proven allocation-free WASI program"
    # The same helper package contains a pure path and a panic path. Reuse its
    # normal cache while alternating whole-program runtime requirements.
    for fixture in ("wasm-runtime-free", "wasm-runtime-required", "wasm-runtime-free"):
        module = directory / f"{fixture}.wasm"
        log = command([str(llgo), "build", "-compiler-verbose", "-target=wasi", "-debug-artifact=none",
                       "-o", str(module), f"./internal/build/testdata/{fixture}"], env)
        minimal = fixture == "wasm-runtime-free"
        if (marker in log) != minimal:
            raise RuntimeError(f"{fixture}: incorrect runtime policy\n{log}")
        result = subprocess.run([wasmer, "run", "--enable-exceptions", "--enable-simd",
                                 "--stack-size=1048576", str(module)], cwd=ROOT, env=env,
                                capture_output=True, timeout=90)
        if minimal:
            expected = "zero\x00 世界 42 false -9223372036854775808 18446744073709551615\n".encode()
            if result.returncode or result.stdout or result.stderr != expected:
                raise RuntimeError(f"minimal program changed stderr/output: {result}")
        elif (result.returncode == 0 or b"runtime-free cache sentinel" not in result.stderr
              or b"helper.Fail" not in result.stderr or b"helper.go:6" not in result.stderr):
            raise RuntimeError(f"cached helper lost runtime or panic location: {result}")
    print("WASI runtime-free/full-runtime cache switching and stderr semantics passed", flush=True)

    module = directory / "wasi-minimal-startup.wasm"
    wasi_exported_allocator_probe(llgo, module, env, "./internal/build/testdata/wasm-runtime-free")
    print("WASI allocation-free startup and exported allocator passed", flush=True)


def wasi_exported_allocator_probe(llgo, module, env, fixture, *, compact=False):
    report = command([str(llgo), "build", "-target=wasi", "-Oz", "-deadcodedrop", "-globaldce=false",
                      "-lto=full", "-debug-artifact=none", "-size", "-size-format=json",
                      "-o", str(module), fixture], env, stdout_only=True)
    memory = json.loads(report)["wasm"]["memories"][0]
    command([env.get("NODE", "node"), str(ROOT / "test/sizebuild/wasi-minimal-startup.mjs"),
             str(module), str(memory["initial_bytes"]), str(memory["maximum_bytes"]),
             *(["compact"] if compact else [])], env, timeout=90)


def controlled_environment(source):
    env = dict(source, LLGO_ROOT=str(ROOT), LLGO_BUILD_CACHE="1")
    # Ambient compiler/linker flags can invalidate the runtime-free proof.
    for name in ("LLGO_LTO_PLUGIN", "LLGO_PLAN9ASM_PKGS", "CCFLAGS", "CFLAGS", "LDFLAGS", "EMCC_CFLAGS"):
        env.pop(name, None)
    return env


def wasi_full_lto_size_probe(llgo, wasmer, directory, env):
    cases = (
        ("wasm-fmt-literal-callers",
         "stdout captured before formatting\nliteral format caller parity passed"),
        ("wasm-wasi-thread-startup", "wasi thread startup ok"),
        ("wasm-wasi-timer-startup", "timer result 42"),
        ("wasm-wasi-single-gc", "live stack and globals 89"),
        ("wasm-fmt-map", "wasi default value formatting ok"),
    )
    for fixture, expected in cases:
        module = directory / f"{fixture}-full-size.wasm"
        log = command([str(llgo), "build", "-target=wasi", "-Oz", "-lto=full",
                       "-globaldce=false", "-debug-artifact=none", "-compiler-verbose",
                       "-o", str(module), f"./internal/build/testdata/{fixture}"], env)
        if "llgo: enable closed-program WASI size pruning" not in log:
            raise RuntimeError(f"{fixture}: WASI Full LTO skipped Go reachability pruning\n{log}")
        if fixture == "wasm-fmt-map" and "llgo: specialize default value formatting" not in log:
            raise RuntimeError(f"{fixture}: closed WASI build skipped value formatting specialization\n{log}")
        collector = {
            "wasm-wasi-single-gc": "link WASI collector for one mutator",
            "wasm-wasi-thread-startup": "retain WASI concurrent collector (reachable go statement)",
            "wasm-wasi-timer-startup": "retain WASI concurrent collector",
        }.get(fixture)
        if collector is not None and f"llgo: {collector}" not in log.splitlines():
            raise RuntimeError(f"{fixture}: expected {collector!r}\n{log}")
        compact_allocator = "llgo: use compact WASI libc allocator"
        if fixture == "wasm-wasi-single-gc":
            if compact_allocator not in log:
                raise RuntimeError(f"{fixture}: compact allocator was not linked\n{log}")
            wasi_exported_allocator_probe(llgo, module, env,
                                          f"./internal/build/testdata/{fixture}", compact=True)
        elif collector and compact_allocator in log:
            raise RuntimeError(f"{fixture}: concurrent program changed libc allocator\n{log}")
        actual = command([wasmer, "run", "--enable-exceptions", "--enable-simd",
                          "--stack-size=1048576", str(module)], env, timeout=90)
        if actual.strip() != expected:
            raise RuntimeError(f"{fixture}: expected {expected!r}, got {actual!r}")
    print("WASI Full LTO preserved formatter callers, worker startup, and GC roots", flush=True)


def reflect_size_probes(llgo, wasmer, directory, env):
    fixtures = (
        ("./test/wasm/reflect-method-value", "wasm reflect method values: 42 42 42 42 42"),
        ("./internal/build/testdata/wasm-reflect-type-queries", "reflect type queries passed"),
    )
    for fixture, expected in fixtures:
        module = directory / f"{pathlib.Path(fixture).name}.wasm"
        command([str(llgo), "build", "-target=wasi", "-Oz", "-lto=full",
                 "-globaldce=false", "-debug-artifact=none", "-o", str(module), fixture], env)
        actual = command([wasmer, "run", "--enable-exceptions", "--enable-simd",
                          "--stack-size=1048576", str(module)], env, timeout=90)
        if actual.strip() != expected:
            raise RuntimeError(f"{fixture} under WASI Full LTO: {actual!r}, expected {expected!r}")


def legacy_json_uses_v2(env):
    imports = command(["go", "list", "-f", "{{range .Imports}}{{println .}}{{end}}",
                       "encoding/json"], env, stdout_only=True)
    return "encoding/json/v2" in imports.splitlines()


def native_stdout_pipe_probe(artifact, env):
    rfd, wfd = os.pipe()
    with os.fdopen(rfd, "rb") as reader, os.fdopen(wfd, "wb") as writer:
        os.set_blocking(wfd, False)
        filled = 0
        for block in (b"x" * 4096, b"x"):
            try:
                while True:
                    filled += os.write(wfd, block)
            except BlockingIOError:
                pass
        with subprocess.Popen([str(artifact)], env=env, stdout=writer, stderr=subprocess.PIPE) as process:
            writer.close()
            try:
                process.wait(timeout=0.2)
            except subprocess.TimeoutExpired:
                pass
            else:
                raise RuntimeError("literal stdout returned while a nonblocking pipe was full")
            reader.read(filled)
            try:
                _, stderr = process.communicate(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                raise
        if process.returncode or reader.read() != b"literal %\n" or stderr != b"stdout count and path error ok\n":
            raise RuntimeError("literal stdout did not resume after draining the pipe")
    rfd, wfd = os.pipe()
    os.close(rfd)
    with os.fdopen(wfd, "wb") as writer:
        result = subprocess.run([str(artifact)], env=env, stdout=writer, stderr=subprocess.PIPE, timeout=10)
    if result.returncode != -signal.SIGPIPE:
        raise RuntimeError(f"literal stdout did not preserve SIGPIPE: {result}")


def full_lto_codec_probe(llgo, directory, env, target, runner, suffix, browser=False):
    binary_marker = "llgo: specialize fixed-size binary encoding"
    json_marker = "llgo: specialize statically typed JSON codec"
    init_marker = "llgo: omit unused JSON package initialization"
    standard_file_marker = "llgo: omit unused standard file initializers"
    stdout_marker = "llgo: specialize closed literal stdout"
    descriptor_writer_marker = "llgo: use the stdout descriptor writer"
    readonly_env_marker = "llgo: specialize WASI read-only environment"
    linux_native = target == "native" and platform.system() == "Linux"
    linux_amd64 = linux_native and platform.machine() == "x86_64"
    mac_arm64 = target == "native" and platform.system() == "Darwin" and platform.machine() == "arm64"
    fixtures = [
        ("binary-fixed", "fixed binary codec ok", binary_marker, True),
        ("binary-mixed", "generic binary codec ok", binary_marker, False),
        ("json-static", "typed JSON encoding ok", json_marker, True),
        ("json-mixed", "generic JSON encoding ok", json_marker, False),
    ]
    on_v2 = legacy_json_uses_v2(env)
    if on_v2:
        fixture = "json-typed-decode"
        expected = command(["go", "run", f"./internal/build/testdata/{fixture}"], env).strip()
        fixtures.append((fixture, expected, json_marker, True))
        if linux_native:
            fixture = "json-init-linked"
            expected = command(["go", "run", f"./internal/build/testdata/{fixture}"], env).strip()
            fixtures.append((fixture, expected, init_marker, False))
    if target != "wasi" and (not linux_native or linux_amd64):
        fixtures.append(("wasm-fmt-map", "wasi default value formatting ok",
                         "llgo: specialize default value formatting", True))
    if target == "wasi" or target.startswith("emscripten"):
        fixtures.append(("wasm-standard-files", "standard files and redirected stdout ok",
                         standard_file_marker, False))
        fixtures.append(("wasm-pclntab", "pcln-ok", None, False))
    if target == "wasi":
        fixtures.append(("wasm-literal-stdout", "literal %\nstdout count and path error ok",
                         stdout_marker, True))
        fixtures.append(("wasm-literal-stdout-and-file", "console and file writes ok",
                         stdout_marker, True))
        fixtures.append(("native-map-init", "true true true", "llgo: omit unused map initializers", True))
    if target == "wasi" or linux_amd64 or mac_arm64:
        fixtures.append(("typed-stdout", "% {17 [true false] lab} true\n{17 [true false] lab}\n42.125\nvalue",
                         "llgo: specialize default value formatting", True))
        environment_marker = readonly_env_marker if target == "wasi" else "llgo: specialize native read-only environment"
        fixtures.append(("wasm-environment-readonly", "read-only environment lookup ok",
                         environment_marker, True))
        fixtures.append(("wasm-environment-mutable", "mutable environment lookup ok",
                         environment_marker, False))
    if target.startswith("emscripten"):
        fixtures.append(("wasm-once-panic", "once panic preserved", standard_file_marker, False))
        fixtures.append(("wasm-callback", "wasm callback-only wake ok",
                         "llgo: omit Emscripten scheduler", False))
    if linux_amd64 or mac_arm64:
        fixtures.extend([
            ("wasm-literal-stdout", "literal %\nstdout count and path error ok", stdout_marker, True),
            ("wasm-literal-stdout-and-file", "console and file writes ok", stdout_marker, False),
            ("wasm-standard-files", "standard files and redirected stdout ok", stdout_marker, False),
        ])
        fixture = "native-link-parity"
        expected = command(["go", "run", f"./internal/build/testdata/{fixture}"], env).strip()
        fixtures.append((fixture, expected, "llgo: prune native type metadata", True))
        fixtures.append(("native-map-init", "true true true", "llgo: omit unused map initializers", True))
    for fixture, expected, marker, specialized in fixtures:
        artifact = directory / f"{target}-{fixture}{suffix}"
        flags = ["-target", target] if target != "native" else []
        wasi_stdout = target == "wasi" and fixture == "wasm-literal-stdout"
        wasi_readonly_env = target == "wasi" and fixture == "wasm-environment-readonly"
        if wasi_stdout or wasi_readonly_env:
            flags.extend(["-size", "-size-format=json"])
        log = command([str(llgo), "build", *flags, "-Oz", "-lto=full", "-deadcodedrop",
                       "-globaldce=false", "-debug-artifact=none", "-compiler-verbose",
                       "-o", str(artifact), f"./internal/build/testdata/{fixture}"], env)
        if marker is not None and (marker in log) != specialized:
            raise RuntimeError(f"{target} {fixture}: incorrect specialization policy\n{log}")
        if fixture == "typed-stdout" and descriptor_writer_marker not in log:
            raise RuntimeError(f"{target}: dynamic stdout kept the standard file writer\n{log}")
        if target.startswith("emscripten") and fixture == "wasm-fmt-map" and standard_file_marker not in log:
            raise RuntimeError(f"{target}: unused standard files retained\n{log}")
        if linux_amd64 and on_v2 and fixture in ("json-static", "json-typed-decode"):
            if init_marker not in log:
                raise RuntimeError(f"{target} {fixture}: unused codec initialization retained\n{log}")
        run = [*runner]
        run_env = env
        if fixture.startswith("wasm-environment-"):
            if target == "wasi":
                run.append("--env=LLGO_WASI_TEST_ENV=old=value")
            else:
                run_env = dict(env, LLGO_WASI_TEST_ENV="old=value")
        if target == "wasi" and fixture.startswith("wasm-environment-"):
            if ("llgo: index the WASI environment without a Go map" in log) != wasi_readonly_env:
                raise RuntimeError(f"{target} {fixture}: incorrect read-only environment index policy\n{log}")
        if target == "wasi" and fixture in ("wasm-literal-stdout", "wasm-literal-stdout-and-file",
                                             "wasm-environment-readonly", "wasm-environment-mutable"):
            if ("llgo: omit unobserved WASI environment snapshot" in log) != wasi_stdout:
                raise RuntimeError(f"{target} {fixture}: incorrect environment snapshot policy\n{log}")
        if target == "wasi" and fixture in ("wasm-literal-stdout", "wasm-literal-stdout-and-file", "wasm-standard-files"):
            if (descriptor_writer_marker in log) != (fixture == "wasm-literal-stdout"):
                raise RuntimeError(f"{target} {fixture}: incorrect descriptor writer policy\n{log}")
            if ("llgo: omit unobserved WASI path state" in log) != wasi_stdout:
                raise RuntimeError(f"{target} {fixture}: incorrect path initialization policy\n{log}")
            if fixture in ("wasm-standard-files", "wasm-literal-stdout-and-file"):
                if fixture == "wasm-standard-files" and stdout_marker in log:
                    raise RuntimeError(f"{target}: stdout redirection was specialized\n{log}")
                run.extend([f"--volume={directory}:/tmp", "--env=TMPDIR=/tmp"])
        actual = command([*run, str(artifact)], run_env, timeout=90)
        expected_output = expected
        if fixture == "json-static":
            expected_output = command(["go", "run", "./internal/build/testdata/json-static"], env).strip()
        if actual.strip() != expected_output:
            raise RuntimeError(f"{target} {fixture}: expected {expected_output!r}, got {actual!r}")
        if (linux_amd64 or mac_arm64) and fixture == "wasm-literal-stdout":
            if descriptor_writer_marker not in log:
                raise RuntimeError(f"{target}: missing descriptor writer")
            native_stdout_pipe_probe(artifact, env)
        if wasi_stdout or wasi_readonly_env:
            # The size report precedes stderr, where verbose diagnostics appear.
            report, _ = json.JSONDecoder().raw_decode(log)
            memory = report["wasm"]["memories"][0]
            probe = "wasi-stdout-host.mjs" if wasi_stdout else "wasi-environment-host.mjs"
            command([env.get("NODE", "node"), str(ROOT / "test/sizebuild" / probe),
                     str(artifact), str(memory["initial_bytes"]), str(memory["maximum_bytes"])], env, timeout=90)
        if target.startswith("emscripten") and fixture == "wasm-standard-files":
            if "llgo: omit Emscripten scheduler" in log:
                raise RuntimeError(f"{target}: suspending filesystem operations lost the scheduler")
            manifest = json.loads(artifact.with_suffix(".wasm-manifest.json").read_text(encoding="utf-8"))
            source_size = (ROOT / "targets" / "wasm_fs.js").stat().st_size
            if not artifact.read_text(encoding="utf-8").startswith('import "./wasm_fs.js";'):
                raise RuntimeError(f"{target}: filesystem loader lost the shared host")
            if not any(entry["path"] == "wasm_fs.js" and entry["role"] == "deployment"
                       and 0 < entry["bytes"] < source_size for entry in manifest["artifacts"]):
                raise RuntimeError(f"{target}: minified shared host missing from the deployment manifest")
            command([runner[0], str(ROOT / "test/sizebuild/wasm-filesystem-host-api.mjs"),
                     str(artifact)], env, timeout=90)
        if target == "emscripten" and fixture == "wasm-standard-files":
            result = subprocess.run([runner[0], "--import", str(ROOT / "test/sizebuild/wasm-invalid-fs-flag.mjs"),
                                     *runner[1:], str(artifact)], cwd=ROOT, env=env, timeout=90,
                                    capture_output=True, text=True)
            if result.returncode != 2 or "panic: syscall/js: call of Value.Int on undefined" not in result.stderr:
                raise RuntimeError(f"{target}: filesystem initialization error changed\n{result}")
        if browser:
            node = env.get("NODE", "node")
            command([node, str(ROOT / "dev/test_wasm_browser.mjs"), str(artifact),
                     expected, "single"], env, timeout=90)
        print(f"{target} Full LTO {fixture} passed", flush=True)

def emscripten_startup_probe(llgo, directory, env, target, runner, browser):
    marker = "llgo: omit Emscripten scheduler for a synchronous executable"
    startup_dir = directory / f"{target}-startup"
    startup_dir.mkdir()
    for fixture in ("cprintf", "println"):
        artifact = startup_dir / f"{fixture}.mjs"
        log = command([str(llgo), "build", f"-target={target}", "-Oz", "-lto=full", "-deadcodedrop",
                       "-globaldce=false", "-debug-artifact=none", "-compiler-verbose",
                       "-o", str(artifact), f"./benchmark/binary_size/{fixture}"], env)
        if marker not in log:
            raise RuntimeError(f"{target} {fixture}: direct entry absent\n{log}")
        manifest = json.loads(artifact.with_suffix(".wasm-manifest.json").read_text(encoding="utf-8"))
        if "-sMALLOC=emmalloc" not in manifest["link_flags"]:
            raise RuntimeError(f"{target} {fixture}: allocation-free entry retained the Go allocator")
        if ("wasm_fs.js" in artifact.read_text(encoding="utf-8")
                or artifact.with_name("wasm_fs.js").exists()
                or any(entry["path"] == "wasm_fs.js" for entry in manifest["artifacts"])
                or any(line.startswith("llgo: artifact ") and "wasm_fs.js" in line for line in log.splitlines())):
            raise RuntimeError(f"{target} {fixture}: allocation-free entry retained the Go filesystem host")
        command([runner[0], str(ROOT / "test/sizebuild/wasm-filesystem-host-api.mjs"),
                 str(artifact)], env, timeout=90)
        expected = "Hello, world"
        actual = command([*runner, str(artifact)], env, timeout=90)
        if actual.strip() != expected:
            raise RuntimeError(f"{target} {fixture}: expected {expected!r}, got {actual!r}")
        if browser:
            command([runner[0], str(ROOT / "dev/test_wasm_browser.mjs"), str(artifact), expected],
                    env, timeout=90)
        print(f"{target} Full LTO {fixture} used direct entry", flush=True)

    # Both variants use the same standard-library archives. The channel
    # variant needs the scheduler even though it prints the same result.
    for concurrent in (False, True):
        source = directory / f"{target}-scheduler-{concurrent}.go"
        source.write_text(f"""package main
import "encoding/binary"
type reading struct {{ Tick uint32; Seen [2]uint16 }}
//go:noinline
func report(value reading) {{
    var buf [8]byte
    if _, err := binary.Encode(buf[:], binary.LittleEndian, value); err != nil {{ panic(err) }}
    var result reading
    if _, err := binary.Decode(buf[:], binary.LittleEndian, &result); err != nil {{ panic(err) }}
    tick := result.Tick
    if {str(concurrent).lower()} {{
        ch := make(chan uint32)
        go func() {{ ch <- tick }}()
        tick = <-ch
    }}
    println(tick)
}}
func main() {{ report(reading{{Tick: 42}}) }}
""", encoding="utf-8")
        artifact = source.with_suffix(".mjs")
        log = command([str(llgo), "build", f"-target={target}", "-Oz", "-lto=full", "-deadcodedrop",
                       "-globaldce=false", "-debug-artifact=none", "-compiler-verbose",
                       "-o", str(artifact), str(source)], env)
        verb = "retain" if concurrent else "omit"
        if f"llgo: {verb} Emscripten scheduler after link reachability check" not in log:
            raise RuntimeError(f"{target}: expected to {verb} the scheduler\n{log[-3000:]}")
        manifest = json.loads(artifact.with_suffix(".wasm-manifest.json").read_text(encoding="utf-8"))
        if ("-sASYNCIFY=1" in manifest["link_flags"]) != concurrent:
            raise RuntimeError(f"{target}: manifest records the wrong link variant")
        if command([*runner, str(artifact)], env, timeout=90).strip() != "42":
            raise RuntimeError(f"{target}: scheduler link probe changed the result")
        if browser:
            command([runner[0], str(ROOT / "dev/test_wasm_browser.mjs"), str(artifact), "42"],
                    env, timeout=90)
    print(f"{target} Full LTO scheduler link selection and fallback passed", flush=True)


def wasi_specialization_probe(llgo, directory, env, runner):
    for case, marker, want in (
        ("fmt-dynamic", "llgo: specialize default value formatting", True),
        ("fmt-method-fallback", "llgo: specialize default value formatting", False),
        ("json-static", "llgo: omit unused JSON package initialization", True),
        ("json-init-linked", "llgo: omit unused JSON package initialization", False),
    ):
        fixture = f"./internal/build/testdata/{case}"
        artifact = directory / (case + ".wasm")
        log = command([str(llgo), "build", "-target=wasi", "-Oz", "-lto=full", "-deadcodedrop",
                       "-globaldce=false", "-debug-artifact=none", "-compiler-verbose",
                       "-o", str(artifact), fixture], env)
        if (marker in log) != want:
            raise RuntimeError(f"{case}: unexpected size specialization policy\n{log}")
        if case == "fmt-dynamic" and "llgo: omit unused standard file initializers (3)" not in log:
            raise RuntimeError(f"{case}: retained unused WASI standard files\n{log}")
        variants = ([], ["%[2]*.[3]*[1]f", "%w", "%p"]) if case == "fmt-dynamic" else ([],)
        for args in variants:
            expected = command(["go", "run", fixture, *args], env)
            actual = command([*runner, str(artifact), *(["--", *args] if args else [])], env, timeout=90)
            if actual != expected:
                raise RuntimeError(f"{case}: expected {expected!r}, got {actual!r}")
        print(f"WASI {case} specialization policy and Go parity passed", flush=True)



def run_esp32c3_case(llgo, env, package, flags, *, cwd=ROOT):
    output = command([str(llgo), "run", "-target=esp32c3-basic", "-emulator",
                      *flags, package], env, stdout_only=True, cwd=cwd)
    # The ROM writes its boot banner before handing control to the program.
    boot = re.search(r"^entry 0x[0-9a-fA-F]+\n", output, re.MULTILINE)
    if boot is None or output[boot.end():] != "Hello, world\n":
        raise RuntimeError(f"ESP32-C3 {package} emulator output mismatch\n{output}")
    print(f"ESP32-C3 {package} emulator output passed", flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", choices=("wasi", "emscripten", "native", "esp32c3"), required=True)
    parser.add_argument("--llgo", type=pathlib.Path, help="use an existing development LLGo executable")
    parser.add_argument("--browser", action="store_true", help="also run Emscripten products in Chrome")
    parser.add_argument("--esp32c3-go-root", type=pathlib.Path, action="append", default=[],
                        help="also run fmt with this Go standard library")
    args = parser.parse_args()
    env = controlled_environment(os.environ)
    with tempfile.TemporaryDirectory(prefix="llgo-size-build-") as directory:
        directory = pathlib.Path(directory)
        suffix = ".exe" if os.name == "nt" else ""
        llgo = args.llgo
        if llgo is None:
            llgo = directory / f"llgo{suffix}"
            command(["go", "build", "-tags=dev", "-o", str(llgo), "./cmd/llgo"], env)
        else:
            llgo = llgo.resolve(strict=True)
        if args.target == "emscripten":
            env["LLGO_WASM_WORKERS"] = "1"
            node = env.get("NODE", "node")
            for target in ("emscripten", "emscripten-memory64"):
                runner = [node, str(ROOT / "targets" / f"{target}-runner.mjs")]
                full_lto_codec_probe(llgo, directory, env, target, runner, ".mjs", args.browser)
                emscripten_startup_probe(llgo, directory, env, target, runner, args.browser)
            return
        if args.target == "native":
            full_lto_codec_probe(llgo, directory, env, "native", [], "")
            return
        if args.target == "esp32c3":
            # This entry never builds the Go runtime. Its preloaded runtime
            # packages must not become DCE inputs with missing metadata.
            for cold in (True, False):
                output = directory / "cprintf.elf"
                flags = ["-a"] if cold else []
                command([str(llgo), "build", *flags, "-target=esp32c3",
                         "-deadcodedrop", "-o", str(output),
                         "./benchmark/binary_size/cprintf"], env)
                if output.read_bytes()[:4] != b"\x7fELF":
                    raise RuntimeError("C-only DCE did not produce an ELF image")
            print("ESP32-C3 C-only DCE cold/warm builds passed", flush=True)
            run_esp32c3_case(llgo, env, "./benchmark/binary_size/cprintf", ["-deadcodedrop"])
            output = directory / "fmtprintf.elf"
            flags = ["-Oz", "-deadcodedrop", "-globaldce=false", "-lto=full"]
            command([str(llgo), "build", "-target=esp32c3", *flags,
                     "-o", str(output), "./benchmark/binary_size/fmtprintf"], env)
            if output.read_bytes()[:4] != b"\x7fELF":
                raise RuntimeError("ESP32-C3 fmt did not produce an ELF image")
            run_esp32c3_case(llgo, env, "./benchmark/binary_size/fmtprintf", flags)
            if args.esp32c3_go_root:
                sample = directory / "fmtprintf"
                sample.mkdir()
                (sample / "go.mod").write_text("module embedded.probe\n\ngo 1.20\n")
                shutil.copyfile(ROOT / "benchmark/binary_size/fmtprintf/main.go", sample / "main.go")
                for goroot in args.esp32c3_go_root:
                    go_env = dict(env, GOROOT=str(goroot), GOTOOLCHAIN="local", GOWORK="off",
                                  PATH=os.pathsep.join((str(goroot / "bin"), env["PATH"])))
                    version = command(["go", "env", "GOVERSION"], go_env, cwd=sample).strip()
                    print(f"Testing ESP32-C3 fmt.Printf with {version}", flush=True)
                    run_esp32c3_case(llgo, go_env, ".", flags, cwd=sample)
            return

        wasmer = env.get("WASMER", "wasmer")
        if not shutil.which(wasmer):
            raise RuntimeError(f"Wasmer runner not found: {wasmer}")
        env["RUST_LOG"] = "off"
        runtime_free_cache_probe(llgo, wasmer, directory, env)
        wasi_full_lto_size_probe(llgo, wasmer, directory, env)
        reflect_size_probes(llgo, wasmer, directory, env)
        wasi_runner = [wasmer, "run", "--enable-exceptions", "--enable-simd",
                       "--stack-size=1048576"]
        full_lto_codec_probe(llgo, directory, env, "wasi", wasi_runner, ".wasm")
        wasi_specialization_probe(llgo, directory, env, wasi_runner)
        cases = (("off", "cprintf"), ("off", "println"),
                 ("thin", "cprintf"), ("full", "cprintf"),
                 ("full", "println"), ("thin", "println"),
                 ("full", "fmtprintf"), ("full", "goexit-defer"),
                 ("thin", "goexit-defer"), ("full", "threaded-gc"))
        probes = {
            "goexit-defer": ("./internal/build/testdata/wasm-wasi-goexit-defer",
                             "wasi worker defer ok"),
            "threaded-gc": ("./internal/build/testdata/wasm-wasi-threaded-gc",
                            "wasi threaded gc ok"),
        }
        for mode, sample in cases:
            fixture, expected = probes.get(
                sample, (f"./benchmark/binary_size/{sample}", "Hello, world"))
            # Reuse the LTO packages for the runtime probes; the smaller
            # samples above independently check forced and cached builds.
            builds = (False,) if sample in probes else (True, False)
            for cold in builds:
                output = directory / f"{sample}-{mode}.wasm"
                flags = ["-a"] if cold else []
                if mode != "off":
                    flags.append(f"-lto={mode}")
                command([str(llgo), "build", *flags, "-target=wasi",
                         "-o", str(output), fixture], env)
                actual = command([wasmer, "run", "--enable-exceptions", "--enable-simd",
                                  "--stack-size=1048576", "--volume=" + str(ROOT),
                                  "--volume=/tmp", str(output)], env, timeout=90)
                if actual.strip() != expected:
                    raise RuntimeError(f"{sample}/{mode}: expected {expected!r}, got {actual!r}")
            print(f"WASI {sample}/{mode} builds and Wasmer runs passed", flush=True)


if __name__ == "__main__":
    main()
