// wasm_fs.js: browser host shim for Go syscall/fs_js.go on LLGo/Emscripten.
// Provides globalThis.fs, process, and path. Load before the generated main.js.
//
// Bind the shim to the Emscripten module instance before Go runs:
//
//   <script src="./wasm_fs.js"></script>
//   <script type="module">
//     import initModule from "./main.js";
//     const Module = {};
//     globalThis.llgoAttachWasmFS(Module);
//     await initModule(Module);
//   </script>
//
// Auto-generated emcc HTML already has a global Module; attach is optional there.
(function (global) {
  const NativeTextDecoder = global.TextDecoder;

  let attached = null;

  function currentModule() {
    return attached || global.Module || null;
  }

  function llgoAttachWasmFS(mod) {
    if (mod && typeof mod === "object") attached = mod;
    return currentModule();
  }

  global.llgoAttachWasmFS = llgoAttachWasmFS;
  if (global.Module && typeof global.Module === "object") {
    llgoAttachWasmFS(global.Module);
  }

  if (global.fs) return;

  let umaskValue = 0o022;
  const pendingLimit = 65536;

  // Emscripten WASI errno numbers from struct_info_generated.json.
  const errnoToCode = {
    2: "EACCES", 6: "EAGAIN", 8: "EBADF", 10: "EBUSY", 13: "ECONNABORTED",
    16: "EDEADLK", 20: "EEXIST", 27: "EINTR", 28: "EINVAL", 29: "EIO",
    31: "EISDIR", 32: "ELOOP", 33: "EMFILE", 34: "EMLINK", 37: "ENAMETOOLONG",
    43: "ENODEV", 44: "ENOENT", 48: "ENOMEM", 51: "ENOSPC", 52: "ENOSYS",
    54: "ENOTDIR", 55: "ENOTEMPTY", 59: "ENOTTY", 63: "EPERM", 64: "EPIPE",
    69: "EROFS", 70: "ESPIPE",
  };

  function enosys() {
    const err = new Error("not implemented");
    err.code = "ENOSYS";
    return err;
  }

  function nodeError(code, message) {
    const err = new Error(message || code);
    err.code = code;
    return err;
  }

  function toNodeError(e) {
    if (!e) return nodeError("EIO");
    if (typeof e.code === "string" && e.code !== "ErrnoError") return e;
    if (typeof e.errno === "number") {
      const code = errnoToCode[e.errno] || "EIO";
      const err = nodeError(code, e.message || code);
      err.errno = e.errno;
      return err;
    }
    return nodeError("EIO", e.message || String(e));
  }

  function currentFS() {
    const mod = currentModule();
    return mod && mod.FS;
  }

  function emFS() {
    const fs = currentFS();
    if (!fs) throw nodeError("ENOSYS", "Emscripten Module.FS is not initialized");
    return fs;
  }

  function timeMs(t) {
    if (t == null) return Date.now();
    if (t instanceof Date) return t.getTime();
    if (typeof t === "number") return t < 1e12 ? t * 1000 : t;
    return Date.now();
  }

  function statObject(st) {
    const atimeMs = timeMs(st.atime);
    const mtimeMs = timeMs(st.mtime);
    const ctimeMs = timeMs(st.ctime);
    const size = st.size || 0;
    return {
      dev: st.dev || 0,
      ino: st.ino || 0,
      mode: st.mode || 0,
      nlink: st.nlink || 1,
      uid: st.uid || 0,
      gid: st.gid || 0,
      rdev: st.rdev || 0,
      size,
      blksize: st.blksize || 4096,
      blocks: st.blocks != null ? st.blocks : Math.ceil(size / 512),
      atimeMs,
      mtimeMs,
      ctimeMs,
      isDirectory: () => (st.mode & 16384) !== 0,
    };
  }

  function call(callback, fn) {
    let value;
    let error;
    try {
      value = fn();
    } catch (e) {
      error = toNodeError(e);
    }
    callback(error, value);
  }

  // Copy off wasm memory / resizable ArrayBuffer views without changing the
  // element type (Uint8Array.from would coerce DataView/Uint16Array values).
  function toUint8(buf) {
    if (buf instanceof Uint8Array) return buf.slice();
    if (ArrayBuffer.isView(buf)) {
      return new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength).slice();
    }
    if (buf instanceof ArrayBuffer) return new Uint8Array(buf.slice(0));
    return new Uint8Array(buf);
  }

  function sliceBuf(buf, offset, length) {
    if (typeof buf.subarray === "function") {
      if (offset === 0 && length === buf.length) return buf;
      return buf.subarray(offset, offset + length);
    }
    return toUint8(buf).subarray(offset, offset + length);
  }

  function stdWriter() {
    return {
      decoder: NativeTextDecoder ? new NativeTextDecoder("utf-8") : null,
      pending: "",
    };
  }

  const stdio = { 1: stdWriter(), 2: stdWriter() };

  function emit(fd, line) {
    const mod = currentModule();
    if (fd === 2) {
      const printErr = mod && (mod.printErr || mod.print);
      if (typeof printErr === "function") printErr(line);
      else console.error(line);
      return;
    }
    const print = mod && mod.print;
    if (typeof print === "function") print(line);
    else console.log(line);
  }

  function decodeBytesFallback(bytes) {
    let s = "";
    for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i] & 0xff);
    return s;
  }

  function flushStdio(fd) {
    const st = stdio[fd];
    if (!st) return;
    if (st.decoder) {
      try {
        st.pending += st.decoder.decode();
      } catch (_) {}
    }
    if (st.pending) {
      emit(fd, st.pending);
      st.pending = "";
    }
  }

  function writeStdio(fd, buf) {
    const bytes = toUint8(buf);
    const st = stdio[fd] || (stdio[fd] = stdWriter());
    const text = st.decoder
      ? st.decoder.decode(bytes, { stream: true })
      : decodeBytesFallback(bytes);
    // pending never contains a newline (complete lines are flushed below), so
    // only the newly decoded text can introduce one.
    st.pending += text;
    if (text.indexOf("\n") !== -1) {
      const nl = st.pending.lastIndexOf("\n");
      const complete = st.pending.substring(0, nl);
      st.pending = st.pending.substring(nl + 1);
      const lines = complete.split("\n");
      for (let i = 0; i < lines.length; i++) emit(fd, lines[i]);
    } else if (st.pending.length >= pendingLimit) {
      emit(fd, st.pending);
      st.pending = "";
    }
    return bytes.length;
  }

  function writeSync(fd, buf, offset, length, position) {
    if (typeof offset === "number" && typeof length === "number") {
      buf = sliceBuf(buf, offset, length);
    }
    if (fd === 1 || fd === 2) {
      if (position != null) throw enosys();
      return writeStdio(fd, buf);
    }
    return emFS().write(streamOf(fd), buf, 0, buf.length, position == null ? undefined : position);
  }

  const asyncHostMethods = { fsync: true, read: true };

  function addSyncMethods(fs) {
    const names = Object.keys(fs);
    for (let i = 0; i < names.length; i++) {
      const name = names[i];
      if (asyncHostMethods[name] || name === "constants" || name.endsWith("Sync") || typeof fs[name] !== "function") continue;
      if (fs[name + "Sync"]) continue;
      const asyncFn = fs[name];
      fs[name + "Sync"] = function () {
        const args = Array.prototype.slice.call(arguments);
        let result, error, called = false;
        args.push(function (err, val) {
          called = true;
          error = err;
          result = val;
        });
        asyncFn.apply(fs, args);
        if (!called) throw enosys();
        if (error) throw error;
        return result;
      };
    }
  }

  function streamOf(fd) {
    return emFS().getStreamChecked(fd);
  }

  function currentCwd() {
    const fs = currentFS();
    if (fs && typeof fs.cwd === "function") {
      try {
        const dir = fs.cwd();
        if (typeof dir === "string" && dir) return dir;
      } catch (_) {}
    }
    return "/";
  }

  function pathResolve(...pathSegments) {
    let out = currentCwd();
    for (let i = 0; i < pathSegments.length; i++) {
      const seg = pathSegments[i];
      if (!seg) continue;
      const part = String(seg);
      if (part.charCodeAt(0) === 47) out = part;
      else out = out.replace(/\/+$/, "") + "/" + part.replace(/^\/+/, "");
    }
    const parts = [];
    const bits = String(out || "/").split("/");
    for (let i = 0; i < bits.length; i++) {
      const p = bits[i];
      if (p === "" || p === ".") continue;
      if (p === "..") {
        if (parts.length) parts.pop();
        continue;
      }
      parts.push(p);
    }
    return "/" + parts.join("/");
  }

  global.fs = {
    constants: {
      O_RDONLY: 0,
      O_WRONLY: 1,
      O_RDWR: 2,
      O_CREAT: 64,
      O_EXCL: 128,
      O_TRUNC: 512,
      O_APPEND: 1024,
      O_DIRECTORY: 65536,
    },
    writeSync,
    write(fd, buf, offset, length, position, callback) {
      call(callback, () => writeSync(fd, buf, offset, length, position));
    },
    read(fd, buffer, offset, length, position, callback) {
      call(callback, () => emFS().read(streamOf(fd), buffer, offset, length, position == null ? undefined : position));
    },
    open(path, flags, mode, callback) {
      call(callback, () => emFS().open(path, flags, mode).fd);
    },
    close(fd, callback) {
      if (fd === 1 || fd === 2) {
        flushStdio(fd);
        callback(null);
        return;
      }
      call(callback, () => {
        emFS().close(streamOf(fd));
      });
    },
    fsync(fd, callback) {
      if (fd === 1 || fd === 2) {
        flushStdio(fd);
        callback(null);
        return;
      }
      call(callback, () => {
        const fs = emFS();
        if (fs.fsync) fs.fsync(streamOf(fd));
      });
    },
    stat(path, callback) { call(callback, () => statObject(emFS().stat(path))); },
    lstat(path, callback) { call(callback, () => statObject(emFS().lstat(path))); },
    fstat(fd, callback) { call(callback, () => statObject(emFS().fstat(fd))); },
    mkdir(path, perm, callback) { call(callback, () => { emFS().mkdir(path, perm); }); },
    unlink(path, callback) { call(callback, () => { emFS().unlink(path); }); },
    rmdir(path, callback) { call(callback, () => { emFS().rmdir(path); }); },
    rename(from, to, callback) { call(callback, () => { emFS().rename(from, to); }); },
    truncate(path, length, callback) { call(callback, () => { emFS().truncate(path, length); }); },
    ftruncate(fd, length, callback) { call(callback, () => { emFS().ftruncate(fd, length); }); },
    chmod(path, mode, callback) { call(callback, () => { emFS().chmod(path, mode); }); },
    fchmod(fd, mode, callback) { call(callback, () => { emFS().fchmod(fd, mode); }); },
    chown(path, uid, gid, callback) { call(callback, () => { emFS().chown(path, uid, gid); }); },
    fchown(fd, uid, gid, callback) { call(callback, () => { emFS().fchown(fd, uid, gid); }); },
    lchown(path, uid, gid, callback) { call(callback, () => { emFS().lchown(path, uid, gid); }); },
    readdir(path, callback) {
      call(callback, () => emFS().readdir(path).filter((x) => x !== "." && x !== ".."));
    },
    readlink(path, callback) { call(callback, () => emFS().readlink(path)); },
    link(from, to, callback) { call(callback, () => { emFS().link(from, to); }); },
    symlink(from, to, callback) { call(callback, () => { emFS().symlink(from, to); }); },
    utimes(path, atime, mtime, callback) {
      call(callback, () => {
        emFS().utime(path, timeMs(atime), timeMs(mtime));
      });
    },
  };
  addSyncMethods(global.fs);

  if (!global.process) {
    global.process = {
      getuid() { return 0; },
      getgid() { return 0; },
      geteuid() { return 0; },
      getegid() { return 0; },
      getgroups() { return []; },
      pid: 1,
      ppid: 0,
      platform: "browser",
      umask(mask) {
        const old = umaskValue;
        if (mask != null) umaskValue = mask;
        return old;
      },
      cwd() { return currentCwd(); },
      chdir(path) {
        const fs = currentFS();
        if (fs && fs.chdir) fs.chdir(path);
      },
    };
  }

  if (!global.path) {
    global.path = {
      resolve: pathResolve,
    };
  }
})(globalThis);
