// Browser syscall/js hosts share the Emscripten runtime thread's FS, including
// C's file descriptors and cwd. Never create a separate MEMFS on each worker.
addToLibrary({
  $llgoBrowserFS__deps: ['llgo_browser_fs_call', 'llgo_browser_fs_result', '$stringToUTF8', '$lengthBytesUTF8', '$UTF8ToString', 'llgo_browser_fs_malloc', 'llgo_browser_fs_free', '$FS'],
  $llgoBrowserFS__postset: 'llgoBrowserFS.install();',
  $llgoBrowserFS: {
    installed: false,
    methods: {
      fs: ['write', 'writeSync', 'read', 'open', 'close', 'fsync', 'stat', 'lstat', 'fstat',
           'mkdir', 'unlink', 'rmdir', 'rename', 'truncate', 'ftruncate', 'chmod', 'fchmod',
           'chown', 'fchown', 'lchown', 'readdir', 'readlink', 'link', 'symlink', 'utimes'],
      process: ['getuid', 'getgid', 'geteuid', 'getegid', 'getgroups', 'umask', 'cwd', 'chdir'],
      path: ['resolve'],
    },
    allowed(target, name) {
      if (!Object.hasOwn(llgoBrowserFS.methods, target)) return false;
      const methods = llgoBrowserFS.methods[target];
      return methods.includes(name) || (target === 'fs' && name.endsWith('Sync') &&
        !['readSync', 'fsyncSync'].includes(name) && methods.includes(name.slice(0, -4)));
    },
    invoke(target, name, args) {
      let request = 0, response = 0, payload = 0;
      try {
        if (!llgoBrowserFS.allowed(target, name)) {
          throw Object.assign(new Error('unsupported filesystem host method'), { code: 'ENOSYS' });
        }
        let buffer, offset, length;
        const bytes = target === 'fs' && ['read', 'write', 'writeSync'].includes(name);
        if (bytes) {
          buffer = args[1];
          offset = args[2] === undefined ? 0 : args[2];
          length = args[3] === undefined ? buffer?.byteLength - offset : args[3];
          if (!(buffer instanceof Uint8Array) || !Number.isSafeInteger(offset) ||
              !Number.isSafeInteger(length) || offset < 0 || length < 0 ||
              offset > buffer.byteLength || length > buffer.byteLength - offset) {
            throw Object.assign(new Error('invalid filesystem buffer range'), { code: 'EINVAL' });
          }
          // Allocate on the requesting worker: malloc may block. Only metadata
          // crosses JSON; byte payloads use shared linear memory in both widths.
          payload = _llgo_browser_fs_malloc(Math.max(1, length));
          if (!payload) throw Object.assign(new Error('filesystem buffer allocation failed'), { code: 'ENOMEM' });
          const pointer = Number(payload);
          if (name !== 'read') HEAPU8.set(buffer.subarray(offset, offset + length), pointer);
          args = [args[0], { pointer, length }, 0, length, args[4]];
        }
        // Each allocation may collect: JS numbers do not publish Go roots.
        const text = JSON.stringify({ target, name, args });
        const requestSize = lengthBytesUTF8(text) + 1;
        request = _llgo_browser_fs_malloc(requestSize);
        stringToUTF8(text, request, requestSize);
        const size = _llgo_browser_fs_call(request);
        response = _llgo_browser_fs_malloc(size);
        if (!response) throw Object.assign(new Error('filesystem response allocation failed'), { code: 'ENOMEM' });
        _llgo_browser_fs_result(request, response);
        const reply = JSON.parse(UTF8ToString(response));
        if (reply.error) throw Object.assign(new Error(reply.error.message), reply.error);
        if (bytes) {
          if (!Number.isSafeInteger(reply.value) || reply.value < 0 || reply.value > length) {
            throw Object.assign(new Error('invalid filesystem byte count'), { code: 'EIO' });
          }
          if (name === 'read') buffer.set(HEAPU8.subarray(Number(payload), Number(payload) + reply.value), offset);
        }
        const value = reply.value;
        if (name === 'stat' || name === 'lstat' || name === 'fstat' ||
            name === 'statSync' || name === 'lstatSync' || name === 'fstatSync') {
          value.isDirectory = () => (value.mode & 16384) !== 0;
        }
        return value;
      } finally {
        if (request) {
          _llgo_browser_fs_result(request, 0);
          _llgo_browser_fs_free(request);
        }
        if (response) _llgo_browser_fs_free(response);
        if (payload) _llgo_browser_fs_free(payload);
      }
    },
    install() {
      if (llgoBrowserFS.installed) return;
      llgoBrowserFS.installed = true;
      if (ENVIRONMENT_IS_NODE) {
#if PTHREADS
        // Node omits process.chdir on workers. The cwd belongs to the process.
        if (ENVIRONMENT_IS_PTHREAD) {
          globalThis.process.chdir = path => llgoBrowserFS.invoke('process', 'chdir', [path]);
          globalThis.process.cwd = () => llgoBrowserFS.invoke('process', 'cwd', []);
        }
#endif
        return;
      }
      globalThis.llgoAttachWasmFS(Module);
#if PTHREADS
      if (!ENVIRONMENT_IS_PTHREAD) return;
      for (const target of ['fs', 'process', 'path']) {
        const host = globalThis[target];
        for (const name of Object.keys(host)) {
          if (typeof host[name] !== 'function' || !llgoBrowserFS.allowed(target, name)) continue;
          host[name] = (...args) => {
            const callback = typeof args[args.length - 1] === 'function' ? args.pop() : null;
            let value, error;
            try { value = llgoBrowserFS.invoke(target, name, args); }
            catch (e) { error = e; }
            // Callback exceptions belong to Go. Do not call a callback twice.
            if (callback) return callback(error || null, value);
            if (error) throw error;
            return value;
          };
        }
      }
#endif
    },
  },

  // Allocate response storage on the requesting worker. Allocating on the
  // browser main thread can contend on the shared allocator, where Atomics.wait
  // is forbidden. Keep only JS strings here between the two proxy calls.
  $llgoBrowserFSResponses: {},
  llgo_browser_fs_result__deps: ['$llgoBrowserFSResponses', '$stringToUTF8', '$lengthBytesUTF8'],
  llgo_browser_fs_result__proxy: 'sync',
  llgo_browser_fs_result__sig: 'vdd',
  llgo_browser_fs_result(request, response) {
    const text = llgoBrowserFSResponses[request];
    if (response) stringToUTF8(text, response, lengthBytesUTF8(text) + 1);
    delete llgoBrowserFSResponses[request];
  },
  llgo_browser_fs_call__deps: ['$llgoBrowserFS', '$llgoBrowserFSResponses', '$UTF8ToString', '$lengthBytesUTF8'],
  llgo_browser_fs_call__proxy: 'sync',
  // Called only from JS: numeric linear-memory offsets work for both widths.
  llgo_browser_fs_call__sig: 'dd',
  llgo_browser_fs_call(request) {
    let reply;
    try {
      // The requesting worker's payload malloc can grow shared memory.
      // Refresh this realm before inspecting pointers or creating byte views.
      if (typeof updateMemoryViews === 'function') updateMemoryViews();
      const { target, name, args } = JSON.parse(UTF8ToString(request));
      if (typeof target !== 'string' || typeof name !== 'string' ||
          !Array.isArray(args) || !llgoBrowserFS.allowed(target, name)) {
        throw Object.assign(new Error('unsupported filesystem host method'), { code: 'ENOSYS' });
      }
      if (target === 'fs' && ['read', 'write', 'writeSync'].includes(name)) {
        const { pointer, length } = args[1] || {};
        if (!Number.isSafeInteger(pointer) || !Number.isSafeInteger(length) ||
            pointer <= 0 || length < 0 || pointer > HEAPU8.length ||
            length > HEAPU8.length - pointer || args[2] !== 0 || args[3] !== length) {
          throw Object.assign(new Error('invalid filesystem shared buffer'), { code: 'EINVAL' });
        }
        args[1] = HEAPU8.subarray(pointer, pointer + length);
      }
      const host = globalThis[target];
      let value;
      if (target === 'fs' && !name.endsWith('Sync')) {
        let called = false;
        let error;
        host[name](...args, (err, result) => { called = true; error = err; value = result; });
        if (!called) throw new Error('Emscripten FS host did not complete synchronously');
        if (error) throw error;
      } else {
        value = host[name](...args);
      }
      reply = { value };
    } catch (e) {
      reply = { error: { message: e.message, code: e.code || 'EIO' } };
    }
    const text = JSON.stringify(reply);
    llgoBrowserFSResponses[request] = text;
    return lengthBytesUTF8(text) + 1;
  },
});
