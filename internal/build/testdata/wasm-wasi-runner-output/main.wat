(module
  (import "wasi_snapshot_preview1" "fd_write"
    (func $fd_write (param i32 i32 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
  (memory (export "memory") 1)
  (data (i32.const 32) "guest stdout\0a")
  ;; Program output that resembles an engine diagnostic must not be filtered.
  (data (i32.const 128) "{\22level\22:\22WARN\22,\22target\22:\22wasmer\22,\22fields\22:{\22message\22:\22guest stderr\22}}\0a")
  (func (export "_start")
    (i32.store (i32.const 0) (i32.const 32))
    (i32.store (i32.const 4) (i32.const 13))
    (drop (call $fd_write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 8)))
    (i32.store (i32.const 0) (i32.const 128))
    (i32.store (i32.const 4) (i32.const 71))
    (drop (call $fd_write (i32.const 2) (i32.const 0) (i32.const 1) (i32.const 8)))
    (call $exit (i32.const 7)))
)
