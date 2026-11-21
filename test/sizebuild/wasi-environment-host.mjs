import assert from 'node:assert/strict';
import fs from 'node:fs';
import { WASI } from 'node:wasi';

const [artifact, initialBytes, maximumBytes] = process.argv.slice(2);
const module = new WebAssembly.Module(fs.readFileSync(artifact));
const memory = new WebAssembly.Memory({
  initial: Number(initialBytes) / 65536,
  maximum: Number(maximumBytes) / 65536,
  shared: true,
});
const wasi = new WASI({ version: 'preview1', args: ['environment'], returnOnExit: true });
const imports = wasi.getImportObject();
imports.env = { memory };
imports.wasi = { 'thread-spawn': () => -1 };

// Supply entries the WASI constructor's env object cannot express.
const entries = [
  'LLGO_WASI_TEST_ENV=old=value',
  'LLGO_WASI_TEST_ENV=duplicate',
  'LLGO_WASI_TEST_EMPTY=',
  'LLGO_WASI_TEST_SEQUENTIAL=1',
  'LLGO_WASI_TEST_MALFORMED',
  '=empty key',
].map((entry) => Buffer.from(entry + '\0'));

imports.wasi_snapshot_preview1.environ_sizes_get = (count, size) => {
  const fields = new DataView(memory.buffer);
  fields.setUint32(count, entries.length, true);
  fields.setUint32(size, entries.reduce((sum, entry) => sum + entry.length, 0), true);
  return 0;
};

imports.wasi_snapshot_preview1.environ_get = (pointers, data) => {
  const fields = new DataView(memory.buffer);
  const bytes = new Uint8Array(memory.buffer);
  for (let i = 0; i < entries.length; i++) {
    fields.setUint32(pointers + i * 4, data, true);
    bytes.set(entries[i], data);
    data += entries[i].length;
  }
  return 0;
};

const instance = new WebAssembly.Instance(module, imports);
assert.equal(wasi.start(instance), 0);
