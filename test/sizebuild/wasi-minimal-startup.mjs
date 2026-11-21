import assert from 'node:assert/strict';
import fs from 'node:fs';
import { WASI } from 'node:wasi';

const [artifact, initialBytes, maximumBytes] = process.argv.slice(2);
const module = new WebAssembly.Module(fs.readFileSync(artifact));
assert.ok(!WebAssembly.Module.imports(module).some(({ name }) =>
  name === 'args_get' || name === 'args_sizes_get'));
assert.ok(Number(initialBytes) < 16 * 1024 * 1024);

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

const size = 16 * 1024 * 1024;
const pointer = instance.exports.malloc(size);
assert.ok(pointer > 0);
assert.equal(pointer % 16, 0);
const data = new Uint8Array(memory.buffer, pointer, size);
data[0] = data[size - 1] = 42;
assert.ok(memory.buffer.byteLength > Number(initialBytes));
instance.exports.free(pointer);
