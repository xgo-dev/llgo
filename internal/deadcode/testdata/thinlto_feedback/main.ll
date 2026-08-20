target triple = "x86_64-unknown-linux-gnu"

@feedback.enabled = external hidden constant i1

declare hidden void @semanticDemand()

define i32 @main() {
entry:
  %enabled = load i1, ptr @feedback.enabled
  br i1 %enabled, label %demand, label %done

demand:
  call void @semanticDemand()
  br label %done

done:
  ret i32 0
}
