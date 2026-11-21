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
const wasi = new WASI({ version: 'preview1', args: ['stdout', 'host'], returnOnExit: true });
const imports = wasi.getImportObject();
imports.env = { memory };
imports.wasi = { 'thread-spawn': () => -1 };

const fsImports = imports.wasi_snapshot_preview1;
const { fd_write: hostWrite, fd_close: hostClose } = fsImports;
const writes = [
  ['literal %\n', 2, 0],
  ['teral %\n', 0, 27], // EINTR
  ['teral %\n', 8, 0],
  ['', 0, 0],
  ['no progress', 0, 0],
  ['oversized', 91, 0],
  ['unlocked', 8, 0],
  ['closed descriptor', 0, 8], // EBADF
];
fsImports.fd_write = (fd, vectors, count, result) => {
  if (fd !== 1) return hostWrite(fd, vectors, count, result);
  assert.equal(count, 1);
  const [text, written, errno] = writes.shift();
  const fields = new DataView(memory.buffer);
  const address = fields.getUint32(vectors, true);
  const length = fields.getUint32(vectors + 4, true);
  assert.equal(Buffer.from(memory.buffer, address, length).toString(), text);
  fields.setUint32(result, written, true);
  return errno;
};
fsImports.fd_close = (fd) => fd === 1 ? 0 : hostClose(fd);

assert.equal(wasi.start(new WebAssembly.Instance(module, imports)), 0);
assert.equal(writes.length, 0);
