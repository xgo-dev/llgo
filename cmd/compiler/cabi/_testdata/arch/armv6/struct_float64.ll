; ModuleID = '../../wrap/struct_float64.c'
source_filename = "../../wrap/struct_float64.c"
target datalayout = "e-m:e-p:32:32-Fi8-i64:64-v128:64:128-a:0:32-n32-S64"
target triple = "armv6kz-unknown-linux-gnueabihf"

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
define dso_local %struct.point1 @demo1(%struct.point1 %0) #0 {
  %2 = alloca %struct.point1, align 8
  %3 = alloca %struct.point1, align 8
  store %struct.point1 %0, %struct.point1* %3, align 8
  %4 = bitcast %struct.point1* %2 to i8*
  %5 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %4, i8* align 8 %5, i32 8, i1 false)
  %6 = load %struct.point1, %struct.point1* %2, align 8
  ret %struct.point1 %6
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local %struct.point2 @demo2(%struct.point2 %0) #0 {
  %2 = alloca %struct.point2, align 8
  %3 = alloca %struct.point2, align 8
  store %struct.point2 %0, %struct.point2* %3, align 8
  %4 = bitcast %struct.point2* %2 to i8*
  %5 = bitcast %struct.point2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %4, i8* align 8 %5, i32 16, i1 false)
  %6 = load %struct.point2, %struct.point2* %2, align 8
  ret %struct.point2 %6
}

; Function Attrs: noinline nounwind optnone
define dso_local %struct.point3 @demo3(%struct.point3 %0) #0 {
  %2 = alloca %struct.point3, align 8
  %3 = alloca %struct.point3, align 8
  store %struct.point3 %0, %struct.point3* %3, align 8
  %4 = bitcast %struct.point3* %2 to i8*
  %5 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %4, i8* align 8 %5, i32 24, i1 false)
  %6 = load %struct.point3, %struct.point3* %2, align 8
  ret %struct.point3 %6
}

; Function Attrs: noinline nounwind optnone
define dso_local %struct.point4 @demo4(%struct.point4 %0) #0 {
  %2 = alloca %struct.point4, align 8
  %3 = alloca %struct.point4, align 8
  store %struct.point4 %0, %struct.point4* %3, align 8
  %4 = bitcast %struct.point4* %2 to i8*
  %5 = bitcast %struct.point4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %4, i8* align 8 %5, i32 32, i1 false)
  %6 = load %struct.point4, %struct.point4* %2, align 8
  ret %struct.point4 %6
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo5(%struct.point5* noalias sret(%struct.point5) align 8 %0, [5 x i64] %1) #0 {
  %3 = alloca %struct.point5, align 8
  %4 = bitcast %struct.point5* %3 to [5 x i64]*
  store [5 x i64] %1, [5 x i64]* %4, align 8
  %5 = bitcast %struct.point5* %0 to i8*
  %6 = bitcast %struct.point5* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo6(%struct.point6* noalias sret(%struct.point6) align 8 %0, [6 x i64] %1) #0 {
  %3 = alloca %struct.point6, align 8
  %4 = bitcast %struct.point6* %3 to [6 x i64]*
  store [6 x i64] %1, [6 x i64]* %4, align 8
  %5 = bitcast %struct.point6* %0 to i8*
  %6 = bitcast %struct.point6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(%struct.point7* noalias sret(%struct.point7) align 8 %0, [7 x i64] %1) #0 {
  %3 = alloca %struct.point7, align 8
  %4 = bitcast %struct.point7* %3 to [7 x i64]*
  store [7 x i64] %1, [7 x i64]* %4, align 8
  %5 = bitcast %struct.point7* %0 to i8*
  %6 = bitcast %struct.point7* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo8(%struct.point8* noalias sret(%struct.point8) align 8 %0, [8 x i64] %1) #0 {
  %3 = alloca %struct.point8, align 8
  %4 = bitcast %struct.point8* %3 to [8 x i64]*
  store [8 x i64] %1, [8 x i64]* %4, align 8
  %5 = bitcast %struct.point8* %0 to i8*
  %6 = bitcast %struct.point8* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(%struct.point9* noalias sret(%struct.point9) align 8 %0, %struct.point9* noundef byval(%struct.point9) align 8 %1) #0 {
  %3 = bitcast %struct.point9* %0 to i8*
  %4 = bitcast %struct.point9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(%struct.point10* noalias sret(%struct.point10) align 8 %0, %struct.point10* noundef byval(%struct.point10) align 8 %1) #0 {
  %3 = bitcast %struct.point10* %0 to i8*
  %4 = bitcast %struct.point10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 80, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(%struct.point11* noalias sret(%struct.point11) align 8 %0, %struct.point11* noundef byval(%struct.point11) align 8 %1) #0 {
  %3 = bitcast %struct.point11* %0 to i8*
  %4 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 88, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(%struct.point12* noalias sret(%struct.point12) align 8 %0, %struct.point12* noundef byval(%struct.point12) align 8 %1) #0 {
  %3 = bitcast %struct.point12* %0 to i8*
  %4 = bitcast %struct.point12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 96, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(%struct.point13* noalias sret(%struct.point13) align 8 %0, %struct.point13* noundef byval(%struct.point13) align 8 %1) #0 {
  %3 = bitcast %struct.point13* %0 to i8*
  %4 = bitcast %struct.point13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 104, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(%struct.point14* noalias sret(%struct.point14) align 8 %0, %struct.point14* noundef byval(%struct.point14) align 8 %1) #0 {
  %3 = bitcast %struct.point14* %0 to i8*
  %4 = bitcast %struct.point14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 112, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(%struct.point15* noalias sret(%struct.point15) align 8 %0, %struct.point15* noundef byval(%struct.point15) align 8 %1) #0 {
  %3 = bitcast %struct.point15* %0 to i8*
  %4 = bitcast %struct.point15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 120, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(%struct.point16* noalias sret(%struct.point16) align 8 %0, %struct.point16* noundef byval(%struct.point16) align 8 %1) #0 {
  %3 = bitcast %struct.point16* %0 to i8*
  %4 = bitcast %struct.point16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 128, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(%struct.point17* noalias sret(%struct.point17) align 8 %0, %struct.point17* noundef byval(%struct.point17) align 8 %1) #0 {
  %3 = bitcast %struct.point17* %0 to i8*
  %4 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 136, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(%struct.point18* noalias sret(%struct.point18) align 8 %0, %struct.point18* noundef byval(%struct.point18) align 8 %1) #0 {
  %3 = bitcast %struct.point18* %0 to i8*
  %4 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 144, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(%struct.point19* noalias sret(%struct.point19) align 8 %0, %struct.point19* noundef byval(%struct.point19) align 8 %1) #0 {
  %3 = bitcast %struct.point19* %0 to i8*
  %4 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 152, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(%struct.point20* noalias sret(%struct.point20) align 8 %0, %struct.point20* noundef byval(%struct.point20) align 8 %1) #0 {
  %3 = bitcast %struct.point20* %0 to i8*
  %4 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 160, i1 false)
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
