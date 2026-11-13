# Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
# Licensed under the Apache License, Version 2.0.

"""Strict native registry/main-stack acceptance, independent of worker unwind.

The complete worker-unwind acceptance lives in gdb_goroutines.py. This script
never claims worker-stack support merely because a thread ID is available.
"""

import gdb


def check_registry_and_main_stack():
    assert len(_llgo_last_stop) == 1 and isinstance(_llgo_last_stop[0], gdb.BreakpointEvent), (
        f"unexpected debugger stop: {_llgo_last_stop}")
    original = gdb.selected_thread()
    goroutines = _goroutines()
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
        print(f"LLGO_REGISTRY={item['goid']} tid={item['procid']} ptid={thread.ptid}")
    stack = gdb.execute("llgo goroutine 1 bt", to_string=True)
    print(f"LLGO_MAIN_STACK\n{stack}")
    for expected in ("main.InspectGoroutineValues", "main.RuntimeGoroutineValues", "main.main"):
        assert expected in stack, f"main stack lacks {expected}:\n{stack}"
    assert gdb.selected_thread() == original, "backtrace changed selected thread"
    print("LLGO_THREAD_PRESERVED=True")
    print("LLGO_GOROUTINE_REGISTRY=root+2workers")


check_registry_and_main_stack()
