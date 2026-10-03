;; Exercise the classic interpreter's v128 stack, rewritten bytecode and EH.
;; Distinct high lanes catch accidental 32/64-bit copies of a 128-bit value.
(module
  (type $vector (func (param v128) (result v128)))
  (memory (export "memory") 1 2 shared)
  (global $g (mut v128) (v128.const i32x4 11 22 33 44))
  (table 1 funcref)
  (elem (i32.const 0) $identity)
  (tag $exception (param v128))

  (func $identity (type $vector) local.get 0)
  (func $throw (param v128) local.get 0 throw $exception)
  (func $check (param i32) local.get 0 i32.eqz if unreachable end)
  (func $equal (param v128 v128)
    local.get 0 local.get 1 i32x4.eq i32x4.all_true call $check)

  (func $run (export "run") (result i32) (local $v v128)
    global.get $g
    local.tee $v
    v128.const i32x4 11 22 33 44
    call $equal

    local.get $v
    i32.const 55
    i32x4.replace_lane 3
    local.set $v
    local.get $v
    global.set $g
    global.get $g
    v128.const i32x4 11 22 33 55
    call $equal

    ;; Unaligned memory and indirect vector argument/result.
    i32.const 1 local.get $v v128.store align=1
    i32.const 1 v128.load align=1
    i32.const 0 call_indirect (type $vector)
    local.get $v call $equal

    ;; Both forms of select retain all four lanes and pop both inputs.
    v128.const i32x4 1 2 3 4 local.get $v i32.const 0 select
    local.get $v call $equal
    local.get $v v128.const i32x4 1 2 3 4 i32.const 1 select (result v128)
    local.get $v call $equal

    ;; The block scanner must understand rewritten vector drop/select/global
    ;; instructions even on paths the interpreter never executes.
    block
      br 0
      global.get $g drop
      local.get $v local.get $v i32.const 0 select drop
      local.get $v global.set $g
    end
    local.get $v drop

    block (result v128)
      local.get $v br 0
    end
    local.get $v call $equal

    ;; Propagate a vector payload across a call and through a catch handler.
    try (result v128)
      local.get $v call $throw
      unreachable
    catch $exception
    end
    local.get $v call $equal

    try (result v128)
      try (result v128)
        local.get $v call $throw
        unreachable
      catch $exception
        drop
        rethrow 0
      end
    catch $exception
    end
    local.get $v call $equal

    local.get $v v128.const i32x4 1 2 3 4 i32x4.add
    v128.const i32x4 12 24 36 59 call $equal
    i32.const 0)

  (func (export "_start") call $run drop)
  (func (export "out_of_bounds")
    i32.const 65530 v128.load drop)
)
