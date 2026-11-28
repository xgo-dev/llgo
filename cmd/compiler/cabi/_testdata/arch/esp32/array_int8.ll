; ModuleID = '../../wrap/array_int8.c'
source_filename = "../../wrap/array_int8.c"
target datalayout = "e-m:e-p:32:32-v1:8:8-i64:64-i128:128-n32"
target triple = "xtensa-esp-unknown-elf"

%struct.array1 = type { [1 x i8] }
%struct.array2 = type { [2 x i8] }
%struct.array3 = type { [3 x i8] }
%struct.array4 = type { [4 x i8] }
%struct.array5 = type { [5 x i8] }
%struct.array6 = type { [6 x i8] }
%struct.array7 = type { [7 x i8] }
%struct.array8 = type { [8 x i8] }
%struct.array9 = type { [9 x i8] }
%struct.array10 = type { [10 x i8] }
%struct.array11 = type { [11 x i8] }
%struct.array12 = type { [12 x i8] }
%struct.array13 = type { [13 x i8] }
%struct.array14 = type { [14 x i8] }
%struct.array15 = type { [15 x i8] }
%struct.array16 = type { [16 x i8] }
%struct.array17 = type { [17 x i8] }
%struct.array18 = type { [18 x i8] }
%struct.array19 = type { [19 x i8] }
%struct.array20 = type { [20 x i8] }

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1(ptr dead_on_unwind noalias writable sret(%struct.array1) align 1 %0, i32 %1) #0 {
  %3 = alloca %struct.array1, align 1
  %4 = getelementptr inbounds %struct.array1, ptr %3, i32 0, i32 0
  %5 = trunc i32 %1 to i8
  store i8 %5, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 1, i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i32(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local void @demo2(ptr dead_on_unwind noalias writable sret(%struct.array2) align 1 %0, i32 %1) #0 {
  %3 = alloca %struct.array2, align 1
  %4 = getelementptr inbounds %struct.array2, ptr %3, i32 0, i32 0
  %5 = trunc i32 %1 to i16
  store i16 %5, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 2, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo3(ptr dead_on_unwind noalias writable sret(%struct.array3) align 1 %0, i32 %1) #0 {
  %3 = alloca %struct.array3, align 1
  %4 = getelementptr inbounds %struct.array3, ptr %3, i32 0, i32 0
  %5 = trunc i32 %1 to i24
  store i24 %5, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 3, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo4(ptr dead_on_unwind noalias writable sret(%struct.array4) align 1 %0, i32 %1) #0 {
  %3 = alloca %struct.array4, align 1
  %4 = getelementptr inbounds %struct.array4, ptr %3, i32 0, i32 0
  store i32 %1, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 4, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo5([2 x i32] %0) #0 {
  %2 = alloca %struct.array5, align 1
  %3 = alloca %struct.array5, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  %6 = getelementptr inbounds %struct.array5, ptr %3, i32 0, i32 0
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 5, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 5, i1 false)
  %7 = getelementptr inbounds %struct.array5, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 5, i1 false)
  %8 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo6([2 x i32] %0) #0 {
  %2 = alloca %struct.array6, align 1
  %3 = alloca %struct.array6, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  %6 = getelementptr inbounds %struct.array6, ptr %3, i32 0, i32 0
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 6, i1 false)
  %7 = getelementptr inbounds %struct.array6, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 6, i1 false)
  %8 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo7([2 x i32] %0) #0 {
  %2 = alloca %struct.array7, align 1
  %3 = alloca %struct.array7, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  %6 = getelementptr inbounds %struct.array7, ptr %3, i32 0, i32 0
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 7, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 7, i1 false)
  %7 = getelementptr inbounds %struct.array7, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 7, i1 false)
  %8 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo8([2 x i32] %0) #0 {
  %2 = alloca %struct.array8, align 1
  %3 = alloca %struct.array8, align 1
  %4 = getelementptr inbounds %struct.array8, ptr %3, i32 0, i32 0
  store [2 x i32] %0, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 8, i1 false)
  %5 = getelementptr inbounds %struct.array8, ptr %2, i32 0, i32 0
  %6 = load [2 x i32], ptr %5, align 1
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo9([3 x i32] %0) #0 {
  %2 = alloca %struct.array9, align 1
  %3 = alloca %struct.array9, align 1
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  %6 = getelementptr inbounds %struct.array9, ptr %3, i32 0, i32 0
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 9, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 9, i1 false)
  %7 = getelementptr inbounds %struct.array9, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 9, i1 false)
  %8 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo10([3 x i32] %0) #0 {
  %2 = alloca %struct.array10, align 1
  %3 = alloca %struct.array10, align 1
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  %6 = getelementptr inbounds %struct.array10, ptr %3, i32 0, i32 0
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 10, i1 false)
  %7 = getelementptr inbounds %struct.array10, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 10, i1 false)
  %8 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo11([3 x i32] %0) #0 {
  %2 = alloca %struct.array11, align 1
  %3 = alloca %struct.array11, align 1
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  %6 = getelementptr inbounds %struct.array11, ptr %3, i32 0, i32 0
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 11, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 11, i1 false)
  %7 = getelementptr inbounds %struct.array11, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 11, i1 false)
  %8 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo12([3 x i32] %0) #0 {
  %2 = alloca %struct.array12, align 1
  %3 = alloca %struct.array12, align 1
  %4 = getelementptr inbounds %struct.array12, ptr %3, i32 0, i32 0
  store [3 x i32] %0, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 12, i1 false)
  %5 = getelementptr inbounds %struct.array12, ptr %2, i32 0, i32 0
  %6 = load [3 x i32], ptr %5, align 1
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo13([4 x i32] %0) #0 {
  %2 = alloca %struct.array13, align 1
  %3 = alloca %struct.array13, align 1
  %4 = alloca [4 x i32], align 4
  %5 = alloca [4 x i32], align 4
  %6 = getelementptr inbounds %struct.array13, ptr %3, i32 0, i32 0
  store [4 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 13, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 13, i1 false)
  %7 = getelementptr inbounds %struct.array13, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 13, i1 false)
  %8 = load [4 x i32], ptr %5, align 4
  ret [4 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo14([4 x i32] %0) #0 {
  %2 = alloca %struct.array14, align 1
  %3 = alloca %struct.array14, align 1
  %4 = alloca [4 x i32], align 4
  %5 = alloca [4 x i32], align 4
  %6 = getelementptr inbounds %struct.array14, ptr %3, i32 0, i32 0
  store [4 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 14, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 14, i1 false)
  %7 = getelementptr inbounds %struct.array14, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 14, i1 false)
  %8 = load [4 x i32], ptr %5, align 4
  ret [4 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo15([4 x i32] %0) #0 {
  %2 = alloca %struct.array15, align 1
  %3 = alloca %struct.array15, align 1
  %4 = alloca [4 x i32], align 4
  %5 = alloca [4 x i32], align 4
  %6 = getelementptr inbounds %struct.array15, ptr %3, i32 0, i32 0
  store [4 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %6, ptr align 4 %4, i32 15, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 15, i1 false)
  %7 = getelementptr inbounds %struct.array15, ptr %2, i32 0, i32 0
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %7, i32 15, i1 false)
  %8 = load [4 x i32], ptr %5, align 4
  ret [4 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo16([4 x i32] %0) #0 {
  %2 = alloca %struct.array16, align 1
  %3 = alloca %struct.array16, align 1
  %4 = getelementptr inbounds %struct.array16, ptr %3, i32 0, i32 0
  store [4 x i32] %0, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 16, i1 false)
  %5 = getelementptr inbounds %struct.array16, ptr %2, i32 0, i32 0
  %6 = load [4 x i32], ptr %5, align 1
  ret [4 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(ptr dead_on_unwind noalias writable sret(%struct.array17) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.array17, align 1
  %4 = alloca [5 x i32], align 4
  %5 = getelementptr inbounds %struct.array17, ptr %3, i32 0, i32 0
  store [5 x i32] %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %5, ptr align 4 %4, i32 17, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 17, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(ptr dead_on_unwind noalias writable sret(%struct.array18) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.array18, align 1
  %4 = alloca [5 x i32], align 4
  %5 = getelementptr inbounds %struct.array18, ptr %3, i32 0, i32 0
  store [5 x i32] %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %5, ptr align 4 %4, i32 18, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(ptr dead_on_unwind noalias writable sret(%struct.array19) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.array19, align 1
  %4 = alloca [5 x i32], align 4
  %5 = getelementptr inbounds %struct.array19, ptr %3, i32 0, i32 0
  store [5 x i32] %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %5, ptr align 4 %4, i32 19, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %3, i32 19, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(ptr dead_on_unwind noalias writable sret(%struct.array20) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.array20, align 1
  %4 = getelementptr inbounds %struct.array20, ptr %3, i32 0, i32 0
  store [5 x i32] %1, ptr %4, align 1
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
