/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0.
 *
 * LLGo's GOOS=js GOARCH=wasm host. It preserves Go's wasm_exec.js API:
 * new Go(), go.importObject, and go.run(instance). llgo build concatenates
 * the Emscripten factory ahead of this file. new Go() uses the last loaded
 * factory; new Go({ wasm: "app.wasm" }) selects a registered program.
 * A later host script only registers its factory and does not redefine Go.
 */

(() => {
	"use strict";

	const global = globalThis;

	// Preserve the browser host contract of Go's wasm_exec.js. The standard
	// library syscall/fs_js.go reads these globals during program startup.
	const enosys = () => {
		const error = new Error("not implemented");
		error.code = "ENOSYS";
		return error;
	};
	if (!global.fs) {
		let output = "";
		const decoder = new TextDecoder("utf-8");
		global.fs = {
			constants: { O_WRONLY: -1, O_RDWR: -1, O_CREAT: -1, O_TRUNC: -1, O_APPEND: -1, O_EXCL: -1, O_DIRECTORY: -1 },
			writeSync(fd, buffer) {
				output += decoder.decode(buffer);
				const newline = output.lastIndexOf("\n");
				if (newline !== -1) {
					console.log(output.substring(0, newline));
					output = output.substring(newline + 1);
				}
				return buffer.length;
			},
			write(fd, buffer, offset, length, position, callback) {
				if (offset !== 0 || length !== buffer.length || position !== null) return callback(enosys());
				callback(null, this.writeSync(fd, buffer));
			},
			fsync(fd, callback) { callback(null); },
		};
		for (const name of ["chmod", "chown", "close", "fchmod", "fchown", "fstat", "ftruncate", "lchown", "link", "lstat", "mkdir", "open", "read", "readdir", "readlink", "rename", "rmdir", "stat", "symlink", "truncate", "unlink", "utimes"]) {
			global.fs[name] = (...args) => args[args.length - 1](enosys());
		}
	}
	if (!global.process) {
		global.process = {
			getuid() { return -1; }, getgid() { return -1; }, geteuid() { return -1; }, getegid() { return -1; },
			getgroups() { throw enosys(); }, pid: -1, ppid: -1,
			umask() { throw enosys(); }, cwd() { throw enosys(); }, chdir() { throw enosys(); },
		};
	}
	if (!global.path) global.path = { resolve: (...segments) => segments.join("/") };
	if (!global.crypto) throw new Error("globalThis.crypto is not available, polyfill required (crypto.getRandomValues only)");
	if (!global.performance) throw new Error("globalThis.performance is not available, polyfill required (performance.now only)");
	if (!global.TextEncoder) throw new Error("globalThis.TextEncoder is not available, polyfill required");
	if (!global.TextDecoder) throw new Error("globalThis.TextDecoder is not available, polyfill required");

	// Chrome rejects TextDecoder.decode on a view of growable Wasm memory.
	// Emscripten's HEAPU8 is such a view when ALLOW_MEMORY_GROWTH is enabled.
	function resizableBuffer(value) {
		if (!value) return null;
		const buffer = value instanceof ArrayBuffer ? value : value.buffer;
		return buffer && (buffer.resizable || buffer.growable) ? buffer : null;
	}
	if (global.TextDecoder && !global.TextDecoder.prototype.__llgoResizableSafe) {
		const NativeTextDecoder = global.TextDecoder;
		class LLGoTextDecoder extends NativeTextDecoder {
			decode(input, options) {
				if (resizableBuffer(input)) {
					const bytes = ArrayBuffer.isView(input)
						? new Uint8Array(input.buffer, input.byteOffset, input.byteLength)
						: new Uint8Array(input);
					input = bytes.slice();
				}
				return super.decode(input, options);
			}
		}
		LLGoTextDecoder.prototype.__llgoResizableSafe = true;
		try {
			global.TextDecoder = LLGoTextDecoder;
		} catch (_) {}
	}

	if (typeof global.Go === "function") {
		return;
	}

	const encoder = new TextEncoder();
	function writeOutput(fd, args) {
		global.fs.writeSync(fd, encoder.encode(args.map(String).join(" ") + "\n"));
	}

	function implementationFactory(options) {
		const name = options && options.wasm;
		if (name) {
			const factory = global.__llgoFactories && global.__llgoFactories[name];
			if (typeof factory !== "function") {
				throw new Error("LLGo: no factory registered for " + name);
			}
			return factory;
		}
		if (typeof global.LLGoModule !== "function") {
			throw new Error("LLGo: wasm_exec.js is missing the bundled LLGoModule factory");
		}
		return global.LLGoModule;
	}

	global.Go = class {
		constructor(options) {
			const nodeArgs = global.process?.versions?.node ? global.process.argv.slice(2) : [];
			this.argv = ["js"].concat(nodeArgs);
			this.env = {};
			this.exit = code => {
				if (code !== 0) {
					console.warn(`exit code: ${code}`);
				}
			};
			this._exited = false;
			this._module = null;
			this._instance = null;
			this._receiveInstance = null;
			this._exitPromise = new Promise(resolve => {
				this._resolveExit = resolve;
			});

			const factory = implementationFactory(options);
			this._factoryPromise = Promise.resolve(factory({
				// Match Go's wasm_exec.js: program output goes through fs.writeSync.
				// Emscripten print/printErr already strip the trailing newline.
				print: (...args) => writeOutput(1, args),
				printErr: (...args) => writeOutput(2, args),
				noInitialRun: true,
				noExitRuntime: false,
				thisProgram: (this.argv && this.argv[0]) || "./this.program",
				instantiateWasm: (imports, receiveInstance) => {
					this.importObject = imports;
					this._receiveInstance = receiveInstance;
				},
				preRun: [module => {
					if (module.ENV != null) {
						Object.assign(module.ENV, this.env);
					}
				}],
				onExit: code => {
					this._notifyExit(code);
				},
			}));

			// Emscripten calls instantiateWasm synchronously while its factory is
			// evaluated. Keep the standard Go API synchronous at this boundary.
			if (this.importObject === undefined) {
				throw new Error("LLGo: wasm_exec.js did not synchronously provide Wasm imports");
			}
		}

		_callMain(argv) {
			const module = this._module;
			const args = argv && argv.length ? argv.map(String) : ["./this.program"];
			if (typeof module._main === "function" && typeof module.stackAlloc === "function" &&
				typeof module.setValue === "function" && typeof module.allocateUTF8 === "function") {
				const ptrSize = 4;
				const argc = args.length;
				const argvPtr = module.stackAlloc((argc + 1) * ptrSize);
				for (let i = 0; i < argc; i++) {
					module.setValue(argvPtr + i * ptrSize, module.allocateUTF8(args[i]), "i32");
				}
				module.setValue(argvPtr + argc * ptrSize, 0, "i32");
				return module._main(argc, argvPtr);
			}
			return module.callMain(args.slice(1));
		}

		_notifyExit(code) {
			if (this._exited) return;
			this._exited = true;
			this.exit(code);
			this._resolveExit(code);
		}

		async run(instance) {
			if (!(instance instanceof WebAssembly.Instance)) {
				throw new Error("Go.run: WebAssembly.Instance expected");
			}
			if (this._instance !== null) {
				throw new Error("Go program has already started");
			}
			this._instance = instance;
			this._receiveInstance(instance);
			try {
				this._module = await this._factoryPromise;
				if (this.argv && this.argv.length > 0) {
					this._module.thisProgram = this.argv[0];
				}
				this._callMain(this.argv);
			} catch (error) {
				if (error?.name === "ExitStatus" && Number.isInteger(error.status)) {
					this._notifyExit(error.status);
				} else {
					throw error;
				}
			}
			if (!this._exited) {
				await new Promise(resolve => {
					const tick = () => {
						if (this._exited) {
							resolve();
							return;
						}
						const asyncify = this._module && this._module.Asyncify;
						if (!(asyncify && asyncify.currData)) {
							this._notifyExit(0);
							resolve();
							return;
						}
						setTimeout(tick, 0);
					};
					tick();
				});
			}
			return await this._exitPromise;
		}
	};
})();
