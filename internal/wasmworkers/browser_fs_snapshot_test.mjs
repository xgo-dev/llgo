import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';

const source = readFileSync(new URL('./browser_fs_snapshot.js', import.meta.url), 'utf8');
const reads = [];
const boxed = [];
const realm = {};
const originalArray = function OriginalArray() {};
const flags = new Proxy({
  O_WRONLY: 1, O_RDWR: 2, O_CREAT: 64, O_TRUNC: 512, O_APPEND: 1024, O_EXCL: 128,
}, {
  get(target, property) {
    reads.push(property);
    return target[property];
  },
});
const originalFS = {
  get constants() {
    reads.push('constants');
    return flags;
  },
};
const originals = {
  Object, String, Array, Function, NaN, Uint8Array: originalArray, Uint8ClampedArray,
  process: {}, path: {}, fs: originalFS,
};
for (const property of Object.keys(originals)) {
  Object.defineProperty(realm, property, {
    configurable: true,
    get() {
      reads.push(property);
      return originals[property];
    },
  });
}
const environment = {
  globalThis: realm,
  Emval: {toHandle(value) { return BigInt(boxed.push(value)); }},
  addToLibrary(library) {
    for (const [name, value] of Object.entries(library)) {
      if (!name.includes('__')) environment[name.replace(/^\$/, '')] = value;
    }
  },
};
runInNewContext(source, environment);

environment.llgo_fs_snapshot_capture(0);
environment.llgo_fs_snapshot_capture(1);
environment.llgo_fs_snapshot_capture(2);
assert.equal(environment.llgo_fs_snapshot_constants(2), 0);
environment.llgo_fs_snapshot_capture(3);
assert.deepEqual([0, 1, 2, 3, 4, 5, 6].map(environment.llgo_fs_snapshot_flag),
  [1, 2, 64, 512, 1024, 128, -1]);
assert.deepEqual(reads, [
  'Object', 'String', 'Array', 'Function', 'NaN', 'Uint8Array', 'Uint8ClampedArray',
  'process', 'path', 'fs', 'constants', 'Uint8Array',
  'O_WRONLY', 'O_RDWR', 'O_CREAT', 'O_TRUNC', 'O_APPEND', 'O_EXCL', 'O_DIRECTORY',
]);
assert.deepEqual(boxed, []);

originals.fs = {};
originals.Uint8Array = function NewArray() {};
assert.equal(environment.llgo_fs_snapshot_ref(2), 1);
assert.equal(environment.llgo_fs_snapshot_ref(3), 2);
assert.equal(environment.llgo_fs_snapshot_global(7), 3);
assert.deepEqual(boxed, [originalFS, originalArray, originalArray]);

flags.O_WRONLY = undefined;
assert.equal(environment.llgo_fs_snapshot_flag(0), NaN);
assert.equal(environment.llgo_fs_snapshot_error(), 8);
flags.O_WRONLY = NaN;
assert.equal(environment.llgo_fs_snapshot_flag(0), NaN);
assert.equal(environment.llgo_fs_snapshot_error(), 0);
flags.O_WRONLY = 1n;
assert.equal(environment.llgo_fs_snapshot_flag(0), NaN);
assert.equal(environment.llgo_fs_snapshot_error(), 32);

originals.fs = null;
environment.llgo_fs_snapshot_capture(2);
assert.equal(environment.llgo_fs_snapshot_constants(2), 17);
