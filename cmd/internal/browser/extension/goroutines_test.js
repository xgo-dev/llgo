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
