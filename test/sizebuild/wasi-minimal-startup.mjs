import assert from 'node:assert/strict';
import fs from 'node:fs';
import { WASI } from 'node:wasi';

const [artifact, initialBytes, maximumBytes, mode] = process.argv.slice(2);
const module = new WebAssembly.Module(fs.readFileSync(artifact));
if (mode !== 'compact') {
  assert.ok(!WebAssembly.Module.imports(module).some(({ name }) =>
    name === 'args_get' || name === 'args_sizes_get'));
  assert.ok(Number(initialBytes) < 16 * 1024 * 1024);
}

const memory = new WebAssembly.Memory({
  initial: Number(initialBytes) / 65536,
  maximum: Number(maximumBytes) / 65536,
  shared: true,
});
const wasi = new WASI({ version: 'preview1', args: ['println', 'unused'], returnOnExit: true });
const imports = wasi.getImportObject();
imports.env = { memory };
imports.wasi = { 'thread-spawn': () => -1 };
const instance = new WebAssembly.Instance(module, imports);
assert.equal(wasi.start(instance), 0);

const size = mode === 'compact' ? Number(initialBytes) : 16 * 1024 * 1024;
const pointer = instance.exports.malloc(size);
assert.ok(pointer > 0);
assert.equal(pointer % 16, 0);
const data = new Uint8Array(memory.buffer, pointer, size);
data[0] = data[size - 1] = 42;
assert.ok(memory.buffer.byteLength > Number(initialBytes));
instance.exports.free(pointer);

if (mode === 'compact') {
  const { malloc, free } = instance.exports;
  const slots = Array(128).fill(null);
  let random = 240185311;
  for (let i = 0; i < 30000; ++i) {
    random ^= random << 13;
    random ^= random >>> 17;
    random ^= random << 5;
    const position = (random >>> 0) % slots.length;
    const previous = slots[position];
    if (previous !== null) {
      const bytes = new Uint8Array(memory.buffer, previous.pointer, previous.length);
      assert.equal(bytes[0], previous.marker);
      assert.equal(bytes[bytes.length - 1], previous.marker);
      free(previous.pointer);
    }
    const length = 1 + ((random >>> 8) % 131072);
    const marker = (i % 255) + 1;
    const pointer = malloc(length);
    assert.ok(pointer > 0);
    assert.equal(pointer % 16, 0);
    const bytes = new Uint8Array(memory.buffer, pointer, length);
    bytes[0] = bytes[length - 1] = marker;
    slots[position] = { pointer, length, marker };
  }
  for (const slot of slots) {
    if (slot !== null) free(slot.pointer);
  }
  assert.equal(malloc(-1), 0);
}
