(module
  (memory (export "memory") 1)
  (tag $exception (param i32))
  (func $raise (throw $exception (i32.const 42)))
  (func (export "_start") (call $raise)))
