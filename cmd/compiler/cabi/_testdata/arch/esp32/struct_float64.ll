; ModuleID = '../../wrap/struct_float64.c'
source_filename = "../../wrap/struct_float64.c"
target datalayout = "e-m:e-p:32:32-v1:8:8-i64:64-i128:128-n32"
target triple = "xtensa-esp-unknown-elf"

%struct.point1 = type { double }
%struct.point2 = type { double, double }
%struct.point3 = type { double, double, double }
%struct.point4 = type { double, double, double, double }
%struct.point5 = type { double, double, double, double, double }
%struct.point6 = type { double, double, double, double, double, double }
%struct.point7 = type { double, double, double, double, double, double, double }
%struct.point8 = type { double, double, double, double, double, double, double, double }
%struct.point9 = type { double, double, double, double, double, double, double, double, double }
%struct.point10 = type { double, double, double, double, double, double, double, double, double, double }
%struct.point11 = type { double, double, double, double, double, double, double, double, double, double, double }
%struct.point12 = type { double, double, double, double, double, double, double, double, double, double, double, double }
%struct.point13 = type { double, double, double, double, double, double, double, double, double, double, double, double, double }
%struct.point14 = type { double, double, double, double, double, double, double, double, double, double, double, double, double, double }
%struct.point15 = type { double, double, double, double, double, double, double, double, double, double, double, double, double, double, double }
%struct.point16 = type { double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double }
%struct.point17 = type { double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double }
%struct.point18 = type { double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double }
%struct.point19 = type { double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double }
%struct.point20 = type { double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double, double }

; Function Attrs: noinline nounwind optnone
define dso_local [1 x i64] @demo1([1 x i64] %0) #0 {
  %2 = alloca %struct.point1, align 8
  %3 = alloca %struct.point1, align 8
  %4 = getelementptr inbounds %struct.point1, ptr %3, i32 0, i32 0
  store [1 x i64] %0, ptr %4, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 8, i1 false)
  %5 = getelementptr inbounds %struct.point1, ptr %2, i32 0, i32 0
  %6 = load [1 x i64], ptr %5, align 8
  ret [1 x i64] %6
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i32(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo2([2 x i64] %0) #0 {
  %2 = alloca %struct.point2, align 8
  %3 = alloca %struct.point2, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo3(ptr dead_on_unwind noalias writable sret(%struct.point3) align 8 %0, [3 x i64] %1) #0 {
  %3 = alloca %struct.point3, align 8
  store [3 x i64] %1, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %3, i32 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo4(ptr dead_on_unwind noalias writable sret(%struct.point4) align 8 %0, ptr noundef byval(%struct.point4) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo5(ptr dead_on_unwind noalias writable sret(%struct.point5) align 8 %0, ptr noundef byval(%struct.point5) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo6(ptr dead_on_unwind noalias writable sret(%struct.point6) align 8 %0, ptr noundef byval(%struct.point6) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(ptr dead_on_unwind noalias writable sret(%struct.point7) align 8 %0, ptr noundef byval(%struct.point7) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo8(ptr dead_on_unwind noalias writable sret(%struct.point8) align 8 %0, ptr noundef byval(%struct.point8) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(ptr dead_on_unwind noalias writable sret(%struct.point9) align 8 %0, ptr noundef byval(%struct.point9) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(ptr dead_on_unwind noalias writable sret(%struct.point10) align 8 %0, ptr noundef byval(%struct.point10) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 80, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(ptr dead_on_unwind noalias writable sret(%struct.point11) align 8 %0, ptr noundef byval(%struct.point11) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 88, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(ptr dead_on_unwind noalias writable sret(%struct.point12) align 8 %0, ptr noundef byval(%struct.point12) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 96, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(ptr dead_on_unwind noalias writable sret(%struct.point13) align 8 %0, ptr noundef byval(%struct.point13) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 104, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(ptr dead_on_unwind noalias writable sret(%struct.point14) align 8 %0, ptr noundef byval(%struct.point14) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 112, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(ptr dead_on_unwind noalias writable sret(%struct.point15) align 8 %0, ptr noundef byval(%struct.point15) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 120, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(ptr dead_on_unwind noalias writable sret(%struct.point16) align 8 %0, ptr noundef byval(%struct.point16) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 128, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(ptr dead_on_unwind noalias writable sret(%struct.point17) align 8 %0, ptr noundef byval(%struct.point17) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 136, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(ptr dead_on_unwind noalias writable sret(%struct.point18) align 8 %0, ptr noundef byval(%struct.point18) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 144, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(ptr dead_on_unwind noalias writable sret(%struct.point19) align 8 %0, ptr noundef byval(%struct.point19) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 152, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(ptr dead_on_unwind noalias writable sret(%struct.point20) align 8 %0, ptr noundef byval(%struct.point20) align 8 %1) #0 {
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %0, ptr align 8 %1, i32 160, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+atomctl,+bool,+clamps,+coprocessor,+debug,+density,+dfpaccel,+div32,+exception,+fp,+highpriinterrupts,+interrupt,+loop,+mac16,+memctl,+minmax,+miscsr,+mul32,+mul32high,+nsa,+prid,+regprotect,+rvector,+s32c1i,+sext,+threadptr,+timerint,+windowed" }
attributes #1 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }

!llvm.module.flags = !{!0, !1}
!llvm.ident = !{!2}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 7, !"frame-pointer", i32 2}
!2 = !{!"clang version 19.1.2 (https://github.com/espressif/llvm-project 510a078c1ad4aee4460818bcb38ff0ba3fbf6a83)"}
