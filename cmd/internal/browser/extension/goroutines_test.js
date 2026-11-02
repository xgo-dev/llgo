const assert = require('node:assert/strict');
const test = require('node:test');
const schema = require('../../../../internal/debugabi/schema_v1.json');
require('./goroutines.js');
require('./plugin.js');

// Memory32 uses 64-bit Go ints and padded Go pointer slots. Read sizes must
// come from DWARF fields, rather than assuming all fields have pointer width.
function fixture(pointerSize) {
  const types = new Map();
  function struct(id, size, fields) {
    types.set(id, {id, kind: 'struct', size,
      fields: fields.map(([name, type, offset]) => ({name, type, offset}))});
  }
  function pointer(id, elem) {
    types.set(id, {id, kind: 'pointer', size: pointerSize, elem});
  }
  for (const size of [4, 8]) types.set(`u${size * 8}`, {kind: 'integer', size});
  pointer('nodePtr', 'node'); pointer('gPtr', 'g'); pointer('mPtr', 'm');
  pointer('pPtr', 'p'); pointer('storePtr', 'store'); pointer('framePtr', 'frame');
  pointer('bytePtr', 'u32');
  struct('registry', 16, [['version', 'u32', 0], ['epoch', 'u32', 4], ['head', 'nodePtr', 8]]);
  struct('node', 48, [['next', 'nodePtr', 0], ['gp', 'gPtr', 16], ['callers', 'storePtr', 24], ['sequence', 'u32', 32], ['processorID', 'u32', 40]]);
  struct('g', 32, [['goid', 'u64', 0], ['parentGoid', 'u64', 8], ['atomicstatus', 'u32', 16], ['m', 'mPtr', 24]]);
  struct('m', 8, [['p', 'pPtr', 0]]);
  struct('p', 8, [['id', 'u64', 0]]);
  struct('store', 24, [['stack', 'slice', 0]]);
  struct('slice', 24, [['data', 'framePtr', 0], ['len', 'u64', 8], ['cap', 'u64', 16]]);
  struct('string', 16, [['data', 'bytePtr', 0], ['len', 'u64', 8]]);
  struct('frame', 40, [['Function', 'string', 0], ['File', 'string', 16], ['Line', 'u64', 32]]);
  const memory = new Uint8Array(16384), view = new DataView(memory.buffer);
  const u32 = (at, value) => view.setUint32(at, value, true);
  const u64 = (at, value) => view.setBigUint64(at, BigInt(value), true);
  const ptr = pointerSize === 4 ? u32 : u64;
  u32(64, 1); u32(68, 2); ptr(72, 128);
  ptr(144, 256); ptr(152, 480); u32(160, 4); u32(168, 2);
  u64(256, 42); u64(264, 1); u32(272, 4); ptr(280, 400);
  ptr(400, 416); u64(416, 2);
  ptr(480, 512); u64(488, 2); u64(496, 2);
  let textAddress = 700;
  function text(at, value) {
    const bytes = new TextEncoder().encode(value);
    ptr(at, textAddress); u64(at + 8, bytes.length);
    memory.set(bytes, textAddress); textAddress += bytes.length;
  }
  text(512, 'main.main'); text(528, '/src/main.go'); u64(544, 10);
  text(552, 'main.wait'); text(568, '/src/main.go'); u64(584, 20);
  const module = {
    rawModuleId: 'module', record: {pointer_size: pointerSize}, types,
    layout: schema.runtime_layouts['2'],
    index: {variables: [{name: 'wasmDebuggerRegistry', scope: 'GLOBAL', type: 'registry',
      locations: [{expression: pointerSize === 4 ? '0340000000' : '034000000000000000'}]}]},
  };
  const reads = [];
  const services = {getWasmLinearMemory: async (at, size) => {
    reads.push({at, size});
    assert.ok(at >= 0 && at + size <= memory.length, `out-of-bounds read ${at}+${size}`);
    return memory.slice(at, at + size).buffer;
  }};
  const plugin = new globalThis.LLGoLanguageExtension.LLGoLanguageExtensionPlugin(services);
  plugin.modules.set('module', module);
  return {plugin, module, memory, u32, u64, ptr, reads, context: {rawModuleId: 'module', codeOffset: 0}};
}

for (const width of [4, 8]) {
  test(`Memory${width * 8} empty caller strings skip memory transport`, async () => {
    const f = fixture(width);
    f.ptr(552, 0); f.u64(560, 0);
    f.u64(576, 0);
    const read = f.plugin.languageServices.getWasmLinearMemory;
    f.plugin.languageServices.getWasmLinearMemory = async (at, size) => {
      assert.ok(size > 0, 'empty strings must not read linear memory');
      return read(at, size);
    };
    const records = await globalThis.LLGoWasmGoroutines.read(f.plugin, f.module, f.context, 'stop');
    assert.deepEqual(records[0].frames[0], {function: '', file: '', line: '20'});
  });

  test(`Memory${width * 8} logical goroutine snapshot and object lifetime`, async () => {
    const {plugin, module, context} = fixture(width);
    const records = await globalThis.LLGoWasmGoroutines.read(plugin, module, context, 'stop');
    assert.deepEqual(records, [{id: '42', parent: '1', state: 'waiting', processor: '2', frames: [
      {function: 'main.wait', file: '/src/main.go', line: '20'},
      {function: 'main.main', file: '/src/main.go', line: '10'},
    ]}]);
    const root = await plugin.evaluate('$goroutines', context, 'stop');
    assert.equal(plugin.objects.size, 1, 'children are allocated only when expanded');
    const children = await plugin.getProperties(root.objectId);
    assert.match(children[0].value.description, /goroutine 42 \[waiting\] P2/);
    const properties = await plugin.getProperties(children[0].value.objectId);
    const frames = properties.find(p => p.name === 'frames').value;
    assert.deepEqual((await plugin.getProperties(frames.objectId)).map(p => p.value.value), [
      'main.wait (/src/main.go:20)', 'main.main (/src/main.go:10)',
    ]);
    const count = plugin.objects.size;
    await plugin.getProperties(root.objectId);
    assert.equal(plugin.objects.size, count, 're-expansion reuses child objects');
    await plugin.releaseObject(root.objectId);
    assert.equal(plugin.objects.size, 0, 'releasing a snapshot releases its expanded children');
  });
}

test('goroutine decoder rejects corrupt, oversized and changing snapshots', async t => {
  const cases = [
    ['version', f => f.u32(64, 2), /unsupported.*registry/],
    ['registry update', f => f.u32(68, 3), /registry is changing/],
    ['stack update', f => f.u32(160, 5), /stack is changing/],
    ['cycle', f => f.ptr(128, 128), /invalid.*registry/],
    ['nil G', f => f.ptr(144, 0), /nil G/],
    ['oversized stack', f => f.u64(488, 4097), /oversized logical stack/],
    ['64-bit string length on Memory32', f => f.u64(560, 0x100000001n), /invalid caller frame/],
    ['registry changed during read', f => {
      const read = f.plugin.readNamedUnsigned.bind(f.plugin);
      let epochs = 0;
      f.plugin.readNamedUnsigned = (...args) => {
        if (args[3] === 'epoch' && ++epochs === 1) f.u32(68, 4);
        return read(...args);
      };
    }, /registry changed/],
    ['stack changed during read', f => {
      const read = f.plugin.readNamedUnsigned.bind(f.plugin);
      let sequences = 0;
      f.plugin.readNamedUnsigned = (...args) => {
        if (args[3] === 'sequence' && ++sequences === 1) f.u32(160, 6);
        return read(...args);
      };
    }, /stack changed/],
  ];
  for (const [name, mutate, error] of cases) await t.test(name, async () => {
    const f = fixture(4);
    mutate(f);
    await assert.rejects(f.plugin.evaluate('$goroutines', f.context, 'stop'), error);
    assert.equal(f.plugin.objects.size, 0, 'failed reads must not retain partial snapshots');
  });
});

test('empty registry and ordinary Wasm builds have explicit results', async () => {
  const f = fixture(4);
  f.ptr(72, 0);
  const empty = await f.plugin.evaluate('$goroutines', f.context, 'stop');
  assert.equal(empty.hasChildren, false);
  f.module.index.variables = [];
  assert.match((await f.plugin.evaluate('$goroutines', f.context, 'stop')).value, /build with llgo.wasm.debugger/);
});


test('deep stacks use batched headers and cache repeated source strings', async () => {
  const f = fixture(4);
  const header = f.memory.slice(552, 592);
  const count = 128, data = 1024;
  for (let index = 0; index < count; ++index) f.memory.set(header, data + index * 40);
  f.ptr(480, data); f.u64(488, count); f.u64(496, count);
  const records = await globalThis.LLGoWasmGoroutines.read(f.plugin, f.module, f.context, 'stop');
  assert.equal(records[0].frames.length, count);
  assert.deepEqual(records[0].frames[127], {function: 'main.wait', file: '/src/main.go', line: '20'});
  assert.equal(f.reads.filter(r => r.at === data && r.size === count * 40).length, 1);
  assert.ok(f.reads.length <= 10, `deep stack used ${f.reads.length} transport calls`);
});

test('incomplete batched reads do not create partial snapshots', async () => {
  const f = fixture(4);
  const read = f.plugin.languageServices.getWasmLinearMemory;
  f.plugin.languageServices.getWasmLinearMemory = async (at, size) =>
    (await read(at, size)).slice(0, size - 1);
  await assert.rejects(f.plugin.evaluate('$goroutines', f.context, 'stop'), /incomplete debugger memory read/);
  assert.equal(f.plugin.objects.size, 0);
});


test('P0 is a valid processor; minus one means the G has not run yet', async () => {
  for (const width of [4, 8]) {
    const f = fixture(width);
    f.u32(168, 0);
    let records = await globalThis.LLGoWasmGoroutines.read(f.plugin, f.module, f.context, 'stop');
    assert.equal(records[0].processor, '0');
    f.u32(168, 0xffffffff);
    records = await globalThis.LLGoWasmGoroutines.read(f.plugin, f.module, f.context, 'stop');
    assert.equal(records[0].processor, null);
  }
});

function parkedFixture(width) {
  const f = fixture(width), {module, memory, u64} = f;
  const frame = module.types.get('frame');
  module.types.set('frameDebug', {id: 'frameDebug', kind: 'struct', size: 8,
    fields: [{name: 'base', type: 'u64', offset: 0}]});
  frame.fields.push({name: 'wasmCallerFrame', type: 'frameDebug', offset: 40});
  frame.size = 48;
  module.types.get('store').fields.push({name: 'panicDepth', type: 'u64', offset: 24});
  module.types.get('store').size = 32;
  memory.set(memory.slice(512, 552), 1024);
  memory.set(memory.slice(552, 592), 1072);
  f.ptr(480, 1024);
  u64(1064, 8192); u64(1112, 9216);
  module.types.set('array', {id: 'array', name: '[4]int', kind: 'array', elem: 'u64', count: 4, size: 32});
  for (let i = 0; i < 4; i++) u64(9200 + i * 8, 40 + i);
  module.index.functions = [{name: 'main.wait', ranges: [{start: 100, end: 200}]}];
  module.index.variables.push({name: 'local', scope: 'LOCAL', type: 'array',
    ranges: [{start: 100, end: 200}], locations: [{expression: '9170'}]});
  for (const name of ['getWasmLocal', 'getWasmOp', 'getWasmGlobal']) {
    f.plugin.languageServices[name] = async () => assert.fail('parked locals must not use current Wasm registers');
  }
  return f;
}

for (const width of [4, 8]) {
  test(`Memory${width * 8} parked frame locals, stale stops and recursive release`, async () => {
    const f = parkedFixture(width), {plugin, module, context} = f;
    let stopId = 7;
    module.index.worker_control = true;
    module.rawModule = {url: 'http://localhost/module.wasm'};
    plugin.fetcher = async () => ({ok: true, json: async () => ({enabled: true, ready: true, paused: true, stopId})});
    const root = await plugin.evaluate('$goroutines', context, 'stop');
    const goroutines = await plugin.getProperties(root.objectId);
    const properties = await plugin.getProperties(goroutines[0].value.objectId);
    const frames = await plugin.getProperties(properties.find(p => p.name === 'frames').value.objectId);
    const locals = await plugin.getProperties(frames[0].value.objectId);
    assert.deepEqual((await plugin.getProperties(locals[0].value.objectId)).map(p => p.value.value), [40, 41, 42, 43]);
    module.types.set('nested', {id: 'nested', kind: 'array', count: 1, elem: 'array', size: 32});
    const aggregate = await plugin.remoteObject(module, module.types.get('nested'), {kind: 'address', value: 9200n}, 'stop');
    const children = await plugin.getProperties(aggregate.objectId);
    assert.ok(children[0].value.objectId);
    await plugin.releaseObject(aggregate.objectId);
    assert.equal(plugin.objects.has(children[0].value.objectId), false, 'aggregate children follow their owner');
    stopId++;
    await assert.rejects(plugin.getProperties(frames[0].value.objectId), /workers resumed or changed/);
    await plugin.releaseObject(root.objectId);
    assert.equal(plugin.objects.size, 0);
  });

  test(`Memory${width * 8} unwound, running and unpublished frame bases stay unavailable`, async () => {
    for (const mutate of [f => f.u64(504, 1), f => f.u32(272, 2), f => { f.u64(1064, 0); f.u64(1112, 0); }]) {
      const f = parkedFixture(width);
      mutate(f);
      const records = await LLGoWasmGoroutines.read(f.plugin, f.module, f.context, 'stop');
      assert.ok(records.every(record => record.frames.every(frame => !frame.base)));
    }
  });
}

test('parked locals reject register, changing, ambiguous and malformed locations', async t => {
  for (const [name, mutate] of [
    ['Wasm register', v => v.locations[0].expression = 'ed00019f'],
    ['nested scope', v => v.ranges[0].start = 110],
    ['scope gap', v => v.ranges = [{start: 100, end: 110}, {start: 120, end: 200}]],
    ['location range', v => v.locations[0] = {start: 100, end: 150, expression: '9170'}],
    ['multiple locations', v => v.locations.push({expression: '9170'})],
    ['truncated SLEB', v => v.locations[0].expression = '9180'],
    ['extra opcode', v => v.locations[0].expression = '91709f'],
    ['oversized SLEB', v => v.locations[0].expression = '91' + '80'.repeat(12) + '00'],
    ['overflowing SLEB', v => v.locations[0].expression = '91' + 'ff'.repeat(9) + '01'],
  ]) await t.test(name, async () => {
    const f = parkedFixture(4);
    mutate(f.module.index.variables[1]);
    const locals = await LLGoWasmGoroutines.locals(f.plugin, f.module, {function: 'main.wait', base: '9216'}, 'stop');
    assert.match(locals[0].value.value, /unavailable/);
    assert.equal(f.reads.length, 0);
  });
  const f = parkedFixture(4);
  f.module.index.variables.push({...f.module.index.variables[1]});
  assert.match((await LLGoWasmGoroutines.locals(f.plugin, f.module, {function: 'main.wait', base: '9216'}, 'stop'))[0].value.value, /unavailable/);
  f.module.index.functions.push({...f.module.index.functions[0]});
  assert.match((await LLGoWasmGoroutines.locals(f.plugin, f.module, {function: 'main.wait', base: '9216'}, 'stop'))[0].value.value, /unambiguous/);
  assert.equal(f.reads.length, 0);
});

test('failed parked reads release objects allocated by earlier locals', async () => {
  const f = parkedFixture(4);
  f.module.index.variables.push({...f.module.index.variables[1], name: 'bad', type: 'u64'});
  f.plugin.languageServices.getWasmLinearMemory = async () => { throw new Error('read failed'); };
  await assert.rejects(LLGoWasmGoroutines.locals(f.plugin, f.module, {function: 'main.wait', base: '9216'}, 'stop'), /read failed/);
  assert.equal(f.plugin.objects.size, 0);
});
