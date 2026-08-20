target triple = "x86_64-unknown-linux-gnu"

@feedback.enabled = hidden constant i1 true

define hidden void @semanticDemand() #0 {
entry:
  ret void
}

attributes #0 = { noinline }
