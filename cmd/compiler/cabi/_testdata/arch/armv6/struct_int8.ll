; ModuleID = '../../wrap/struct_int8.c'
source_filename = "../../wrap/struct_int8.c"
target datalayout = "e-m:e-p:32:32-Fi8-i64:64-v128:64:128-a:0:32-n32-S64"
target triple = "armv6kz-unknown-linux-gnueabihf"

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
define dso_local i8 @demo1([1 x i32] %0) #0 {
  %2 = alloca %struct.point1, align 1
  %3 = alloca %struct.point1, align 1
  %4 = alloca [1 x i32], align 4
  %5 = getelementptr inbounds %struct.point1, %struct.point1* %3, i32 0, i32 0
  store [1 x i32] %0, [1 x i32]* %4, align 4
  %6 = bitcast [1 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 1, i1 false)
  %7 = bitcast %struct.point1* %2 to i8*
  %8 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 1, i1 false)
  %9 = getelementptr inbounds %struct.point1, %struct.point1* %2, i32 0, i32 0
  %10 = load i8, i8* %9, align 1
  ret i8 %10
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local i16 @demo2([1 x i32] %0) #0 {
  %2 = alloca %struct.point2, align 1
  %3 = alloca %struct.point2, align 1
  %4 = alloca [1 x i32], align 4
  store [1 x i32] %0, [1 x i32]* %4, align 4
  %5 = bitcast %struct.point2* %3 to i8*
  %6 = bitcast [1 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 2, i1 false)
  %7 = bitcast %struct.point2* %2 to i8*
  %8 = bitcast %struct.point2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 2, i1 false)
  %9 = bitcast %struct.point2* %2 to i16*
  %10 = load i16, i16* %9, align 1
  ret i16 %10
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo3([1 x i32] %0) #0 {
  %2 = alloca %struct.point3, align 1
  %3 = alloca %struct.point3, align 1
  %4 = alloca [1 x i32], align 4
  %5 = alloca i32, align 4
  store [1 x i32] %0, [1 x i32]* %4, align 4
  %6 = bitcast %struct.point3* %3 to i8*
  %7 = bitcast [1 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %6, i8* align 4 %7, i32 3, i1 false)
  %8 = bitcast %struct.point3* %2 to i8*
  %9 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %8, i8* align 1 %9, i32 3, i1 false)
  %10 = bitcast i32* %5 to i8*
  %11 = bitcast %struct.point3* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %10, i8* align 1 %11, i32 3, i1 false)
  %12 = load i32, i32* %5, align 4
  ret i32 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo4([1 x i32] %0) #0 {
  %2 = alloca %struct.point4, align 1
  %3 = alloca %struct.point4, align 1
  %4 = bitcast %struct.point4* %3 to [1 x i32]*
  store [1 x i32] %0, [1 x i32]* %4, align 1
  %5 = bitcast %struct.point4* %2 to i8*
  %6 = bitcast %struct.point4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 1 %6, i32 4, i1 false)
  %7 = bitcast %struct.point4* %2 to i32*
  %8 = load i32, i32* %7, align 1
  ret i32 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo5(%struct.point5* noalias sret(%struct.point5) align 1 %0, [2 x i32] %1) #0 {
  %3 = alloca %struct.point5, align 1
  %4 = alloca [2 x i32], align 4
  store [2 x i32] %1, [2 x i32]* %4, align 4
  %5 = bitcast %struct.point5* %3 to i8*
  %6 = bitcast [2 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 5, i1 false)
  %7 = bitcast %struct.point5* %0 to i8*
  %8 = bitcast %struct.point5* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 5, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo6(%struct.point6* noalias sret(%struct.point6) align 1 %0, [2 x i32] %1) #0 {
  %3 = alloca %struct.point6, align 1
  %4 = alloca [2 x i32], align 4
  store [2 x i32] %1, [2 x i32]* %4, align 4
  %5 = bitcast %struct.point6* %3 to i8*
  %6 = bitcast [2 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 6, i1 false)
  %7 = bitcast %struct.point6* %0 to i8*
  %8 = bitcast %struct.point6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(%struct.point7* noalias sret(%struct.point7) align 1 %0, [2 x i32] %1) #0 {
  %3 = alloca %struct.point7, align 1
  %4 = alloca [2 x i32], align 4
  store [2 x i32] %1, [2 x i32]* %4, align 4
  %5 = bitcast %struct.point7* %3 to i8*
  %6 = bitcast [2 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 7, i1 false)
  %7 = bitcast %struct.point7* %0 to i8*
  %8 = bitcast %struct.point7* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 7, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo8(%struct.point8* noalias sret(%struct.point8) align 1 %0, [2 x i32] %1) #0 {
  %3 = alloca %struct.point8, align 1
  %4 = bitcast %struct.point8* %3 to [2 x i32]*
  store [2 x i32] %1, [2 x i32]* %4, align 1
  %5 = bitcast %struct.point8* %0 to i8*
  %6 = bitcast %struct.point8* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 1 %6, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(%struct.point9* noalias sret(%struct.point9) align 1 %0, [3 x i32] %1) #0 {
  %3 = alloca %struct.point9, align 1
  %4 = alloca [3 x i32], align 4
  store [3 x i32] %1, [3 x i32]* %4, align 4
  %5 = bitcast %struct.point9* %3 to i8*
  %6 = bitcast [3 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 9, i1 false)
  %7 = bitcast %struct.point9* %0 to i8*
  %8 = bitcast %struct.point9* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 9, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(%struct.point10* noalias sret(%struct.point10) align 1 %0, [3 x i32] %1) #0 {
  %3 = alloca %struct.point10, align 1
  %4 = alloca [3 x i32], align 4
  store [3 x i32] %1, [3 x i32]* %4, align 4
  %5 = bitcast %struct.point10* %3 to i8*
  %6 = bitcast [3 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 10, i1 false)
  %7 = bitcast %struct.point10* %0 to i8*
  %8 = bitcast %struct.point10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(%struct.point11* noalias sret(%struct.point11) align 1 %0, [3 x i32] %1) #0 {
  %3 = alloca %struct.point11, align 1
  %4 = alloca [3 x i32], align 4
  store [3 x i32] %1, [3 x i32]* %4, align 4
  %5 = bitcast %struct.point11* %3 to i8*
  %6 = bitcast [3 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 11, i1 false)
  %7 = bitcast %struct.point11* %0 to i8*
  %8 = bitcast %struct.point11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 11, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(%struct.point12* noalias sret(%struct.point12) align 1 %0, [3 x i32] %1) #0 {
  %3 = alloca %struct.point12, align 1
  %4 = bitcast %struct.point12* %3 to [3 x i32]*
  store [3 x i32] %1, [3 x i32]* %4, align 1
  %5 = bitcast %struct.point12* %0 to i8*
  %6 = bitcast %struct.point12* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 1 %6, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(%struct.point13* noalias sret(%struct.point13) align 1 %0, [4 x i32] %1) #0 {
  %3 = alloca %struct.point13, align 1
  %4 = alloca [4 x i32], align 4
  store [4 x i32] %1, [4 x i32]* %4, align 4
  %5 = bitcast %struct.point13* %3 to i8*
  %6 = bitcast [4 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 13, i1 false)
  %7 = bitcast %struct.point13* %0 to i8*
  %8 = bitcast %struct.point13* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 13, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(%struct.point14* noalias sret(%struct.point14) align 1 %0, [4 x i32] %1) #0 {
  %3 = alloca %struct.point14, align 1
  %4 = alloca [4 x i32], align 4
  store [4 x i32] %1, [4 x i32]* %4, align 4
  %5 = bitcast %struct.point14* %3 to i8*
  %6 = bitcast [4 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 14, i1 false)
  %7 = bitcast %struct.point14* %0 to i8*
  %8 = bitcast %struct.point14* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 14, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(%struct.point15* noalias sret(%struct.point15) align 1 %0, [4 x i32] %1) #0 {
  %3 = alloca %struct.point15, align 1
  %4 = alloca [4 x i32], align 4
  store [4 x i32] %1, [4 x i32]* %4, align 4
  %5 = bitcast %struct.point15* %3 to i8*
  %6 = bitcast [4 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 15, i1 false)
  %7 = bitcast %struct.point15* %0 to i8*
  %8 = bitcast %struct.point15* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 15, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(%struct.point16* noalias sret(%struct.point16) align 1 %0, [4 x i32] %1) #0 {
  %3 = alloca %struct.point16, align 1
  %4 = bitcast %struct.point16* %3 to [4 x i32]*
  store [4 x i32] %1, [4 x i32]* %4, align 1
  %5 = bitcast %struct.point16* %0 to i8*
  %6 = bitcast %struct.point16* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 1 %6, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(%struct.point17* noalias sret(%struct.point17) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point17, align 1
  %4 = alloca [5 x i32], align 4
  store [5 x i32] %1, [5 x i32]* %4, align 4
  %5 = bitcast %struct.point17* %3 to i8*
  %6 = bitcast [5 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 17, i1 false)
  %7 = bitcast %struct.point17* %0 to i8*
  %8 = bitcast %struct.point17* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 17, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(%struct.point18* noalias sret(%struct.point18) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point18, align 1
  %4 = alloca [5 x i32], align 4
  store [5 x i32] %1, [5 x i32]* %4, align 4
  %5 = bitcast %struct.point18* %3 to i8*
  %6 = bitcast [5 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 18, i1 false)
  %7 = bitcast %struct.point18* %0 to i8*
  %8 = bitcast %struct.point18* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(%struct.point19* noalias sret(%struct.point19) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point19, align 1
  %4 = alloca [5 x i32], align 4
  store [5 x i32] %1, [5 x i32]* %4, align 4
  %5 = bitcast %struct.point19* %3 to i8*
  %6 = bitcast [5 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 4 %6, i32 19, i1 false)
  %7 = bitcast %struct.point19* %0 to i8*
  %8 = bitcast %struct.point19* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 19, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(%struct.point20* noalias sret(%struct.point20) align 1 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point20, align 1
  %4 = bitcast %struct.point20* %3 to [5 x i32]*
  store [5 x i32] %1, [5 x i32]* %4, align 1
  %5 = bitcast %struct.point20* %0 to i8*
  %6 = bitcast %struct.point20* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 1 %6, i32 20, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="arm1176jzf-s" "target-features"="+armv6kz,+dsp,+fp64,+strict-align,+vfp2,+vfp2sp,-aes,-d32,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-neon,-sha2,-thumb-mode,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3, !4, !5, !6, !7, !8}
!llvm.ident = !{!9}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"min_enum_size", i32 4}
!2 = !{i32 8, !"branch-target-enforcement", i32 0}
!3 = !{i32 8, !"sign-return-address", i32 0}
!4 = !{i32 8, !"sign-return-address-all", i32 0}
!5 = !{i32 8, !"sign-return-address-with-bkey", i32 0}
!6 = !{i32 7, !"PIC Level", i32 2}
!7 = !{i32 7, !"PIE Level", i32 2}
!8 = !{i32 7, !"frame-pointer", i32 2}
!9 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
