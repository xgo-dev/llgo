; ModuleID = '../../wrap/basic.c'
source_filename = "../../wrap/basic.c"
target datalayout = "e-m:e-p:32:32-v1:8:8-i64:64-i128:128-n32"
target triple = "xtensa-esp-unknown-elf"

; Function Attrs: noinline nounwind optnone
define dso_local zeroext i8 @basic_int8(i8 noundef zeroext %0) #0 {
  %2 = alloca i8, align 1
  store i8 %0, ptr %2, align 1
  %3 = load i8, ptr %2, align 1
  ret i8 %3
}

; Function Attrs: noinline nounwind optnone
define dso_local signext i16 @basic_int16(i16 noundef signext %0) #0 {
  %2 = alloca i16, align 2
  store i16 %0, ptr %2, align 2
  %3 = load i16, ptr %2, align 2
  ret i16 %3
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @basic_int32(i32 noundef %0) #0 {
  %2 = alloca i32, align 4
  store i32 %0, ptr %2, align 4
  %3 = load i32, ptr %2, align 4
  ret i32 %3
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @basic_int64(i64 noundef %0) #0 {
  %2 = alloca i64, align 8
  store i64 %0, ptr %2, align 8
  %3 = load i64, ptr %2, align 8
  ret i64 %3
}

; Function Attrs: noinline nounwind optnone
define dso_local float @basic_float32(i32 noundef %0) #0 {
  %2 = alloca float, align 4
  %3 = alloca float, align 4
  store i32 %0, ptr %2, align 4
  %4 = load float, ptr %2, align 4
  store float %4, ptr %3, align 4
  %5 = load float, ptr %3, align 4
  ret float %5
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @basic_float64(i64 noundef %0) #0 {
  %2 = alloca double, align 8
  %3 = alloca double, align 8
  %4 = alloca double, align 8
  store i64 %0, ptr %3, align 8
  %5 = load double, ptr %3, align 8
  store double %5, ptr %4, align 8
  %6 = load double, ptr %4, align 8
  store double %6, ptr %2, align 8
  %7 = load i64, ptr %2, align 8
  ret i64 %7
}

; Function Attrs: noinline nounwind optnone
define dso_local ptr @basic_pointer(i32 noundef %0) #0 {
  %2 = alloca ptr, align 4
  %3 = alloca ptr, align 4
  %4 = inttoptr i32 %0 to ptr
  store ptr %4, ptr %2, align 4
  %5 = load ptr, ptr %2, align 4
  store ptr %5, ptr %3, align 4
  %6 = load ptr, ptr %3, align 4
  ret ptr %6
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+atomctl,+bool,+clamps,+coprocessor,+debug,+density,+dfpaccel,+div32,+exception,+fp,+highpriinterrupts,+interrupt,+loop,+mac16,+memctl,+minmax,+miscsr,+mul32,+mul32high,+nsa,+prid,+regprotect,+rvector,+s32c1i,+sext,+threadptr,+timerint,+windowed" }

!llvm.module.flags = !{!0, !1}
!llvm.ident = !{!2}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 7, !"frame-pointer", i32 2}
!2 = !{!"clang version 19.1.2 (https://github.com/espressif/llvm-project 510a078c1ad4aee4460818bcb38ff0ba3fbf6a83)"}
