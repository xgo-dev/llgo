# Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
# Licensed under the Apache License, Version 2.0.

"""Run inside the actual GDB process after InspectGoroutineValues stops.

Each live worker must map to a distinct native thread and unwind through its
own application closure. Seeing the parent's similarly named frame alone is
not evidence that worker stacks work on a particular OS or architecture.
"""

import gdb


def check_goroutine_stacks():
    assert len(_llgo_last_stop) == 1 and isinstance(_llgo_last_stop[0], gdb.BreakpointEvent), (
        f"unexpected debugger stop: {_llgo_last_stop}")
    original = gdb.selected_thread()
    goroutines = _goroutines()  # The production adapter is loaded by llgo gdb.
    root = [item for item in goroutines if item["goid"] == 1]
    # Keep the runtime service visible in the registry and verify its native
    # stack before excluding it from this fixture's two application workers.
    workers = []
    for item in goroutines:
        if item["parent"] != 1:
            continue
        assert item["procid"] > 0 and item["thread"] is not None, item
        stack = gdb.execute(f"llgo goroutine {item['goid']} bt", to_string=True)
        assert gdb.selected_thread() == original, "backtrace changed selected thread"
        if "runtime.runExplicitGC" not in stack:
            workers.append(item)
    assert len(root) == 1, f"expected main goroutine: {goroutines}"
    assert len(workers) == 2, f"expected two live workers: {goroutines}"
    seen_threads = set()
    for item in root + workers:
        thread = item["thread"]
        assert item["procid"] > 0 and thread is not None, (
            f"goroutine has no native thread: {item}")
        assert thread.num not in seen_threads, f"duplicate native thread: {item}"
        seen_threads.add(thread.num)
        stack = gdb.execute(f"llgo goroutine {item['goid']} bt", to_string=True)
        print(f"LLGO_STACK={item['goid']} ptid={thread.ptid}\n{stack}")
        assert gdb.selected_thread() == original, "backtrace changed selected thread"
        expected = ("main.InspectGoroutineValues" if item["goid"] == 1
                    else "main.RuntimeGoroutineValues$1")
        assert expected in stack, f"goroutine {item['goid']} lacks {expected}:\n{stack}"
    print("LLGO_THREAD_PRESERVED=True")
    print("LLGO_GOROUTINE_STACKS=root+2workers")


check_goroutine_stacks()
