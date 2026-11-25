; ModuleID = '../../wrap/struct_int8.c'
source_filename = "../../wrap/struct_int8.c"
target datalayout = "e-m:e-p:32:32-v1:8:8-i64:64-i128:128-n32"
target triple = "xtensa-esp-unknown-elf"

%struct.point1 = type { i8 }
%struct.point2 = type { i8, i8 }
%struct.point3 = type { i8, i8, i8 }
%struct.point4 = type { i8, i8, i8, i8 }
%struct.point5 = type { i8, i8, i8, i8, i8 }
%struct.point6 = type { i8, i8, i8, i8, i8, i8 }
%struct.point7 = type { i8, i8, i8, i8, i8, i8, i8 }
%struct.point8 = type { i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point9 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point10 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point11 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point12 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point13 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point14 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point15 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point16 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point17 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point18 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point19 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point20 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1(ptr dead_on_unwind noalias writable sret(%struct.point1) align 1 %0, i32 %1) #0 {
  %3 = alloca %struct.point1, align 1
  %4 = getelementptr inbounds %struct.point1, ptr %3, i32 0, i32 0
  %5 = trunc i32 %1 to i8
  store i8 %5, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 1, i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i32(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local void @demo2(ptr dead_on_unwind noalias writable sret(%struct.point2) align 1 %0, i32 %1) #0 {
  %3 = alloca %struct.point2, align 1
  %4 = trunc i32 %1 to i16
  store i16 %4, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 2, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo3(ptr dead_on_unwind noalias writable sret(%struct.point3) align 1 %0, i32 %1) #0 {
  %3 = alloca %struct.point3, align 1
  %4 = trunc i32 %1 to i24
  store i24 %4, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 3, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo4(ptr dead_on_unwind noalias writable sret(%struct.point4) align 1 %0, i32 %1) #0 {
  %3 = alloca %struct.point4, align 1
  store i32 %1, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 4, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo5([2 x i32] %0) #0 {
  %2 = alloca %struct.point5, align 1
  %3 = alloca %struct.point5, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 5, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 5, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 5, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo6([2 x i32] %0) #0 {
  %2 = alloca %struct.point6, align 1
  %3 = alloca %struct.point6, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo7([2 x i32] %0) #0 {
  %2 = alloca %struct.point7, align 1
  %3 = alloca %struct.point7, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 7, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 7, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 7, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo8([2 x i32] %0) #0 {
  %2 = alloca %struct.point8, align 1
  %3 = alloca %struct.point8, align 1
  store [2 x i32] %0, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 1
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo9([3 x i32] %0) #0 {
  %2 = alloca %struct.point9, align 1
  %3 = alloca %struct.point9, align 1
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 9, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 9, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 9, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo10([3 x i32] %0) #0 {
  %2 = alloca %struct.point10, align 1
  %3 = alloca %struct.point10, align 1
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo11([3 x i32] %0) #0 {
  %2 = alloca %struct.point11, align 1
  %3 = alloca %struct.point11, align 1
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 11, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 11, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 11, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo12([3 x i32] %0) #0 {
  %2 = alloca %struct.point12, align 1
  %3 = alloca %struct.point12, align 1
  store [3 x i32] %0, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 1
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo13([4 x i32] %0) #0 {
  %2 = alloca %struct.point13, align 1
  %3 = alloca %struct.point13, align 1
  %4 = alloca [4 x i32], align 4
  %5 = alloca [4 x i32], align 4
  store [4 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 13, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 13, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 13, i1 false)
  %6 = load [4 x i32], ptr %5, align 4
  ret [4 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo14([4 x i32] %0) #0 {
  %2 = alloca %struct.point14, align 1
  %3 = alloca %struct.point14, align 1
  %4 = alloca [4 x i32], align 4
  %5 = alloca [4 x i32], align 4
  store [4 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 14, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 14, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 14, i1 false)
  %6 = load [4 x i32], ptr %5, align 4
  ret [4 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo15([4 x i32] %0) #0 {
  %2 = alloca %struct.point15, align 1
  %3 = alloca %struct.point15, align 1
  %4 = alloca [4 x i32], align 4
  %5 = alloca [4 x i32], align 4
  store [4 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 15, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 15, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 15, i1 false)
  %6 = load [4 x i32], ptr %5, align 4
  ret [4 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo16([4 x i32] %0) #0 {
  %2 = alloca %struct.point16, align 1
  %3 = alloca %struct.point16, align 1
  store [4 x i32] %0, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 1
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(ptr dead_on_unwind noalias writable sret(%struct.point17) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point17, align 1
  %4 = alloca [5 x i32], align 4
  store [5 x i32] %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 17, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 17, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(ptr dead_on_unwind noalias writable sret(%struct.point18) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point18, align 1
  %4 = alloca [5 x i32], align 4
  store [5 x i32] %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 18, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(ptr dead_on_unwind noalias writable sret(%struct.point19) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point19, align 1
  %4 = alloca [5 x i32], align 4
  store [5 x i32] %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 19, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 19, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(ptr dead_on_unwind noalias writable sret(%struct.point20) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point20, align 1
  store [5 x i32] %1, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 20, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+atomctl,+bool,+clamps,+coprocessor,+debug,+density,+dfpaccel,+div32,+exception,+fp,+highpriinterrupts,+interrupt,+loop,+mac16,+memctl,+minmax,+miscsr,+mul32,+mul32high,+nsa,+prid,+regprotect,+rvector,+s32c1i,+sext,+threadptr,+timerint,+windowed" }
attributes #1 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }

!llvm.module.flags = !{!0, !1}
!llvm.ident = !{!2}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 7, !"frame-pointer", i32 2}
!2 = !{!"clang version 19.1.2 (https://github.com/espressif/llvm-project 510a078c1ad4aee4460818bcb38ff0ba3fbf6a83)"}
