"""Native LLGo source-debug acceptance for faults and Go/host frames."""

import os
from pathlib import Path
from typing import Iterable, List

import lldb


class AcceptanceError(RuntimeError):
    pass


def marker_line(path: Path, marker: str) -> int:
    matches = [
        number
        for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1)
        if marker in line
    ]
    if len(matches) != 1:
        raise AcceptanceError(
            f"expected one {marker!r} marker in {path}, found {len(matches)}"
        )
    return matches[0]


def frame_name(frame: lldb.SBFrame) -> str:
    return frame.GetFunctionName() or frame.GetDisplayFunctionName() or "<unknown>"


def frame_location(frame: lldb.SBFrame) -> tuple[str, int]:
    entry = frame.GetLineEntry()
    if not entry.IsValid():
        return "<unknown>", 0
    return entry.GetFileSpec().GetFilename() or "<unknown>", entry.GetLine()


def frames(process: lldb.SBProcess) -> List[lldb.SBFrame]:
    thread = process.GetSelectedThread()
    return [thread.GetFrameAtIndex(index) for index in range(thread.GetNumFrames())]


def describe_process(process: lldb.SBProcess) -> str:
    thread = process.GetSelectedThread()
    lines = [
        f"state={lldb.SBDebugger.StateAsCString(process.GetState())} "
        f"stop={thread.GetStopDescription(256)!r}"
    ]
    for index, frame in enumerate(frames(process)):
        source, line = frame_location(frame)
        lines.append(f"#{index} {frame_name(frame)} at {source}:{line}")
    return "\n".join(lines)


def require_stopped(process: lldb.SBProcess, scenario: str) -> None:
    if not process.IsValid() or process.GetState() != lldb.eStateStopped:
        raise AcceptanceError(f"{scenario} did not stop:\n{describe_process(process)}")


def find_frame(
    process: lldb.SBProcess,
    function_suffix: str,
    source_name: str,
    source_line: int,
) -> int:
    for index, frame in enumerate(frames(process)):
        source, line = frame_location(frame)
        if (
            frame_name(frame).endswith(function_suffix)
            and source == source_name
            and line == source_line
        ):
            return index
    raise AcceptanceError(
        f"missing {function_suffix} at {source_name}:{source_line}:\n"
        f"{describe_process(process)}"
    )


def create_session(executable: str) -> tuple[lldb.SBDebugger, lldb.SBTarget]:
    debugger = lldb.SBDebugger.Create()
    debugger.SetAsync(False)
    # The fixtures contain their DWARF. Keep inferior launch and unwinding
    # independent of external symbol servers, as in the runtime debug suite.
    debugger.HandleCommand("settings set symbols.enable-external-lookup false")
    target = debugger.CreateTarget(executable)
    if not target.IsValid():
        lldb.SBDebugger.Destroy(debugger)
        raise AcceptanceError(f"could not create target for {executable}")
    return debugger, target


def launch(target: lldb.SBTarget, scenario: str, cwd: str) -> lldb.SBProcess:
    process = target.LaunchSimple([scenario], None, cwd)
    require_stopped(process, scenario)
    return process


def destroy_session(debugger: lldb.SBDebugger, process: lldb.SBProcess) -> None:
    if process.IsValid() and process.GetState() not in (
        lldb.eStateExited,
        lldb.eStateDetached,
    ):
        process.Kill()
    lldb.SBDebugger.Destroy(debugger)


def set_panic_breakpoint(target: lldb.SBTarget) -> lldb.SBBreakpoint:
    candidates: Iterable[str] = (
        "github.com/xgo-dev/llgo/runtime/internal/runtime.Panic",
        "runtime.Panic",
    )
    for name in candidates:
        breakpoint = target.BreakpointCreateByName(name)
        if breakpoint.IsValid() and breakpoint.GetNumLocations() > 0:
            return breakpoint
        target.BreakpointDelete(breakpoint.GetID())
    breakpoint = target.BreakpointCreateByRegex(
        r"(^|/)runtime(/internal/runtime)?\.Panic$"
    )
    if not breakpoint.IsValid() or breakpoint.GetNumLocations() == 0:
        raise AcceptanceError("could not resolve the LLGo runtime.Panic entry")
    return breakpoint


def check_go_panic(
    executable: str,
    cwd: str,
    scenario: str,
    function: str,
    line: int,
) -> None:
    debugger, target = create_session(executable)
    process = lldb.SBProcess()
    try:
        set_panic_breakpoint(target)
        process = launch(target, scenario, cwd)
        top = process.GetSelectedThread().GetFrameAtIndex(0)
        if not frame_name(top).endswith("runtime.Panic"):
            raise AcceptanceError(
                f"{scenario} did not stop at the shared runtime.Panic entry:\n"
                f"{describe_process(process)}"
            )
        find_frame(process, function, "main.go", line)
    finally:
        destroy_session(debugger, process)


def check_optimized_inline(executable: str, cwd: str, root: Path) -> None:
    debugger, target = create_session(executable)
    process = lldb.SBProcess()
    try:
        leaf_line = marker_line(root / "inline.go", "LLDB_STOP: inline_leaf")
        breakpoint = target.BreakpointCreateByLocation("inline.go", leaf_line)
        if not breakpoint.IsValid() or breakpoint.GetNumLocations() == 0:
            raise AcceptanceError("optimized inline leaf has no source breakpoint")
        process = launch(target, "inline", cwd)
        leaf = find_frame(process, "main.inlineLeaf", "inline.go", leaf_line)
        middle = find_frame(
            process,
            "main.inlineMiddle",
            "inline.go",
            marker_line(root / "inline.go", "LLDB_STOP: inline_middle"),
        )
        caller = find_frame(
            process,
            "main.optimizedInlineCaller",
            "inline.go",
            marker_line(root / "inline.go", "LLDB_STOP: inline_caller"),
        )
        current_frames = frames(process)
        if not leaf < middle < caller:
            raise AcceptanceError(
                "optimized inline frames are not nested in source order:\n"
                f"{describe_process(process)}"
            )
        if not current_frames[leaf].IsInlined() or not current_frames[middle].IsInlined():
            raise AcceptanceError(
                "optimized leaf and middle are not represented as inline frames:\n"
                f"{describe_process(process)}"
            )
        if current_frames[caller].IsInlined():
            raise AcceptanceError(
                "the noinline caller unexpectedly has only an inline frame:\n"
                f"{describe_process(process)}"
            )

        after_line = marker_line(root / "inline.go", "LLDB_STOP: inline_after_call")
        for _ in range(8):
            process.GetSelectedThread().StepOver()
            require_stopped(process, "optimized inline step-over")
            top = process.GetSelectedThread().GetFrameAtIndex(0)
            source, line = frame_location(top)
            if (
                frame_name(top).endswith("main.optimizedInlineCaller")
                and source == "inline.go"
                and line == after_line
            ):
                break
        else:
            raise AcceptanceError(
                "step-over did not return from inline code to its caller's next line:\n"
                f"{describe_process(process)}"
            )
    finally:
        destroy_session(debugger, process)


def check_optimized_aggregate(executable: str, cwd: str, root: Path) -> None:
    debugger, target = create_session(executable)
    process = lldb.SBProcess()
    try:
        line = marker_line(root / "aggregate.go", "LLDB_STOP: aggregate_updated")
        breakpoint = target.BreakpointCreateByLocation("aggregate.go", line)
        if not breakpoint.IsValid() or breakpoint.GetNumLocations() == 0:
            raise AcceptanceError("optimized aggregate has no source breakpoint")
        process = launch(target, "aggregate", cwd)
        index = find_frame(process, "main.optimizedAggregate", "aggregate.go", line)
        frame = frames(process)[index]
        for name in ("parameter", "local"):
            value = frame.FindVariable(name)
            member = value.GetChildAtIndex(1)
            if not member.IsValid() or member.GetValueAsSigned(-1) != 12:
                raise AcceptanceError(f"optimized {name} lost its updated storage: {value}")
    finally:
        destroy_session(debugger, process)


def run_all(executable: str, optimized_executable: str, source_root: str) -> None:
    root = Path(source_root).resolve()
    cwd = str(root)
    panic_cases = (
        ("panic", "main.explicitPanic", "LLDB_STOP: explicit_panic"),
        ("divide", "main.divideByZero", "LLDB_STOP: divide_by_zero"),
        ("invalid-memory", "main.invalidMemory", "LLDB_STOP: invalid_memory"),
    )
    for scenario, function, marker in panic_cases:
        print(f"Native debug acceptance: {scenario}", flush=True)
        check_go_panic(
            executable,
            cwd,
            scenario,
            function,
            marker_line(root / "main.go", marker),
        )
    print("Native debug acceptance: optimized inline", flush=True)
    check_optimized_inline(optimized_executable, cwd, root)
    print("Native debug acceptance: optimized aggregate", flush=True)
    check_optimized_aggregate(optimized_executable, cwd, root)
    print("NATIVE_DEBUG_ACCEPTANCE_OK")


def main() -> None:
    run_all(
        os.environ["LLGO_NATIVE_DEBUG_ARTIFACT"],
        os.environ["LLGO_NATIVE_DEBUG_OPTIMIZED_ARTIFACT"],
        os.environ["LLGO_NATIVE_DEBUG_SOURCE"],
    )


if __name__ == "__main__":
    main()
