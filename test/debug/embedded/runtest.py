"""Real Cortex-M source debugging and explicit image download with both backends."""

import argparse
import json
import os
from pathlib import Path
import shlex
import subprocess
import sys
import tempfile

from debug_session import FIXTURE, run, session, tool


def qmp_command(process, command):
    process.stdin.write(json.dumps({"execute": command}) + "\n")
    process.stdin.flush()
    while True:
        response = json.loads(process.stdout.readline())
        if "error" in response:
            raise RuntimeError(f"QMP {command}: {response['error']}")
        if "return" in response:
            return response["return"]


def verify_download(log):
    events = [json.loads(line) for line in log.read_text().splitlines()]
    resets = [i for i, event in enumerate(events) if event["event"] == "reset"]
    if len(resets) != 2:
        raise AssertionError(f"expected reset/load/reset, got {events}")
    writes = events[resets[0] + 1:resets[1]]
    if sum(event.get("bytes", 0) for event in writes) < 64:
        raise AssertionError(f"no substantive image download between resets: {events}")
    print(f"verified reset/load/reset with {sum(e.get('bytes', 0) for e in writes)} downloaded bytes")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--backend", choices=("gdb", "lldb", "all"), default="all")
    args = parser.parse_args()
    qemu = tool("qemu-system-arm", os.getenv("QEMU_SYSTEM_ARM"), "qemu-system-arm")
    # Exercise the target's production server template, including on Windows
    # where the isolated QEMU installation is selected by QEMU_SYSTEM_ARM.
    os.environ["PATH"] += os.pathsep + str(Path(qemu).parent)
    objcopy = tool("llvm-objcopy", os.getenv("LLVM_OBJCOPY"), "llvm-objcopy", "llvm-objcopy-22")
    dwarfutil = tool("llvm-dwarfutil", os.getenv("LLVM_DWARFUTIL"), "llvm-dwarfutil", "llvm-dwarfutil-22")
    dwarfdump = tool("llvm-dwarfdump", os.getenv("LLVM_DWARFDUMP"), "llvm-dwarfdump", "llvm-dwarfdump-22")
    backends = ("gdb", "lldb") if args.backend == "all" else (args.backend,)
    with tempfile.TemporaryDirectory(prefix="llgo-embedded-debug-") as directory:
        tmp = Path(directory)
        artifact = tmp / "embedded-debug.elf"
        for backend in backends:
            debugger = (tool("GDB", os.getenv("LLGO_GDB"), "gdb-multiarch", "arm-none-eabi-gdb", "gdb")
                        if backend == "gdb" else tool("LLDB", os.getenv("LLGO_LLDB"), "lldb-22", "lldb"))
            # Existing preloaded-image transport, with no reset/write permission.
            session(backend, debugger, artifact, "cortex-m-qemu")
            # Start with NO kernel image. Reaching the fixture can only succeed
            # when the product's explicit -load path really writes the image.
            log = tmp / f"{backend}-load.jsonl"
            with open(tmp / f"{backend}-qemu.log", "w") as qemu_log:
                process = subprocess.Popen([qemu, "-machine", "lm3s6965evb", "-semihosting", "-nographic",
                                            "-serial", "null", "-monitor", "none", "-S",
                                            "-gdb", "tcp:127.0.0.1:0", "-qmp", "stdio"],
                                           stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                           stderr=qemu_log, text=True)
                try:
                    # QEMU keeps its OS-assigned port bound. Discover it over
                    # QMP instead of closing a reservation before starting it.
                    greeting = json.loads(process.stdout.readline())
                    if "QMP" not in greeting:
                        raise RuntimeError(f"missing QMP greeting: {greeting}")
                    qmp_command(process, "qmp_capabilities")
                    filename = next(item["filename"] for item in qmp_command(process, "query-chardev")
                                    if item["label"] == "gdb")
                    port = int(filename.split("tcp:127.0.0.1:", 1)[1].split(",", 1)[0])
                    proxy = shlex.join([Path(sys.executable).as_posix(), (FIXTURE / "qemu_load_proxy.py").as_posix(),
                                        "--transport", "{debug-stdio}", "--qemu", str(port), "--log", log.as_posix()])
                    session(backend, debugger, artifact, "cortex-m-qemu", ["-server", proxy, "-load"])
                    verify_download(log)
                finally:
                    process.terminate()
                    process.wait(timeout=10)
                    process.stdin.close()
                    process.stdout.close()
            print(f"{backend}: preloaded and reset/load/reset Cortex-M sessions passed")
        # Host DWARF must not change the same ELF's flash bytes.
        stripped = tmp / "stripped.elf"
        run([objcopy, "--strip-debug", artifact, stripped])
        for elf, binary in [(artifact, tmp / "debug.bin"), (stripped, tmp / "stripped.bin")]:
            run([objcopy, "-O", "binary", elf, binary])
        if (tmp / "debug.bin").read_bytes() != (tmp / "stripped.bin").read_bytes():
            raise AssertionError("DWARF changed flash bytes")
        # Verify a GC'd copy: LLD leaves tombstones for discarded functions.
        verified = tmp / "verified.elf"
        run([dwarfutil, "--garbage-collection", "--verify", artifact, verified])
        run([dwarfdump, "--verify", verified])
    print("embedded debugger acceptance passed (QEMU; no physical probe)")


if __name__ == "__main__":
    main()
