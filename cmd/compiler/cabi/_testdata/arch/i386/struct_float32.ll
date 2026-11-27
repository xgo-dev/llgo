; ModuleID = '../../wrap/struct_float32.c'
source_filename = "../../wrap/struct_float32.c"
target datalayout = "e-m:e-p:32:32-p270:32:32-p271:32:32-p272:64:64-f64:32:64-f80:32-n8:16:32-S128"
target triple = "i386-unknown-linux-gnu"

%struct.point1 = type { float }
%struct.point2 = type { float, float }
%struct.point3 = type { float, float, float }
%struct.point4 = type { float, float, float, float }
%struct.point5 = type { float, float, float, float, float }
%struct.point6 = type { float, float, float, float, float, float }
%struct.point7 = type { float, float, float, float, float, float, float }
%struct.point8 = type { float, float, float, float, float, float, float, float }
%struct.point9 = type { float, float, float, float, float, float, float, float, float }
%struct.point10 = type { float, float, float, float, float, float, float, float, float, float }
%struct.point11 = type { float, float, float, float, float, float, float, float, float, float, float }
%struct.point12 = type { float, float, float, float, float, float, float, float, float, float, float, float }
%struct.point13 = type { float, float, float, float, float, float, float, float, float, float, float, float, float }
%struct.point14 = type { float, float, float, float, float, float, float, float, float, float, float, float, float, float }
%struct.point15 = type { float, float, float, float, float, float, float, float, float, float, float, float, float, float, float }
%struct.point16 = type { float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float }
%struct.point17 = type { float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float }
%struct.point18 = type { float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float }
%struct.point19 = type { float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float }
%struct.point20 = type { float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float, float }

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo1(%struct.point1* noalias sret(%struct.point1) align 4 %0, float %1) #0 {
  %3 = alloca i8*, align 4
  %4 = alloca %struct.point1, align 4
  %5 = bitcast %struct.point1* %0 to i8*
  store i8* %5, i8** %3, align 4
  %6 = getelementptr inbounds %struct.point1, %struct.point1* %4, i32 0, i32 0
  store float %1, float* %6, align 4
  %7 = bitcast %struct.point1* %0 to i8*
  %8 = bitcast %struct.point1* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %7, i8* align 4 %8, i32 4, i1 false)
  ret void
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo2(%struct.point2* noalias sret(%struct.point2) align 4 %0, float %1, float %2) #0 {
  %4 = alloca i8*, align 4
  %5 = alloca %struct.point2, align 4
  %6 = bitcast %struct.point2* %0 to i8*
  store i8* %6, i8** %4, align 4
  %7 = getelementptr inbounds %struct.point2, %struct.point2* %5, i32 0, i32 0
  store float %1, float* %7, align 4
  %8 = getelementptr inbounds %struct.point2, %struct.point2* %5, i32 0, i32 1
  store float %2, float* %8, align 4
  %9 = bitcast %struct.point2* %0 to i8*
  %10 = bitcast %struct.point2* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %9, i8* align 4 %10, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo3(%struct.point3* noalias sret(%struct.point3) align 4 %0, float %1, float %2, float %3) #0 {
  %5 = alloca i8*, align 4
  %6 = alloca %struct.point3, align 4
  %7 = bitcast %struct.point3* %0 to i8*
  store i8* %7, i8** %5, align 4
  %8 = getelementptr inbounds %struct.point3, %struct.point3* %6, i32 0, i32 0
  store float %1, float* %8, align 4
  %9 = getelementptr inbounds %struct.point3, %struct.point3* %6, i32 0, i32 1
  store float %2, float* %9, align 4
  %10 = getelementptr inbounds %struct.point3, %struct.point3* %6, i32 0, i32 2
  store float %3, float* %10, align 4
  %11 = bitcast %struct.point3* %0 to i8*
  %12 = bitcast %struct.point3* %6 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %11, i8* align 4 %12, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo4(%struct.point4* noalias sret(%struct.point4) align 4 %0, float %1, float %2, float %3, float %4) #0 {
  %6 = alloca i8*, align 4
  %7 = alloca %struct.point4, align 4
  %8 = bitcast %struct.point4* %0 to i8*
  store i8* %8, i8** %6, align 4
  %9 = getelementptr inbounds %struct.point4, %struct.point4* %7, i32 0, i32 0
  store float %1, float* %9, align 4
  %10 = getelementptr inbounds %struct.point4, %struct.point4* %7, i32 0, i32 1
  store float %2, float* %10, align 4
  %11 = getelementptr inbounds %struct.point4, %struct.point4* %7, i32 0, i32 2
  store float %3, float* %11, align 4
  %12 = getelementptr inbounds %struct.point4, %struct.point4* %7, i32 0, i32 3
  store float %4, float* %12, align 4
  %13 = bitcast %struct.point4* %0 to i8*
  %14 = bitcast %struct.point4* %7 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %13, i8* align 4 %14, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo5(%struct.point5* noalias sret(%struct.point5) align 4 %0, %struct.point5* noundef byval(%struct.point5) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point5* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point5* %0 to i8*
  %6 = bitcast %struct.point5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo6(%struct.point6* noalias sret(%struct.point6) align 4 %0, %struct.point6* noundef byval(%struct.point6) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point6* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point6* %0 to i8*
  %6 = bitcast %struct.point6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo7(%struct.point7* noalias sret(%struct.point7) align 4 %0, %struct.point7* noundef byval(%struct.point7) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point7* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point7* %0 to i8*
  %6 = bitcast %struct.point7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo8(%struct.point8* noalias sret(%struct.point8) align 4 %0, %struct.point8* noundef byval(%struct.point8) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point8* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point8* %0 to i8*
  %6 = bitcast %struct.point8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo9(%struct.point9* noalias sret(%struct.point9) align 4 %0, %struct.point9* noundef byval(%struct.point9) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point9* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point9* %0 to i8*
  %6 = bitcast %struct.point9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo10(%struct.point10* noalias sret(%struct.point10) align 4 %0, %struct.point10* noundef byval(%struct.point10) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point10* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point10* %0 to i8*
  %6 = bitcast %struct.point10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo11(%struct.point11* noalias sret(%struct.point11) align 4 %0, %struct.point11* noundef byval(%struct.point11) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point11* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point11* %0 to i8*
  %6 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 44, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo12(%struct.point12* noalias sret(%struct.point12) align 4 %0, %struct.point12* noundef byval(%struct.point12) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point12* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point12* %0 to i8*
  %6 = bitcast %struct.point12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo13(%struct.point13* noalias sret(%struct.point13) align 4 %0, %struct.point13* noundef byval(%struct.point13) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point13* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point13* %0 to i8*
  %6 = bitcast %struct.point13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 52, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo14(%struct.point14* noalias sret(%struct.point14) align 4 %0, %struct.point14* noundef byval(%struct.point14) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point14* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point14* %0 to i8*
  %6 = bitcast %struct.point14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo15(%struct.point15* noalias sret(%struct.point15) align 4 %0, %struct.point15* noundef byval(%struct.point15) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point15* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point15* %0 to i8*
  %6 = bitcast %struct.point15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 60, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo16(%struct.point16* noalias sret(%struct.point16) align 4 %0, %struct.point16* noundef byval(%struct.point16) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point16* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point16* %0 to i8*
  %6 = bitcast %struct.point16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo17(%struct.point17* noalias sret(%struct.point17) align 4 %0, %struct.point17* noundef byval(%struct.point17) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point17* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point17* %0 to i8*
  %6 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 68, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo18(%struct.point18* noalias sret(%struct.point18) align 4 %0, %struct.point18* noundef byval(%struct.point18) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point18* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point18* %0 to i8*
  %6 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo19(%struct.point19* noalias sret(%struct.point19) align 4 %0, %struct.point19* noundef byval(%struct.point19) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point19* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point19* %0 to i8*
  %6 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 76, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo20(%struct.point20* noalias sret(%struct.point20) align 4 %0, %struct.point20* noundef byval(%struct.point20) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point20* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point20* %0 to i8*
  %6 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 80, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone uwtable "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="pentium4" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3, !4, !5}
!llvm.ident = !{!6}

!0 = !{i32 1, !"NumRegisterParameters", i32 0}
!1 = !{i32 1, !"wchar_size", i32 4}
!2 = !{i32 7, !"PIC Level", i32 2}
!3 = !{i32 7, !"PIE Level", i32 2}
!4 = !{i32 7, !"uwtable", i32 2}
!5 = !{i32 7, !"frame-pointer", i32 2}
!6 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
