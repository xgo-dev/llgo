# Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
# Licensed under the Apache License, Version 2.0.

import base64
import json
from pathlib import Path
import shlex
import shutil
import subprocess
import tempfile
import threading
import lldb

DIRECTORY = Path(__file__).resolve().parent
INDEX = json.loads((DIRECTORY / 'index.json').read_text(encoding='utf-8'))
SCHEMA = json.loads((DIRECTORY / 'schema.json').read_text(encoding='utf-8'))
TYPES = {value['id']: value for value in INDEX['types']}
FUNCTIONS = {}
VARIABLES = {}
for function in INDEX.get('functions') or []:
    FUNCTIONS.setdefault(function['name'], []).append(function)
for variable in INDEX.get('variables') or []:
    if variable['scope'] != 'GLOBAL':
        VARIABLES.setdefault(variable['name'], []).append(variable)
_BASE_CACHE = None


def _memory_base(process):
    global _BASE_CACHE
    key = (process.GetUniqueID(), process.GetStopID())
    if _BASE_CACHE is not None and _BASE_CACHE[0] == key:
        return _BASE_CACHE[1]
    spec = SCHEMA['runtime_layouts'][str(INDEX['record']['runtime_layout_version'])]['wasm_memory']
    offset = spec['anchor_offset']
    bases = set()
    for thread in process:
        for frame in thread:
            value = frame.FindVariable(spec['anchor_variable'])
            if value.IsValid() and value.GetError().Success():
                address = value.GetLoadAddress()
                if address != lldb.LLDB_INVALID_ADDRESS and address >= offset:
                    bases.add(address - offset)
    if len(bases) != 1:
        raise ValueError('no stable guest memory base; stop in an LLGo guest frame')
    base = bases.pop()
    _BASE_CACHE = (key, base)
    return base


def _local_request(frame, expression, base):
    value = frame.FindVariable(expression)
    if not value.IsValid():
        return {}  # The common decoder also resolves globals from DWARF.
    if value.GetError().Fail():
        raise ValueError('variable is unavailable: ' + str(value.GetError()))
    function = frame.GetFunctionName() or ''
    # LLDB may append a signature, including receiver parentheses. Match the
    # exact name and each possible signature boundary without scanning INDEX.
    names = [function] + [function[:i] for i, char in enumerate(function) if char == '(']
    functions = [f for name in names for f in FUNCTIONS.get(name, [])]
    candidates = [v for v in VARIABLES.get(expression, []) if any(
                      r['start'] < s['end'] and s['start'] < r['end']
                      for f in functions for r in f['ranges'] for s in v.get('ranges', []))]
    types = {v['type'] for v in candidates}
    if len(types) != 1:
        raise ValueError('local has no unambiguous module-owned DWARF type; use frame variable')
    request = {'type': types.pop()}
    address = value.GetLoadAddress()
    if address != lldb.LLDB_INVALID_ADDRESS and 0 <= address - base < 2**32:
        request['address'] = address - base
    else:
        pointer = value.GetChildMemberWithName('__ptr')
        if pointer.IsValid() and pointer.GetError().Success():
            request['value'] = str(pointer.GetValueAsUnsigned())
        elif value.GetValue() is not None:
            kind = TYPES[request['type']]
            visited = set()
            while kind['kind'] == 'typedef' and kind['id'] not in visited:
                visited.add(kind['id'])
                kind = TYPES[kind['elem']]
            if kind['kind'] not in ('integer', 'bool', 'pointer', 'enum'):
                raise ValueError('register value requires native frame variable inspection')
            request['value'] = str(value.GetValueAsUnsigned())
        else:
            raise ValueError('variable is unavailable outside linear memory; use frame variable')
    return request


def inspect(debugger, expression, depth=2):
    """Return a fresh view; no object handles survive a resume or memory growth."""
    if not 0 <= depth <= 8:
        raise ValueError('depth must be between 0 and 8')
    process = debugger.GetSelectedTarget().GetProcess()
    if not process.IsValid() or process.GetState() != lldb.eStateStopped:
        raise ValueError('WASI runtime inspection requires an all-stop process')
    stop = process.GetStopID()
    identity = process.GetUniqueID()
    base = _memory_base(process)
    node = shutil.which('node')
    if not node:
        raise ValueError('WASI runtime views require Node.js 22 or newer')
    request = {'expression': expression, 'depth': depth, 'stopId': str(stop)}
    frame = process.GetSelectedThread().GetSelectedFrame()
    if expression != '$goroutines':
        request.update(_local_request(frame, expression, base))
    def stable():
        if (process.GetState() != lldb.eStateStopped or process.GetStopID() != stop or
                process.GetUniqueID() != identity):
            raise ValueError('process resumed or stop changed; inspect the current stop again')
    # A pipe which is only read on failure can fill and deadlock a decoder
    # writing diagnostics. Keep stderr separate from the stdout RPC stream.
    with tempfile.TemporaryFile(mode='w+', encoding='utf-8', errors='replace') as errors, \
            subprocess.Popen([node, str(DIRECTORY / 'decoder.js')], stdin=subprocess.PIPE,
                             stdout=subprocess.PIPE, stderr=errors, text=True) as child:
        def diagnostic():
            errors.seek(0)
            return errors.read(65536)

        timer = threading.Timer(30, child.kill)
        timer.start()
        reads, total = 0, 0
        try:
            child.stdin.write(json.dumps(request) + '\n')
            child.stdin.flush()
            for line in child.stdout:
                stable()
                message = json.loads(line)
                if 'error' in message:
                    raise ValueError(message['error'])
                if 'result' in message:
                    result = message['result']
                    break
                read = message['read']
                address, size = read['address'], read['length']
                reads, total = reads + 1, total + size
                if (not isinstance(address, int) or not isinstance(size, int) or address < 0 or
                        size < 0 or size > 1024 * 1024 or address + size > 2**32 or
                        read['stopId'] != str(stop) or reads > 16384 or total > 32 * 1024 * 1024):
                    raise ValueError('invalid or oversized WASI memory request')
                error = lldb.SBError()
                data = process.ReadMemory(base + address, size, error) if size else b''
                stable()
                response = ({'bytes': base64.b64encode(data).decode('ascii')}
                            if error.Success() and data is not None and len(data) == size
                            else {'error': 'guest memory is unreadable: ' + str(error)})
                child.stdin.write(json.dumps(response) + '\n')
                child.stdin.flush()
            else:
                raise ValueError('WASI value decoder failed or timed out: ' + diagnostic())
            child.wait(timeout=5)
            if child.returncode:
                raise ValueError('WASI value decoder failed: ' + diagnostic())
            stable()
            return result
        finally:
            timer.cancel()
            if child.poll() is None:
                child.kill()
                child.wait()


def command(debugger, arguments, result, _internal_dict):
    try:
        args = shlex.split(arguments)
        depth = 2
        if len(args) >= 2 and args[-2] == '-depth':
            depth = int(args[-1])
            args = args[:-2]
        if len(args) == 1 and args[0] == 'goroutines':
            expression = '$goroutines'
        elif len(args) == 2 and args[0] == 'print':
            expression = args[1]
        else:
            raise ValueError('usage: llgo-wasi print NAME|goroutines [-depth 0..8]')
        result.AppendMessage(json.dumps(inspect(debugger, expression, depth), ensure_ascii=False, indent=2))
    except (ValueError, OSError, KeyError, subprocess.SubprocessError) as error:
        result.SetError(str(error))


def __lldb_init_module(debugger, _internal_dict):
    debugger.HandleCommand('command script add -f llgo_wasi.command llgo-wasi')
