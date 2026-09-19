import vm from "node:vm";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const src = fs.readFileSync(path.join(repoRoot, "targets", "wasm_fs.js"), "utf8");

function assert(cond, msg) {
  if (!cond) throw new Error(msg || "assertion failed");
}

function assertEq(got, want, msg) {
  if (got !== want) {
    throw new Error(`${msg || "values differ"}: got ${JSON.stringify(got)}, want ${JSON.stringify(want)}`);
  }
}

function loadShim(extra = {}) {
  const context = {
    console,
    TextDecoder,
    Uint8Array,
    Uint16Array,
    Int8Array,
    ArrayBuffer,
    DataView,
    Error,
    Date,
    Math,
    Number,
    String,
    Object,
    Function,
    JSON,
    ...extra,
  };
  context.globalThis = context;
  vm.createContext(context);
  vm.runInContext(src, context);
  return context;
}

function encoder() {
  return new TextEncoder();
}

{
  const native = globalThis.TextDecoder;
  const ctx = loadShim();
  assert(ctx.TextDecoder !== native, "shim wraps TextDecoder for resizable wasm memory");
  assert(ctx.TextDecoder.prototype.__llgoResizableSafe, "wrapped decoder is marked safe");
  const dec = new ctx.TextDecoder("utf-8");
  const bytes = Uint8Array.of(65, 66);
  assertEq(dec.decode(new DataView(bytes.buffer)), "AB", "DataView decode");
  assertEq(dec.decode(new Uint16Array(Uint8Array.of(65, 66).buffer)), "AB", "Uint16Array decode");
}

{
  const existing = { tag: "node-fs" };
  const ctx = loadShim({ fs: existing });
  assert(ctx.fs === existing, "existing fs must be left in place");
  assert(typeof ctx.llgoAttachWasmFS === "function", "attach is exported even when fs already exists");
}

{
  const printed = [];
  const printedErr = [];
  const Module = {
    print: (line) => printed.push(line),
    printErr: (line) => printedErr.push(line),
  };
  const ctx = loadShim();
  ctx.llgoAttachWasmFS(Module);
  const enc = encoder();
  ctx.fs.writeSync(1, enc.encode("stdout-prefix"));
  const stderr = enc.encode("stderr-line\n");
  ctx.fs.write(2, stderr, 0, stderr.length, null, (err, n) => {
    assert(!err, `stderr write: ${err && err.message}`);
    assertEq(n, stderr.length, "stderr bytes written");
  });
  assertEq(printed.length, 0, "stdout without newline stays buffered");
  assertEq(printedErr.join("\n"), "stderr-line", "stderr must not mix with stdout");
  ctx.fs.writeSync(1, enc.encode("!\n"));
  assertEq(printed.join("\n"), "stdout-prefix!", "stdout flushes independently");
}

{
  const printed = [];
  const ctx = loadShim();
  ctx.llgoAttachWasmFS({ print: (line) => printed.push(line) });
  ctx.fs.writeSync(1, encoder().encode("no-nl"));
  assertEq(printed.length, 0, "partial stdout stays buffered");
  ctx.fs.fsync(1, (err) => { assert(!err, `fsync: ${err && err.message}`); });
  assertEq(printed.join("\n"), "no-nl", "fsync flushes stdout without newline");
  ctx.fs.writeSync(1, encoder().encode("again"));
  ctx.fs.close(1, (err) => { assert(!err, `close: ${err && err.message}`); });
  assertEq(printed.join("\n"), "no-nl\nagain", "close flushes remaining stdout");
}

{
  const printed = [];
  const Module = { print: (line) => printed.push(line) };
  const ctx = loadShim();
  ctx.llgoAttachWasmFS(Module);
  const hello = encoder().encode("你好\n");
  ctx.fs.writeSync(1, hello.subarray(0, 1));
  ctx.fs.writeSync(1, hello.subarray(1));
  assertEq(printed.join("\n"), "你好", "split UTF-8 writes must decode as a stream");
}

{
  class ChromeTextDecoder extends globalThis.TextDecoder {
    decode(input, options) {
      const buffer = ArrayBuffer.isView(input) ? input.buffer : input;
      if (buffer && (buffer.resizable || buffer.growable)) {
        throw new TypeError("The provided ArrayBuffer value must not be resizable");
      }
      return super.decode(input, options);
    }
  }
  const printed = [];
  const ctx = loadShim({ TextDecoder: ChromeTextDecoder });
  ctx.llgoAttachWasmFS({ print: (line) => printed.push(line) });
  let view;
  try {
    const ab = new ArrayBuffer(16, { maxByteLength: 64 });
    view = new Uint8Array(ab, 0, 6);
  } catch (_) {
    view = null;
  }
  if (view && view.buffer.resizable) {
    view.set(encoder().encode("hello\n"));
    ctx.fs.writeSync(1, view);
    assertEq(printed.join("\n"), "hello", "stdout copies off resizable wasm memory");
  }
}

{
  const printed = [];
  const ctx = loadShim();
  ctx.llgoAttachWasmFS({ print: (line) => printed.push(line) });
  let opened = false;
  ctx.fs.open("/tmp/review.txt", 64, 0o666, (err) => {
    opened = true;
    assert(err && err.code === "ENOSYS", `open without FS: ${err && err.code}`);
  });
  assert(opened, "open callback ran");
  const mem = {
    files: Object.create(null),
    cwd() { return "/tmp"; },
    mkdir(p) { this.files[p] = { dir: true }; },
    open(p) {
      this.files[p] = this.files[p] || { data: new Uint8Array(0) };
      return { fd: 10 };
    },
    write(stream, buf, offset, length) {
      return length;
    },
    getStreamChecked(fd) { return { fd }; },
  };
  ctx.llgoAttachWasmFS({ FS: mem, print: (line) => printed.push(line) });
  ctx.fs.open("/tmp/review.txt", 64, 0o666, (err, fd) => {
    assert(!err, `open with attached FS: ${err && err.message}`);
    assertEq(fd, 10, "open fd");
  });
  assertEq(ctx.fs.openSync("/tmp/review2.txt", 64, 0o666), 10, "openSync fd");
  const n = ctx.fs.writeSync(10, encoder().encode("hi"), 0, 2, null);
  assertEq(n, 2, "writeSync bytes");
  assert(ctx.fs.readSync === undefined, "readSync must not be synthesized; blocking reads freeze the scheduler");
  assert(ctx.fs.fsyncSync === undefined, "fsyncSync must not be synthesized; blocking fsync freezes the scheduler");
  assert(typeof ctx.fs.writeSync === "function", "writeSync remains available for non-blocking writes");
  assert(typeof ctx.fs.openSync === "function", "openSync remains available");
  assertEq(ctx.path.resolve("foo", "../bar"), "/tmp/bar", "path.resolve uses cwd and normalizes ..");
  assertEq(ctx.path.resolve("/a", "b", "/c", "d"), "/c/d", "path.resolve resets on absolute segment");
  assertEq(ctx.path.resolve(1, "x"), "/tmp/1/x", "path.resolve stringifies non-string segments");
  assertEq(ctx.process.cwd(), "/tmp", "process.cwd uses attached FS");
}

{
  const ctx = loadShim();
  const errnoError = (errno) => {
    const e = new Error("ErrnoError");
    e.code = "ErrnoError";
    e.errno = errno;
    return e;
  };
  ctx.llgoAttachWasmFS({
    FS: {
      write() { throw errnoError(44); },
      getStreamChecked() { return {}; },
      chdir() { throw errnoError(44); },
    },
  });
  let threw = false;
  try {
    ctx.fs.writeSync(10, encoder().encode("x"), 0, 1, null);
  } catch (e) {
    threw = true;
    assertEq(e.code, "ENOENT", "writeSync normalizes ErrnoError errno to Node code");
  }
  assert(threw, "writeSync threw");
  threw = false;
  try {
    ctx.process.chdir("/missing");
  } catch (e) {
    threw = true;
    assertEq(e.code, "ENOENT", "chdir normalizes ErrnoError errno to Node code");
  }
  assert(threw, "chdir threw");
}

{
  const printed = [];
  const ctx = loadShim({ Module: { print: (line) => printed.push(line) } });
  assert(typeof ctx.fs.writeSync === "function", "global Module is attached at load");
  ctx.fs.writeSync(1, encoder().encode("hello\n"));
  assertEq(printed.join("\n"), "hello", "global Module.print receives stdout");
}

{
  const printed = [];
  const ctx = loadShim({ TextDecoder: undefined, Module: { print: (line) => printed.push(line) } });
  ctx.fs.writeSync(1, Uint8Array.of(0x68, 0x69, 0x0a));
  assertEq(printed.join("\n"), "hi", "stdout works without TextDecoder");
}

console.log("ok");
