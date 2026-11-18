# Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
# Licensed under the Apache License, Version 2.0.

import json
from pathlib import Path
import traceback
import lldb
import llgo_wasi as adapter


def child(value, name):
    return next(v['value'] for v in value['children'] if v['name'] == name)


def frames(process, name):
    return [f for t in process for f in t if name == (f.GetFunctionName() or '').removeprefix('::').split('(')[0]]


def view(debugger, name, depth=2):
    return adapter.inspect(debugger, 'debugViews.' + name, depth)


def check(debugger):
    debugger.SetAsync(False)
    target = debugger.GetSelectedTarget()
    target.BreakpointCreateByName('llgo_debug_cpp_probe')
    process = target.LaunchSimple(None, None, str(Path.cwd()))
    assert process.IsValid() and process.GetState() == lldb.eStateStopped, process.GetState()
    thread = process.GetSelectedThread()
    cpp = thread.GetFrameAtIndex(0)
    assert 'llgo_debug_cpp_probe' in cpp.GetFunctionName(), cpp.GetFunctionName()
    assert cpp.FindVariable('input').GetValueAsSigned() == 14
    assert cpp.GetLineEntry().GetFileSpec().GetFilename() == 'probe.cpp'
    observed, steps = [], []
    for _ in range(5):
        current = thread.GetFrameAtIndex(0)
        observed.append(current.FindVariable('cpp_local').GetValueAsSigned())
        if 21 in observed and steps:
            break
        stop = process.GetStopID()
        thread.StepOver()
        steps.append((stop, process.GetStopID(), thread.GetStopDescription(256), str(thread.GetFrameAtIndex(0))))
    assert 21 in observed, (observed, steps)
    assert frames(process, 'main.main'), 'missing Go caller source stack'
    waiters = frames(process, 'main.debugWaiter')
    assert len(waiters) == 9, [f.GetFunctionName() for t in process for f in t]
    # Native all-stop also preserves every parked guest frame, without
    # borrowing Wasm registers from the currently selected worker.
    for frame in waiters:
        process.SetSelectedThread(frame.GetThread())
        frame.GetThread().SetSelectedFrame(frame.GetFrameID())
        depth = frame.FindVariable('depth').GetValueAsSigned()
        local = adapter.inspect(debugger, 'local', 1)
        values = [v['value']['value'] for v in local['children']]
        expected = [depth + i * 11 + (100 if i == depth & 3 else 0) for i in range(4)]
        assert values == expected, (depth, values, expected)
    process.SetSelectedThread(thread)
    thread.SetSelectedFrame(0)
    assert view(debugger, 'Text')['value'] == 'runtime 世界'
    nodes = view(debugger, 'Nodes', 3)
    assert child(child(nodes, '0'), 'Name')['value'] == 'runtime 世界'
    assert child(child(child(nodes, '0'), 'Values'), '0')['value'] == 3
    for name in ('Box', 'Pointer', 'Imported', 'ImportedPointer', 'Label'):
        value = view(debugger, name, 3)
        assert value['children'], (name, value)
        assert '<unavailable' not in json.dumps(value), (name, value)
    typed_nil = view(debugger, 'TypedNil')
    assert typed_nil['description'] == 'type=*main.debugNode', typed_nil
    assert child(typed_nil, 'value')['description'] == 'nil', typed_nil
    for name in ('Function', 'Closure', 'Bound', 'Nested', 'Counter', 'OtherCounter'):
        value = view(debugger, name, 2)
        assert 'func[' in value['description'], (name, value)
        if name in ('Closure', 'Nested', 'Counter', 'OtherCounter'):
            assert child(value, 'captures')['children'], (name, value)
    assert 'len=49' in view(debugger, 'Mapping', 0)['description']
    assert view(debugger, 'Indirect', 3)['children']
    queue = view(debugger, 'Queue', 3)
    assert 'len=3' in queue['description'] and 'closed' in queue['description'], queue
    assert view(debugger, 'Empty', 1)['children']
    goroutines = adapter.inspect(debugger, '$goroutines', 3)
    parked_ids = {g['name'] for g in goroutines['children']
                  if 'main.debugWaiter' in json.dumps(g)}
    assert len(parked_ids) == 2, goroutines
    first_stop = process.GetStopID()
    process.Continue()
    assert process.GetState() == lldb.eStateStopped and process.GetStopID() != first_stop
    assert not frames(process, 'main.debugWaiter'), 'workers did not exit'
    current = adapter.inspect(debugger, '$goroutines', 3)
    assert not parked_ids.intersection(g['name'] for g in current['children']), current
    assert child(child(child(view(debugger, 'Nodes', 3), '0'), 'Values'), '0')['value'] == 55
    assert 'len=48' in view(debugger, 'Mapping', 0)['description']
    assert 'len=0' in view(debugger, 'Queue', 0)['description']
    process.Continue()
    assert process.GetState() == lldb.eStateExited and process.GetExitStatus() == 0
    try:
        adapter.inspect(debugger, '$goroutines')
    except ValueError as error:
        assert 'all-stop' in str(error)
    else:
        raise AssertionError('inspection accepted an exited process')


def run(debugger, result_path):
    try:
        check(debugger)
        Path(result_path).write_text('ok', encoding='utf-8')
        print('wasi-source-runtime-debug-ok')
    except Exception:
        traceback.print_exc()
        Path(result_path).write_text('failed', encoding='utf-8')
        process = debugger.GetSelectedTarget().GetProcess()
        if process.IsValid() and process.GetState() != lldb.eStateExited:
            process.Kill()
