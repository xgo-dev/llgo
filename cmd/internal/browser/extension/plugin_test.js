const assert = require('node:assert/strict');
const test = require('node:test');
const debuggerSchema = require('../../../../internal/debugabi/schema_v1.json');

require('./plugin.js');
require('./goroutines.js');

const {LLGoLanguageExtensionPlugin} = globalThis.LLGoLanguageExtension;

function uleb(value) {
  const result = [];
  do {
    let current = value & 0x7f;
    value >>>= 7;
    if (value) current |= 0x80;
    result.push(current);
  } while (value);
  return result;
}

function custom(name, payload) {
  const nameBytes = [...new TextEncoder().encode(name)];
  const contents = [...uleb(nameBytes.length), ...nameBytes, ...payload];
  return [0, ...uleb(contents.length), ...contents];
}

function moduleBytes({marker = true, id = [1, 2, 3, 4], pointerSize = 4} = {}) {
  const result = [0, 0x61, 0x73, 0x6d, 1, 0, 0, 0];
  if (marker) {
    result.push(...custom('llgo.debugger', [
      0x4c, 0x4c, 0x47, 0x4f, 0x44, 0x42, 0x47, 0,
      1, 1, 2, 1, 0, pointerSize, 1, 0,
    ]));
  }
  result.push(...custom('build_id', [...uleb(id.length), ...id]));
  return new Uint8Array(result);
}

function fixtureIndex() {
  return {
    contract: 'llgo.browser.debug', version: 1, build_id: '01020304', artifact: 'embedded',
    record: {
      record_version: 1, schema_version: 1, runtime_layout_version: 2,
      llgo_abi_version: 1, pointer_size: 4, byte_order: 1,
    },
    sources: [{id: 'source', path: '/src/main.go', url: '/__llgo/source/source', local: true}],
    lines: [{source: 'source', line: 7, column: 1, start: 10, end: 20}],
    functions: [{name: 'main.main', ranges: [{start: 10, end: 20}]}],
    variables: [
      {name: 'constant', scope: 'LOCAL', type: 'int32', depth: 2, ranges: [{start: 10, end: 20}],
       constant: {kind: 'signed', value: '41'}},
      {name: 'local', scope: 'LOCAL', type: 'int32', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: 'ed00009f'}]},
    ],
    types: [{id: 'int32', name: 'int32', kind: 'integer', size: 4, signed: true, complete: true}],
  };
}

const schema = {
  schema_version: 1, runtime_layout_version: 2, llgo_abi_version: 1,
  runtime_layouts: {'2': {
    string: {type_name: 'string', data: 'data', length: 'len'},
    slice: {type_pattern: '^\\[\\].+', data: 'data', length: 'len', capacity: 'cap'},
  }},
};

test('coordinated inspection rejects changed stops and stale expanded objects', async () => {
  const index = {...fixtureIndex(), worker_control: true};
  let state = {enabled: true, ready: true, paused: true, stopId: 7};
  const plugin = new LLGoLanguageExtensionPlugin({
    getWasmLocal: async () => {
      state = {...state, stopId: 8};
      return {type: 'i32', value: 42};
    },
  }, async url => {
    if (url.endsWith('/debug-index.json')) return response(index);
    if (url.endsWith('/debug-schema.json')) return response(schema);
    if (url.endsWith('/worker-state')) return response(state);
    if (url.endsWith('/plugin-ready')) return new Response(null, {status: 204});
    throw new Error(`unexpected URL ${url}`);
  });
  await plugin.addRawModule('module', undefined, {
    url: 'http://127.0.0.1:1234/program.wasm', code: moduleBytes().buffer,
  });
  const context = {rawModuleId: 'module', codeOffset: 12, inlineFrameIndex: 0};
  assert.equal((await plugin.evaluate('constant', context, 1n)).value, 41);
  await assert.rejects(plugin.evaluate('local', context, 1n), /resumed or changed/);
  const module = plugin.module('module');
  await assert.rejects(plugin.withWorkerStop(module, async () => {
    const value = plugin.snapshotObject(module, {type: 'object'}, []);
    state = {...state, stopId: state.stopId + 1};
    return value;
  }), /resumed or changed/);
  assert.equal(plugin.objects.size, 0, 'rejected inspection releases unpublished handles');
  const object = await plugin.withWorkerStop(module, async () =>
    plugin.snapshotObject(module, {type: 'object'}, [{name: 'id', value: {type: 'number', value: 1}}]));
  state = {...state, paused: false, stopId: 9};
  await assert.rejects(plugin.getProperties(object.objectId), /resumed or changed/);
  state = {...state, error: 'worker transport disconnected'};
  await assert.rejects(plugin.evaluate('constant', context, 1n), /transport disconnected/);
});

function response(value, init) {
  return new Response(typeof value === 'string' ? value : JSON.stringify(value), init);
}

for (const width of [4, 8]) {
test(`closure captures preserve reference and value storage (${width} bytes)`, async () => {
  const index = fixtureIndex();
  index.record.pointer_size = width;
  index.function_values = [{index: 7, name: 'main.capture$1', environment: 'envptr'},
    {index: 8, name: 'main.opaque', environment: 'word'},
    {index: 9, name: 'main.incomplete', environment: 'missingptr'}];
  index.types.push(
    {id: 'word', kind: 'integer', name: 'uintptr', size: width, complete: true},
    {id: 'intptr', kind: 'pointer', name: '*int32', size: width, elem: 'int32', complete: true},
    {id: 'envptr', kind: 'pointer', size: width, elem: 'env', complete: true},
    {id: 'missingptr', kind: 'pointer', size: width, elem: 'missing', complete: true},
    {id: 'missing', kind: 'struct', size: width, complete: false},
    {id: 'env', kind: 'struct', name: 'captures', size: width + 4, complete: true,
      fields: [{name: 'reference', type: 'intptr', offset: 0}, {name: 'value', type: 'int32', offset: width}]},
    {id: 'fn', kind: 'struct', name: 'struct{$f func(); $data unsafe.Pointer}', size: width * 2, complete: true,
      fields: [{name: '$f', type: 'word', offset: 0}, {name: '$data', type: 'word', offset: width}]},
  );
  const memory = new Uint8Array(256), view = new DataView(memory.buffer);
  const word = (at, value) => width === 4 ? view.setUint32(at, value, true) : view.setBigUint64(at, BigInt(value), true);
  word(16, 7); word(16 + width, 128); word(128, 192);
  view.setInt32(128 + width, 23, true); view.setInt32(192, 41, true);
  const plugin = new LLGoLanguageExtensionPlugin({
    getWasmLinearMemory: async (offset, length) => memory.slice(offset, offset + length).buffer,
  }, async url => {
    if (url.endsWith('/debug-index.json')) return response(index);
    if (url.endsWith('/debug-schema.json')) return response(debuggerSchema);
    if (url.endsWith('/plugin-ready')) return new Response(null, {status: 204});
    throw new Error(`unexpected URL ${url}`);
  });
  await plugin.addRawModule('module', undefined, {
    url: 'http://host/closure.wasm', code: moduleBytes({pointerSize: width}).buffer,
  });
  const module = plugin.module('module');
  const read = () => plugin.remoteObject(module, module.types.get('fn'), {kind: 'address', value: 16n}, 'stop');
  for (const value of [41, 52]) {
    view.setInt32(192, value, true);
    const fn = await read();
    assert.equal(fn.description, 'func[7] main.capture$1 data=0x80');
    const captures = (await plugin.getProperties(fn.objectId)).find(p => p.name === 'captures').value;
    const fields = Object.fromEntries((await plugin.getProperties(captures.objectId)).map(p => [p.name, p.value]));
    assert.equal(fields.value.value, 23);
    assert.equal((await plugin.getProperties(fields.reference.objectId))[0].value.value, value);
    await plugin.releaseObject(fn.objectId);
    assert.equal(plugin.objects.size, 0, 'release includes captured pointers');
  }
  for (const code of [8, 9, 99]) {
    word(16, code);
    const fn = await read();
    assert.ok(!(await plugin.getProperties(fn.objectId)).some(p => p.name === 'captures'),
      'opaque, incomplete and unknown functions have no guessed capture layout');
    await plugin.releaseObject(fn.objectId);
  }
  word(16, 7); word(16 + width, 0);
  const empty = await read();
  assert.ok(!(await plugin.getProperties(empty.objectId)).some(p => p.name === 'captures'));
  await plugin.releaseObject(empty.objectId);
  word(16, 0);
  assert.equal((await read()).type, 'null');
});
}

test('LLGo language extension maps DWARF index and evaluates constants and Wasm locals', async () => {
  const calls = [];
  const services = {
    getWasmLocal: async (index, stopId) => {
      calls.push(['local', index, stopId]);
      return {type: 'i32', value: 42};
    },
    getWasmGlobal: async () => { throw new Error('unexpected global'); },
    getWasmOp: async () => { throw new Error('unexpected operand'); },
    getWasmLinearMemory: async () => { throw new Error('unexpected memory'); },
  };
  const fetcher = async (url, options = {}) => {
    if (String(url).endsWith('/__llgo/debug-index.json')) return response(fixtureIndex());
    if (String(url).endsWith('/__llgo/debug-schema.json')) return response(schema);
    if (String(url).endsWith('/__llgo/plugin-ready')) {
      calls.push(['ready']);
      return new Response(null, {status: 204});
    }
    throw new Error(`unexpected URL ${url}`);
  };
  const plugin = new LLGoLanguageExtensionPlugin(services, fetcher);
  const rawModule = {url: 'http://127.0.0.1:1234/program.wasm', code: moduleBytes().buffer};
  const sources = await plugin.addRawModule('module', undefined, rawModule);
  assert.deepEqual(sources, ['http://127.0.0.1:1234/__llgo/source/source']);
  assert.deepEqual(await plugin.sourceLocationToRawLocation({
    rawModuleId: 'module', sourceFileURL: sources[0], lineNumber: 7, columnNumber: 0,
  }), [{rawModuleId: 'module', startOffset: 10, endOffset: 20}]);
  assert.deepEqual(await plugin.rawLocationToSourceLocation({
    rawModuleId: 'module', codeOffset: 12, inlineFrameIndex: 0,
  }), [{rawModuleId: 'module', sourceFileURL: sources[0], lineNumber: 7, columnNumber: 1}]);
  assert.deepEqual(await plugin.getMappedLines('module', sources[0]), [7]);
  assert.deepEqual(await plugin.getFunctionInfo({rawModuleId: 'module', codeOffset: 12, inlineFrameIndex: 0}), {
    frames: [{name: 'main.main'}], missingSymbolFiles: [],
  });
  assert.deepEqual((await plugin.listVariablesInScope({rawModuleId: 'module', codeOffset: 12})).map(v => v.name),
                   ['constant', 'local']);
  assert.deepEqual(await plugin.evaluate('constant', {rawModuleId: 'module', codeOffset: 12}, 'stop'), {
    type: 'number', value: 41, description: '41', hasChildren: false,
  });
  assert.deepEqual(await plugin.evaluate('local', {rawModuleId: 'module', codeOffset: 12}, 'stop'), {
    type: 'number', value: 42, description: '42', hasChildren: false,
  });
  assert.deepEqual(calls, [['ready'], ['local', 0, 'stop']]);
});

test('LLGo language extension reports missing symbols and ignores non-LLGo modules', async () => {
  const services = {};
  const fetcher = async url => {
    if (String(url).endsWith('/__llgo/debug-index.json')) {
      return response({missing_symbol_files: ['http://host/missing.wasm']}, {status: 424});
    }
    throw new Error(`unexpected URL ${url}`);
  };
  const plugin = new LLGoLanguageExtensionPlugin(services, fetcher);
  const missing = await plugin.addRawModule('missing', 'http://host/missing.wasm', {
    url: 'http://host/program.wasm', code: moduleBytes().buffer,
  });
  assert.deepEqual(missing, {missingSymbolFiles: ['http://host/missing.wasm']});

  const ignored = await plugin.addRawModule('plain', undefined, {
    url: 'http://host/plain.wasm', code: moduleBytes({marker: false}).buffer,
  });
  assert.deepEqual(ignored, []);
});

test('LLGo language extension presents strings, slices, aggregates, and children from linear memory', async () => {
  const index = fixtureIndex();
  index.types.push(
      {id: 'uint32', name: 'uint32', kind: 'integer', size: 4, complete: true},
      {id: 'int32ptr', name: '*int32', kind: 'pointer', size: 4, elem: 'int32', complete: true},
      {id: 'string', name: 'string', kind: 'struct', size: 8, complete: true,
       fields: [{name: 'data', type: 'int32ptr', offset: 0}, {name: 'len', type: 'uint32', offset: 4}]},
      {id: 'slice', name: '[]int32', kind: 'struct', size: 12, complete: true,
       fields: [{name: 'data', type: 'int32ptr', offset: 0}, {name: 'len', type: 'uint32', offset: 4},
                {name: 'cap', type: 'uint32', offset: 8}]},
      {id: 'pair', name: 'main.Pair', kind: 'struct', size: 8, complete: true,
       fields: [{name: 'Left', type: 'int32', offset: 0}, {name: 'Right', type: 'int32', offset: 4}]},
  );
  index.variables.push(
      {name: 'text', scope: 'LOCAL', type: 'string', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0310000000'}]},
      {name: 'values', scope: 'LOCAL', type: 'slice', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0320000000'}]},
      {name: 'pair', scope: 'LOCAL', type: 'pair', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0340000000'}]},
  );
  const memory = new Uint8Array(256);
  const view = new DataView(memory.buffer);
  view.setUint32(16, 100, true);
  view.setUint32(20, 3, true);
  memory.set(new TextEncoder().encode('abc'), 100);
  view.setUint32(32, 120, true);
  view.setUint32(36, 2, true);
  view.setUint32(40, 3, true);
  view.setInt32(64, 9, true);
  view.setInt32(68, 10, true);
  view.setInt32(120, 7, true);
  view.setInt32(124, 8, true);
  const services = {
    getWasmLinearMemory: async (offset, length) => memory.slice(offset, offset + length).buffer,
  };
  const fetcher = async url => {
    if (String(url).endsWith('/__llgo/debug-index.json')) return response(index);
    if (String(url).endsWith('/__llgo/debug-schema.json')) return response(schema);
    if (String(url).endsWith('/__llgo/plugin-ready')) return new Response(null, {status: 204});
    throw new Error(`unexpected URL ${url}`);
  };
  const plugin = new LLGoLanguageExtensionPlugin(services, fetcher);
  const context = {rawModuleId: 'module', codeOffset: 12};
  await plugin.addRawModule('module', undefined, {
    url: 'http://127.0.0.1:1234/program.wasm', code: moduleBytes().buffer,
  });
  assert.deepEqual(await plugin.evaluate('text', context, 'stop'), {
    type: 'string', className: 'string', value: 'abc', description: '"abc"', hasChildren: false,
    linearMemoryAddress: 100, linearMemorySize: 3,
  });
  const values = await plugin.evaluate('values', context, 'stop');
  assert.equal(values.description, '[]int32 len=2 cap=3');
  assert.deepEqual((await plugin.getProperties(values.objectId)).map(item => [item.name, item.value.value]),
                   [['0', 7], ['1', 8]]);
  assert.deepEqual(await plugin.evaluate('pair.Right', context, 'stop'), {
    type: 'number', value: 10, description: '10', hasChildren: false,
  });
});

test('LLGo language extension rejects stale browser indexes', async () => {
  const index = fixtureIndex();
  index.build_id = 'ffffffff';
  const fetcher = async url => {
    if (String(url).endsWith('/__llgo/debug-index.json')) return response(index);
    throw new Error(`unexpected URL ${url}`);
  };
  const plugin = new LLGoLanguageExtensionPlugin({}, fetcher);
  await assert.rejects(plugin.addRawModule('stale', undefined, {
    url: 'http://host/program.wasm', code: moduleBytes().buffer,
  }), /build_id mismatch/);
});

for (const structPrefix of ['', 'struct ']) {
test(`LLGo language extension presents runtime layouts with struct prefix ${JSON.stringify(structPrefix)}`, async () => {
  const index = fixtureIndex();
  index.types.push(
      {id: 'u8', name: 'uint8', kind: 'integer', size: 1, complete: true},
      {id: 'u32', name: 'uint32', kind: 'integer', size: 4, complete: true},
      {id: 'u64', name: 'uint64', kind: 'integer', size: 8, complete: true},
      {id: 'ptrInt', name: '*int32', kind: 'pointer', size: 4, elem: 'int32', complete: true},
      {id: 'string', name: 'string', kind: 'struct', size: 8, complete: true,
       fields: [{name: 'data', type: 'ptrInt', offset: 0}, {name: 'len', type: 'u32', offset: 4}]},
      {id: 'runtimeType', name: 'github.com/xgo-dev/llgo/runtime/abi.Type', kind: 'struct', size: 12,
       complete: true, fields: [{name: 'TFlag', type: 'u8', offset: 0}, {name: 'Kind_', type: 'u8', offset: 1}, {name: 'Str_', type: 'string', offset: 4}]},
      {id: 'ptrRuntimeType', name: '*github.com/xgo-dev/llgo/runtime/abi.Type', kind: 'pointer', size: 4,
       elem: 'runtimeType', complete: true},
      {id: 'itab', name: 'github.com/xgo-dev/llgo/runtime/internal/runtime.itab', kind: 'struct', size: 4,
       complete: true, fields: [{name: '_type', type: 'ptrRuntimeType', offset: 0}]},
      {id: 'ptrItab', name: '*github.com/xgo-dev/llgo/runtime/internal/runtime.itab', kind: 'pointer', size: 4,
       elem: 'itab', complete: true},
      {id: 'emptyInterface', name: 'interface{}', kind: 'struct', size: 8, complete: true,
       fields: [{name: 'type', type: 'ptrRuntimeType', offset: 0}, {name: 'data', type: 'ptrInt', offset: 4}]},
      {id: 'namedAny', name: 'main.Any', kind: 'typedef', size: 8, elem: 'emptyInterface', complete: true},
      {id: 'pointerBox', name: 'main.PointerBox', kind: 'struct', size: 4, complete: true,
       fields: [{name: 'Pointer', type: 'ptrInt', offset: 0}]},
      {id: 'nonemptyInterface', name: 'interface{Value() int32}', kind: 'struct', size: 8, complete: true,
       fields: [{name: 'type', type: 'ptrItab', offset: 0}, {name: 'data', type: 'ptrInt', offset: 4}]},
      {id: 'funcval', name: 'struct{$f func(); $data unsafe.Pointer}', kind: 'struct', size: 8,
       complete: true, fields: [{name: '$f', type: 'u32', offset: 0}, {name: '$data', type: 'ptrInt', offset: 4}]},
      {id: 'topArray', name: '[8]uint8', kind: 'array', size: 8, elem: 'u8', count: 8, complete: true},
      {id: 'keyArray', name: '[8]int32', kind: 'array', size: 32, elem: 'int32', count: 8, complete: true},
      {id: 'valueArray', name: '[8]int32', kind: 'array', size: 32, elem: 'int32', count: 8, complete: true},
      {id: 'bucket', name: 'bucket<int32,int32>', kind: 'struct', size: 76, complete: true,
       fields: [{name: 'tophash', type: 'topArray', offset: 0}, {name: 'keys', type: 'keyArray', offset: 8},
                {name: 'values', type: 'valueArray', offset: 40}, {name: 'overflow', type: 'ptrBucket', offset: 72}]},
      {id: 'ptrBucket', name: '*bucket<int32,int32>', kind: 'pointer', size: 4, elem: 'bucket', complete: true},
      {id: 'hash', name: 'hash<int32,int32>', kind: 'struct', size: 16, complete: true,
       fields: [{name: 'count', type: 'u32', offset: 0}, {name: 'flags', type: 'u8', offset: 4},
                {name: 'B', type: 'u8', offset: 5}, {name: 'buckets', type: 'ptrBucket', offset: 8},
                {name: 'oldbuckets', type: 'ptrBucket', offset: 12}]},
      {id: 'map', name: 'map[int32]int32', kind: 'pointer', size: 4, elem: 'hash', complete: true},
      {id: 'waiter', name: 'sudog<int32>', kind: 'struct', size: 4, complete: true,
       fields: [{name: 'elem', type: 'ptrInt', offset: 0}]},
      {id: 'ptrWaiter', name: '*sudog<int32>', kind: 'pointer', size: 4, elem: 'waiter', complete: true},
      {id: 'waitq', name: 'waitq<int32>', kind: 'struct', size: 4, complete: true,
       fields: [{name: 'first', type: 'ptrWaiter', offset: 0}]},
      {id: 'hchan', name: 'hchan<int32>', kind: 'struct', size: 24, complete: true,
       fields: [{name: 'qcount', type: 'u32', offset: 0}, {name: 'dataqsiz', type: 'u32', offset: 4},
                {name: 'buf', type: 'ptrInt', offset: 8}, {name: 'closed', type: 'u32', offset: 12},
                {name: 'recvx', type: 'u32', offset: 16}, {name: 'recvq', type: 'waitq', offset: 20}]},
      {id: 'chan', name: 'chan int32', kind: 'pointer', size: 4, elem: 'hchan', complete: true},
      {id: 'g', name: 'github.com/xgo-dev/llgo/runtime/internal/runtime.g', kind: 'struct', size: 24,
       complete: true, fields: [{name: 'alllink', type: 'ptrG', offset: 0},
                               {name: 'atomicstatus', type: 'u32', offset: 4},
                               {name: 'goid', type: 'u64', offset: 8},
                               {name: 'parentGoid', type: 'u64', offset: 16}]},
      {id: 'ptrG', name: '*github.com/xgo-dev/llgo/runtime/internal/runtime.g', kind: 'pointer', size: 4,
       elem: 'g', complete: true},
  );
  index.variables.push(
      {name: 'mapping', scope: 'LOCAL', type: 'map', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0350000000'}]},
      {name: 'queue', scope: 'LOCAL', type: 'chan', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0354000000'}]},
      {name: 'dynamic', scope: 'LOCAL', type: 'emptyInterface', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0358000000'}]},
      {name: 'pointerAny', scope: 'LOCAL', type: 'emptyInterface', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0378000000'}]},
      {name: 'typedNil', scope: 'LOCAL', type: 'emptyInterface', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0380000000'}]},
      {name: 'namedAny', scope: 'LOCAL', type: 'namedAny', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0388000000'}]},
      {name: 'boxedPointer', scope: 'LOCAL', type: 'emptyInterface', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0390000000'}]},
      {name: 'callback', scope: 'LOCAL', type: 'funcval', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0360000000'}]},
      {name: 'nonempty', scope: 'LOCAL', type: 'nonemptyInterface', depth: 2, ranges: [{start: 10, end: 20}],
       locations: [{expression: '0370000000'}]},
      {name: 'github.com/xgo-dev/llgo/runtime/internal/runtime.debuggerAllgV1', scope: 'GLOBAL', type: 'ptrG', depth: 0,
       locations: [{expression: '0368000000'}]},
  );

  for (const type of index.types) {
    if (type.kind === 'struct') type.name = structPrefix + type.name;
  }

  const memory = new Uint8Array(1024);
  const view = new DataView(memory.buffer);
  const u32 = (address, value) => view.setUint32(address, value, true);
  const u64 = (address, value) => view.setBigUint64(address, BigInt(value), true);
  u32(80, 160); // map header pointer
  u32(84, 300); // channel header pointer
  u32(88, 500); u32(92, 520); // empty interface type/data
  u32(96, 10); u32(100, 123); // function code/data
  u32(104, 600); // debuggerAllgV1
  u32(112, 560); u32(116, 520); // non-empty interface itab/data
  u32(160, 2); u32(168, 200); // hmap count, buckets
  memory[200] = 5; memory[201] = 6;
  view.setInt32(208, 1, true); view.setInt32(212, 2, true);
  view.setInt32(240, 11, true); view.setInt32(244, 22, true);
  u32(300, 2); u32(304, 3); u32(308, 400); u32(312, 0); u32(316, 1);
  view.setInt32(400, 10, true); view.setInt32(404, 11, true); view.setInt32(408, 12, true);
  u32(504, 550); u32(508, 5); memory.set(new TextEncoder().encode('int32'), 550);
  view.setInt32(520, 42, true);
  u32(560, 500);
  // Direct-interface words contain the pointer, not a pointer to its storage.
  u32(120, 700); u32(124, 520);
  u32(128, 700); u32(132, 0); // non-nil interface with a typed nil pointer
  u32(136, 500); u32(140, 520); // named empty interface, still a type word
  u32(144, 760); u32(148, 520); // one-pointer struct stored directly
  memory[701] = 32;
  u32(704, 740); u32(708, 6); memory.set(new TextEncoder().encode('*int32'), 740);
  memory[761] = 32;
  u32(764, 780); u32(768, 15); memory.set(new TextEncoder().encode('main.PointerBox'), 780);
  u32(600, 640); u32(604, 2); u64(608, 1); u64(616, 0);
  u32(640, 0); u32(644, 1); u64(648, 2); u64(656, 1);

  const services = {
    getWasmLinearMemory: async (offset, length) => memory.slice(offset, offset + length).buffer,
  };
  const fetcher = async url => {
    if (String(url).endsWith('/__llgo/debug-index.json')) return response(index);
    if (String(url).endsWith('/__llgo/debug-schema.json')) return response(debuggerSchema);
    if (String(url).endsWith('/__llgo/plugin-ready')) return new Response(null, {status: 204});
    throw new Error(`unexpected URL ${url}`);
  };
  const plugin = new LLGoLanguageExtensionPlugin(services, fetcher);
  const context = {rawModuleId: 'module', codeOffset: 12};
  await plugin.addRawModule('module', undefined, {
    url: 'http://127.0.0.1:1234/program.wasm', code: moduleBytes().buffer,
  });

  const dynamic = await plugin.evaluate('dynamic', context, 'stop');
  assert.equal(dynamic.description, 'type=int32');
  assert.equal((await plugin.getProperties(dynamic.objectId))[0].value.value, 42);
  assert.equal((await plugin.evaluate('nonempty', context, 'stop')).description, 'type=int32');
  const pointerAny = await plugin.evaluate('pointerAny', context, 'stop');
  assert.equal(pointerAny.description, 'type=*int32');
  const pointerValue = (await plugin.getProperties(pointerAny.objectId))[0].value;
  assert.equal(pointerValue.description, '0x208');
  assert.equal((await plugin.getProperties(pointerValue.objectId))[0].value.value, 42);
  const typedNil = await plugin.evaluate('typedNil', context, 'stop');
  assert.equal(typedNil.description, 'type=*int32');
  assert.equal(typedNil.hasChildren, true);
  assert.equal((await plugin.getProperties(typedNil.objectId))[0].value.description, 'nil');
  const namedAny = await plugin.evaluate('namedAny', context, 'stop');
  assert.equal(namedAny.description, 'type=int32');
  assert.equal((await plugin.getProperties(namedAny.objectId))[0].value.value, 42);
  const boxedPointer = await plugin.evaluate('boxedPointer', context, 'stop');
  const boxedValue = (await plugin.getProperties(boxedPointer.objectId))[0].value;
  assert.equal((await plugin.getProperties(boxedValue.objectId))[0].value.description, '0x208');
  assert.equal((await plugin.evaluate('callback', context, 'stop')).description, 'func[10] data=0x7b');
  const mapping = await plugin.evaluate('mapping', context, 'stop');
  assert.equal(mapping.description, 'len=2');
  assert.deepEqual((await plugin.getProperties(mapping.objectId)).map(item => [item.name, item.value.value]),
                   [['key[0]', 1], ['value[0]', 11], ['key[1]', 2], ['value[1]', 22]]);
  const queue = await plugin.evaluate('queue', context, 'stop');
  assert.equal(queue.description, 'len=2 cap=3');
  assert.deepEqual((await plugin.getProperties(queue.objectId)).map(item => item.value.value), [11, 12]);
  assert.ok(!(await plugin.listVariablesInScope(context)).some(variable => variable.name === '$goroutines'));
  assert.match((await plugin.evaluate('$goroutines', context, 'stop')).value, /build with llgo.wasm.debugger/);
});
}


for (const width of [4, 8]) {
test(`interface type lookup uses runtime import paths and preserves pointer identity (${width} bytes)`, async () => {
  const prefix = 'github.com/xgo-dev/llgo/runtime/abi.';
  const types = [
    {id: 'byte', kind: 'integer', size: 1},
    {id: 'word', kind: 'integer', size: width},
    {id: 'string', name: 'struct string', kind: 'struct', size: 2 * width,
      fields: [{name: 'data', type: 'word', offset: 0}, {name: 'len', type: 'word', offset: width}]},
    {id: 'metadata', name: prefix + 'Type', kind: 'struct', size: 3 * width,
      fields: [{name: 'TFlag', type: 'byte', offset: 0}, {name: 'Kind_', type: 'byte', offset: 1},
        {name: 'Str_', type: 'string', offset: width}]},
    {id: 'pointerMetadata', name: prefix + 'PtrType', kind: 'struct', size: 4 * width,
      fields: [{name: 'Elem', type: 'word', offset: 3 * width}]},
    {id: 'uncommon', name: prefix + 'UncommonType', kind: 'struct', size: 2 * width,
      fields: [{name: 'PkgPath_', type: 'string', offset: 0}]},
    {id: 'int', name: 'int32', kind: 'integer', size: 4, signed: true},
    {id: 'actual', name: 'example.com/one.Record', kind: 'typedef', elem: 'int', size: 4},
    {id: 'pointer', name: '*example.com/one.Record', kind: 'pointer', elem: 'actual', size: width},
    {id: 'decoy', name: 'pkg.Record', kind: 'integer', size: 1},
    {id: 'other', name: 'example.com/two.Record', kind: 'integer', size: 1},
  ];
  const memory = new Uint8Array(2048);
  const view = new DataView(memory.buffer);
  function word(address, value) {
    if (width === 8) view.setBigUint64(address, BigInt(value), true);
    else view.setUint32(address, value, true);
  }
  function string(address, text, data) {
    const bytes = new TextEncoder().encode(text);
    memory.set(bytes, data);
    word(address, data);
    word(address + width, bytes.length);
  }
  memory[256] = 5; // named + uncommon
  memory[257] = 2; // int
  string(256 + width, 'pkg.Record', 1024);
  string(256 + 3 * width, 'example.com/one', 1088);
  memory[385] = 22 | 32; // pointer + direct interface, no uncommon metadata
  string(384 + width, '*pkg.Record', 1152);
  word(384 + 3 * width, 256);
  const plugin = new LLGoLanguageExtensionPlugin({
    getWasmLinearMemory: async (offset, length) => memory.slice(offset, offset + length).buffer,
  });
  const module = {record: {pointer_size: width}, layout: debuggerSchema.runtime_layouts['2'],
    types: new Map(types.map(type => [type.id, type])), typesByName: new Map(types.map(type => [type.name, type]))};
  assert.equal((await plugin.interfaceValueType(module, 256n, 'pkg.Record', 'stop')).id, 'actual');
  assert.equal((await plugin.interfaceValueType(module, 384n, '*pkg.Record', 'stop')).id, 'pointer');
  string(256 + 3 * width, 'example.com/missing', 1088);
  assert.equal(await plugin.interfaceValueType(module, 256n, 'pkg.Record', 'stop'), null,
    'a short-name collision must not substitute another package');
  memory[256] = 4; // named but unavailable uncommon metadata
  assert.equal(await plugin.interfaceValueType(module, 256n, 'pkg.Record', 'stop'), null);
  word(384 + 3 * width, 384); // corrupt recursive pointer metadata
  assert.equal(await plugin.interfaceValueType(module, 384n, '*pkg.Record', 'stop'), null);
});
}

test('DWARF frame bases retain J64 addresses and Wasm float and signed local values', async () => {
  const index = fixtureIndex();
  index.record.pointer_size = 8;
  index.types.push(
    {id: 'int64', name: 'int64', kind: 'integer', size: 8, signed: true, complete: true},
    {id: 'uint32', name: 'uint32', kind: 'integer', size: 4, signed: false, complete: true},
    {id: 'float32', name: 'float32', kind: 'float', size: 4, complete: true},
    {id: 'float64', name: 'float64', kind: 'float', size: 8, complete: true},
  );
  index.variables = [
    {name: 'stacked', scope: 'LOCAL', type: 'int64', depth: 2,
     locations: [{expression: '9170'}], frame_base: [{expression: 'ed00019f'}]},
    {name: 'single', scope: 'LOCAL', type: 'float32', depth: 2, locations: [{expression: 'ed00029f'}]},
    {name: 'double', scope: 'LOCAL', type: 'float64', depth: 2, locations: [{expression: 'ed00039f'}]},
    {name: 'negative', scope: 'LOCAL', type: 'int32', depth: 2, locations: [{expression: 'ed00049f'}]},
    {name: 'unsigned', scope: 'LOCAL', type: 'uint32', depth: 2, locations: [{expression: 'ed00059f'}]},
    {name: 'noFrameBase', scope: 'LOCAL', type: 'int64', depth: 2, locations: [{expression: '9170'}]},
  ];
  const frameBase = 0x100000050n;
  const memory = new ArrayBuffer(8);
  new DataView(memory).setBigInt64(0, -7n, true);
  const reads = [];
  const services = {
    getWasmLocal: async (id, stopId) => {
      assert.equal(stopId, 'j64-stop');
      return {
        1: {type: 'i64', value: frameBase},
        2: {type: 'f32', value: -1.25},
        3: {type: 'f64', value: 1.5e125},
        4: {type: 'i32', value: -7},
        5: {type: 'i32', value: -1},
      }[id];
    },
    getWasmLinearMemory: async (offset, length, stopId) => {
      reads.push([offset, length, stopId]);
      assert.equal(offset, Number(frameBase - 16n));
      assert.equal(length, 8);
      return memory;
    },
  };
  const fetcher = async url => {
    if (String(url).endsWith('/__llgo/debug-index.json')) return response(index);
    if (String(url).endsWith('/__llgo/debug-schema.json')) return response(schema);
    if (String(url).endsWith('/__llgo/plugin-ready')) return new Response(null, {status: 204});
    throw new Error(`unexpected URL ${url}`);
  };
  const plugin = new LLGoLanguageExtensionPlugin(services, fetcher);
  await plugin.addRawModule('j64', undefined, {
    url: 'http://127.0.0.1:1234/program.wasm', code: moduleBytes({pointerSize: 8}).buffer,
  });
  const context = {rawModuleId: 'j64', codeOffset: 12};
  for (const [name, expected] of [['stacked', -7], ['single', -1.25], ['double', 1.5e125],
                                ['negative', -7], ['unsigned', 4294967295]]) {
    const value = await plugin.evaluate(name, context, 'j64-stop');
    assert.equal(value.value, expected, name);
  }
  assert.equal(await plugin.evaluate('noFrameBase', context, 'j64-stop'), null);
  assert.deepEqual(reads, [[Number(frameBase - 16n), 8, 'j64-stop']]);
});


test('direct Wasm pointers use unsigned target width without changing signed integers', async () => {
  for (const pointerSize of [4, 8]) {
    const index = fixtureIndex();
    index.record.pointer_size = pointerSize;
    index.types.push({id: 'pointer', name: '*int32', kind: 'pointer', size: pointerSize,
                      elem: 'int32', complete: true});
    index.variables = [
      {name: 'pointer', scope: 'LOCAL', type: 'pointer', depth: 2, locations: [{expression: 'ed00009f'}]},
      {name: 'integer', scope: 'LOCAL', type: 'int32', depth: 2, locations: [{expression: 'ed00019f'}]},
    ];
    const reads = [];
    const services = {
      getWasmLocal: async id => id === 0 && pointerSize === 8 ? {type: 'i64', value: 0x100000050n} :
                                                              {type: 'i32', value: -2147483648},
      getWasmLinearMemory: async (offset, length) => {
        reads.push([offset, length]);
        const memory = new ArrayBuffer(length);
        new DataView(memory).setInt32(0, 42, true);
        return memory;
      },
    };
    const fetcher = async url => {
      if (String(url).endsWith('/__llgo/debug-index.json')) return response(index);
      if (String(url).endsWith('/__llgo/debug-schema.json')) return response(schema);
      if (String(url).endsWith('/__llgo/plugin-ready')) return new Response(null, {status: 204});
      throw new Error(`unexpected URL ${url}`);
    };
    const plugin = new LLGoLanguageExtensionPlugin(services, fetcher);
    await plugin.addRawModule('pointers', undefined, {
      url: 'http://127.0.0.1:1234/program.wasm', code: moduleBytes({pointerSize}).buffer,
    });
    const context = {rawModuleId: 'pointers', codeOffset: 12};
    const pointer = await plugin.evaluate('pointer', context, 'stop');
    const expectedAddress = pointerSize === 4 ? 0x80000000 : 0x100000050;
    assert.equal(pointer.description, `0x${expectedAddress.toString(16)}`);
    assert.equal(pointer.linearMemoryAddress, expectedAddress);
    assert.equal((await plugin.getProperties(pointer.objectId))[0].value.value, 42);
    assert.deepEqual(reads, [[expectedAddress, 4]]);
    assert.equal((await plugin.evaluate('integer', context, 'stop')).value, -2147483648);
  }
});

test('unsupported DWARF constant encodings remain unavailable instead of becoming zero', async () => {
  const index = fixtureIndex();
  index.variables = [
    {name: 'zero', scope: 'LOCAL', type: 'int32', depth: 2, constant: {kind: 'signed', value: '0'}},
    {name: 'raw', scope: 'LOCAL', type: 'int32', depth: 2, constant: {kind: 'bytes', value: '01000000'}},
    {name: 'text', scope: 'LOCAL', type: 'int32', depth: 2, constant: {kind: 'string', value: 'hello'}},
  ];
  const fetcher = async url => {
    if (String(url).endsWith('/__llgo/debug-index.json')) return response(index);
    if (String(url).endsWith('/__llgo/debug-schema.json')) return response(schema);
    if (String(url).endsWith('/__llgo/plugin-ready')) return new Response(null, {status: 204});
    throw new Error(`unexpected URL ${url}`);
  };
  const plugin = new LLGoLanguageExtensionPlugin({}, fetcher);
  await plugin.addRawModule('constants', undefined, {
    url: 'http://127.0.0.1:1234/program.wasm', code: moduleBytes().buffer,
  });
  const context = {rawModuleId: 'constants', codeOffset: 12};
  assert.equal((await plugin.evaluate('zero', context, 'stop')).value, 0);
  assert.equal(await plugin.evaluate('raw', context, 'stop'), null);
  assert.equal(await plugin.evaluate('text', context, 'stop'), null);
});
