// Capture the realm's properties during syscall initialization. Materialize
// emval handles only when Go needs a JavaScript value.
addToLibrary({
  $llgoFSSnapshot: {
    globals: null,
    refs: [],
    error: 0,
    captureGlobals() {
      const state = llgoFSSnapshot;
      if (!state.globals) {
        state.globals = [globalThis, globalThis.Object, globalThis.String,
          globalThis.Array, globalThis.Function, globalThis.NaN, 0,
          globalThis.Uint8Array, globalThis.Uint8ClampedArray];
      }
    },
    // The values match syscall/js.Type.
    type(value) {
      if (value === null) return 1;
      switch (typeof value) {
        case 'boolean': return 2;
        case 'number': return 3;
        case 'string': return 4;
        case 'symbol': return 5;
        case 'object': return 6;
        case 'function': return 7;
        case 'undefined': return 0;
        default: return 8;
      }
    },
    valueError(operation, type) {
      return type === 8 ? 32 : operation | type;
    },
  },
  llgo_fs_snapshot_capture__deps: ['$llgoFSSnapshot'],
  llgo_fs_snapshot_capture(slot) {
    const state = llgoFSSnapshot;
    state.captureGlobals();
    state.refs[slot] = globalThis[['process', 'path', 'fs', 'Uint8Array'][slot]];
  },
  llgo_fs_snapshot_constants__deps: ['$llgoFSSnapshot'],
  llgo_fs_snapshot_constants(slot) {
    const state = llgoFSSnapshot;
    const fs = state.refs[slot];
    const type = state.type(fs);
    if (type !== 6 && type !== 7) return state.valueError(16, type);
    state.constants = fs.constants;
    return 0;
  },
  llgo_fs_snapshot_flag__deps: ['$llgoFSSnapshot'],
  llgo_fs_snapshot_flag(flag) {
    const state = llgoFSSnapshot;
    state.error = 0;
    const type = state.type(state.constants);
    if (type !== 6 && type !== 7) {
      state.error = state.valueError(16, type);
      return NaN;
    }
    const key = ['O_WRONLY', 'O_RDWR', 'O_CREAT', 'O_TRUNC', 'O_APPEND', 'O_EXCL', 'O_DIRECTORY'][flag];
    const value = state.constants[key];
    if (flag === 6 && value === undefined) return -1;
    if (typeof value !== 'number') {
      state.error = state.valueError(8, state.type(value));
      return NaN;
    }
    return value;
  },
  llgo_fs_snapshot_error__deps: ['$llgoFSSnapshot'],
  llgo_fs_snapshot_error() { return llgoFSSnapshot.error; },
  llgo_fs_snapshot_ref__deps: ['$llgoFSSnapshot', '$Emval'],
  llgo_fs_snapshot_ref(slot) {
    return Number(Emval.toHandle(llgoFSSnapshot.refs[slot]));
  },
  llgo_fs_snapshot_global__deps: ['$llgoFSSnapshot', '$Emval'],
  llgo_fs_snapshot_global(slot) {
    const state = llgoFSSnapshot;
    state.captureGlobals();
    return Number(Emval.toHandle(state.globals[slot]));
  },
});
