target triple = "x86_64-unknown-linux-gnu"

define i32 @main() {
entry:
  ret i32 0
}

define hidden void @deletedDemand() noinline {
entry:
  ret void
}
