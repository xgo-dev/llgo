# Browser filesystem host

Emscripten builds include the `syscall/js` filesystem host in their generated
module. Both Memory32 and Memory64 work with the single-worker scheduler and
the bounded worker pool. Applications do not need to load a sidecar manually
to enable the built-in host; the existing `wasm_fs.js` sidecar/attach API remains
available for embedding and custom loaders.

In a browser, Go file operations use the Emscripten runtime thread's `Module.FS`.
Files, file descriptors, and cwd are shared by Go goroutines on different
workers and by C code in the same module. This is Emscripten's virtual
filesystem, normally MEMFS; persistence still requires an application-selected
mount. It is not direct access to the user's host filesystem.

Worker-local Node-shaped methods proxy operations to the runtime thread using
Emscripten's synchronous pthread proxy. Callback values stay in their originating
JS realm. Response buffers are allocated on the requesting worker, because the
browser main thread cannot block on the shared allocator's `Atomics.wait`.
Read/write payloads pass through bounded shared-memory buffers; JSON carries
only metadata. The bridge validates methods and buffer ranges and installs
its worker wrappers once. Read completion copies only the returned bytes into
the caller's requested offset, preserving bytes outside that range.

In the bounded Go scheduler, `syscall/js` uses Go main's worker-zero realm.
Filesystem handles cached in other Go workers' TLS refer to that same realm;
their calls park the calling G and execute on worker zero before entering the
filesystem bridge. The bridge's browser runtime-thread proxy remains necessary
for `Module.FS` and Go/C interoperability.

Node continues using `node:fs`. Its worker `process.cwd/chdir` methods proxy to
the runtime thread so cwd changes are visible across workers. Node's C filesystem
remains Emscripten's virtual filesystem, as before.

`dev/test_wasm_workers.sh` builds and runs the filesystem fixture with both
memory widths and worker counts 1/2 in Node and Chrome. It verifies distinct
scheduler workers, shared files and descriptors, positioned reads, stat after
unlink, ENOENT, cross-worker cwd, GC, and browser Go/C read/write interoperability.
The existing callback and worker tests also verify per-thread bridge installation.
