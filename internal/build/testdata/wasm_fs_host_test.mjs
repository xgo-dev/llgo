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
  assert(ctx.TextDecoder === native, "shim must not replace global TextDecoder");
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
  assertEq(ctx.path.resolve("foo", "../bar"), "/tmp/bar", "path.resolve uses cwd and normalizes ..");
  assertEq(ctx.path.resolve("/a", "b", "/c", "d"), "/c/d", "path.resolve resets on absolute segment");
  assertEq(ctx.path.resolve(1, "x"), "/tmp/1/x", "path.resolve stringifies non-string segments");
  assertEq(ctx.process.cwd(), "/tmp", "process.cwd uses attached FS");
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
