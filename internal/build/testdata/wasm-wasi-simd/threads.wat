(module
  (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
  (import "env" "memory" (memory 1 1 shared))
  (import "wasi" "thread-spawn" (func $spawn (param i32) (result i32)))
  (export "memory" (memory 0))
  ;; Exercise v128 calls, locals and exception payloads in a spawned thread.
  (tag $e (param v128))
  (func $add (param $a v128) (param $b v128) (result v128)
    local.get $a local.get $b i32x4.add)
  (func (export "wasi_thread_start") (param i32 i32) (local $value i32)
    block $caught (result v128)
      try_table (catch $e $caught)
        v128.const i32x4 10 20 30 40
        v128.const i32x4 1 2 3 4
        call $add
        throw $e
      end
      unreachable
    end
    i32x4.extract_lane 3
    local.set $value
    i32.const 0 local.get $value i32.atomic.store
    i32.const 0 i32.const 1 memory.atomic.notify drop)
  (func (export "_start")
    i32.const 0 call $spawn i32.const 0 i32.lt_s
    if unreachable end
    loop $wait
      i32.const 0 i32.atomic.load i32.eqz
      if
        i32.const 0 i32.const 0 i64.const 1000000000 memory.atomic.wait32 drop
        br $wait
      end
    end
    i32.const 0 i32.atomic.load i32.const 44 i32.ne
    if unreachable end))
