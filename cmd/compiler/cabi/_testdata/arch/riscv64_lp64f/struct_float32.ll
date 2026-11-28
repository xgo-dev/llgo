; ModuleID = '../../wrap/struct_float32.c'
source_filename = "../../wrap/struct_float32.c"
target datalayout = "e-m:e-p:64:64-i64:64-i128:128-n64-S128"
target triple = "riscv64-unknown-unknown-elf"

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

; Function Attrs: noinline nounwind optnone
define dso_local float @demo1(float %0) #0 {
  %2 = alloca %struct.point1, align 4
  %3 = alloca %struct.point1, align 4
  %4 = bitcast %struct.point1* %3 to { float }*
  %5 = getelementptr inbounds { float }, { float }* %4, i32 0, i32 0
  store float %0, float* %5, align 4
  %6 = bitcast %struct.point1* %2 to i8*
  %7 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 4 %7, i64 4, i1 false)
  %8 = bitcast %struct.point1* %2 to { float }*
  %9 = getelementptr inbounds { float }, { float }* %8, i32 0, i32 0
  %10 = load float, float* %9, align 4
  ret float %10
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local { float, float } @demo2(float %0, float %1) #0 {
  %3 = alloca %struct.point2, align 4
  %4 = alloca %struct.point2, align 4
  %5 = bitcast %struct.point2* %4 to { float, float }*
  %6 = getelementptr inbounds { float, float }, { float, float }* %5, i32 0, i32 0
  store float %0, float* %6, align 4
  %7 = getelementptr inbounds { float, float }, { float, float }* %5, i32 0, i32 1
  store float %1, float* %7, align 4
  %8 = bitcast %struct.point2* %3 to i8*
  %9 = bitcast %struct.point2* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 8, i1 false)
  %10 = bitcast %struct.point2* %3 to { float, float }*
  %11 = getelementptr inbounds { float, float }, { float, float }* %10, i32 0, i32 0
  %12 = load float, float* %11, align 4
  %13 = getelementptr inbounds { float, float }, { float, float }* %10, i32 0, i32 1
  %14 = load float, float* %13, align 4
  %15 = insertvalue { float, float } undef, float %12, 0
  %16 = insertvalue { float, float } %15, float %14, 1
  ret { float, float } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo3([2 x i64] %0) #0 {
  %2 = alloca %struct.point3, align 4
  %3 = alloca %struct.point3, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point3* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point3* %2 to i8*
  %9 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point3* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo4([2 x i64] %0) #0 {
  %2 = alloca %struct.point4, align 4
  %3 = alloca %struct.point4, align 4
  %4 = bitcast %struct.point4* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point4* %2 to i8*
  %6 = bitcast %struct.point4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point4* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo5(%struct.point5* noalias sret(%struct.point5) align 4 %0, %struct.point5* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point5* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point5* %0 to i8*
  %6 = bitcast %struct.point5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo6(%struct.point6* noalias sret(%struct.point6) align 4 %0, %struct.point6* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point6* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point6* %0 to i8*
  %6 = bitcast %struct.point6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(%struct.point7* noalias sret(%struct.point7) align 4 %0, %struct.point7* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point7* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point7* %0 to i8*
  %6 = bitcast %struct.point7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo8(%struct.point8* noalias sret(%struct.point8) align 4 %0, %struct.point8* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point8* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point8* %0 to i8*
  %6 = bitcast %struct.point8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(%struct.point9* noalias sret(%struct.point9) align 4 %0, %struct.point9* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point9* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point9* %0 to i8*
  %6 = bitcast %struct.point9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(%struct.point10* noalias sret(%struct.point10) align 4 %0, %struct.point10* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point10* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point10* %0 to i8*
  %6 = bitcast %struct.point10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(%struct.point11* noalias sret(%struct.point11) align 4 %0, %struct.point11* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point11* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point11* %0 to i8*
  %6 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 44, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(%struct.point12* noalias sret(%struct.point12) align 4 %0, %struct.point12* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point12* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point12* %0 to i8*
  %6 = bitcast %struct.point12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(%struct.point13* noalias sret(%struct.point13) align 4 %0, %struct.point13* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point13* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point13* %0 to i8*
  %6 = bitcast %struct.point13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 52, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(%struct.point14* noalias sret(%struct.point14) align 4 %0, %struct.point14* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point14* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point14* %0 to i8*
  %6 = bitcast %struct.point14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(%struct.point15* noalias sret(%struct.point15) align 4 %0, %struct.point15* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point15* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point15* %0 to i8*
  %6 = bitcast %struct.point15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 60, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(%struct.point16* noalias sret(%struct.point16) align 4 %0, %struct.point16* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point16* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point16* %0 to i8*
  %6 = bitcast %struct.point16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(%struct.point17* noalias sret(%struct.point17) align 4 %0, %struct.point17* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point17* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point17* %0 to i8*
  %6 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 68, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(%struct.point18* noalias sret(%struct.point18) align 4 %0, %struct.point18* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point18* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point18* %0 to i8*
  %6 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(%struct.point19* noalias sret(%struct.point19) align 4 %0, %struct.point19* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point19* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point19* %0 to i8*
  %6 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 76, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(%struct.point20* noalias sret(%struct.point20) align 4 %0, %struct.point20* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point20* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point20* %0 to i8*
  %6 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 80, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+64bit,+a,+c,+d,+f,+m,+relax,-save-restore" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3}
!llvm.ident = !{!4}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"target-abi", !"lp64f"}
!2 = !{i32 7, !"frame-pointer", i32 2}
!3 = !{i32 1, !"SmallDataLimit", i32 8}
!4 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
