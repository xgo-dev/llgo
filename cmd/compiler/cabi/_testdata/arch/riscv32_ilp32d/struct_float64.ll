; ModuleID = '../../wrap/struct_float64.c'
source_filename = "../../wrap/struct_float64.c"
target datalayout = "e-m:e-p:32:32-i64:64-n32-S128"
target triple = "riscv32-unknown-unknown-elf"

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
define dso_local double @demo1(double %0) #0 {
  %2 = alloca %struct.point1, align 8
  %3 = alloca %struct.point1, align 8
  %4 = bitcast %struct.point1* %3 to { double }*
  %5 = getelementptr inbounds { double }, { double }* %4, i32 0, i32 0
  store double %0, double* %5, align 8
  %6 = bitcast %struct.point1* %2 to i8*
  %7 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %6, i8* align 8 %7, i32 8, i1 false)
  %8 = bitcast %struct.point1* %2 to { double }*
  %9 = getelementptr inbounds { double }, { double }* %8, i32 0, i32 0
  %10 = load double, double* %9, align 8
  ret double %10
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local { double, double } @demo2(double %0, double %1) #0 {
  %3 = alloca %struct.point2, align 8
  %4 = alloca %struct.point2, align 8
  %5 = bitcast %struct.point2* %4 to { double, double }*
  %6 = getelementptr inbounds { double, double }, { double, double }* %5, i32 0, i32 0
  store double %0, double* %6, align 8
  %7 = getelementptr inbounds { double, double }, { double, double }* %5, i32 0, i32 1
  store double %1, double* %7, align 8
  %8 = bitcast %struct.point2* %3 to i8*
  %9 = bitcast %struct.point2* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %8, i8* align 8 %9, i32 16, i1 false)
  %10 = bitcast %struct.point2* %3 to { double, double }*
  %11 = getelementptr inbounds { double, double }, { double, double }* %10, i32 0, i32 0
  %12 = load double, double* %11, align 8
  %13 = getelementptr inbounds { double, double }, { double, double }* %10, i32 0, i32 1
  %14 = load double, double* %13, align 8
  %15 = insertvalue { double, double } undef, double %12, 0
  %16 = insertvalue { double, double } %15, double %14, 1
  ret { double, double } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo3(%struct.point3* noalias sret(%struct.point3) align 8 %0, %struct.point3* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point3* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point3* %0 to i8*
  %6 = bitcast %struct.point3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo4(%struct.point4* noalias sret(%struct.point4) align 8 %0, %struct.point4* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point4* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point4* %0 to i8*
  %6 = bitcast %struct.point4* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo5(%struct.point5* noalias sret(%struct.point5) align 8 %0, %struct.point5* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point5* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point5* %0 to i8*
  %6 = bitcast %struct.point5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo6(%struct.point6* noalias sret(%struct.point6) align 8 %0, %struct.point6* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point6* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point6* %0 to i8*
  %6 = bitcast %struct.point6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(%struct.point7* noalias sret(%struct.point7) align 8 %0, %struct.point7* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point7* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point7* %0 to i8*
  %6 = bitcast %struct.point7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo8(%struct.point8* noalias sret(%struct.point8) align 8 %0, %struct.point8* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point8* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point8* %0 to i8*
  %6 = bitcast %struct.point8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(%struct.point9* noalias sret(%struct.point9) align 8 %0, %struct.point9* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point9* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point9* %0 to i8*
  %6 = bitcast %struct.point9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(%struct.point10* noalias sret(%struct.point10) align 8 %0, %struct.point10* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point10* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point10* %0 to i8*
  %6 = bitcast %struct.point10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 80, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(%struct.point11* noalias sret(%struct.point11) align 8 %0, %struct.point11* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point11* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point11* %0 to i8*
  %6 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 88, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(%struct.point12* noalias sret(%struct.point12) align 8 %0, %struct.point12* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point12* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point12* %0 to i8*
  %6 = bitcast %struct.point12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 96, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(%struct.point13* noalias sret(%struct.point13) align 8 %0, %struct.point13* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point13* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point13* %0 to i8*
  %6 = bitcast %struct.point13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 104, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(%struct.point14* noalias sret(%struct.point14) align 8 %0, %struct.point14* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point14* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point14* %0 to i8*
  %6 = bitcast %struct.point14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 112, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(%struct.point15* noalias sret(%struct.point15) align 8 %0, %struct.point15* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point15* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point15* %0 to i8*
  %6 = bitcast %struct.point15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 120, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(%struct.point16* noalias sret(%struct.point16) align 8 %0, %struct.point16* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point16* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point16* %0 to i8*
  %6 = bitcast %struct.point16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 128, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(%struct.point17* noalias sret(%struct.point17) align 8 %0, %struct.point17* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point17* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point17* %0 to i8*
  %6 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 136, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(%struct.point18* noalias sret(%struct.point18) align 8 %0, %struct.point18* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point18* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point18* %0 to i8*
  %6 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 144, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(%struct.point19* noalias sret(%struct.point19) align 8 %0, %struct.point19* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point19* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point19* %0 to i8*
  %6 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 152, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(%struct.point20* noalias sret(%struct.point20) align 8 %0, %struct.point20* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point20* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point20* %0 to i8*
  %6 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %5, i8* align 8 %6, i32 160, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+a,+c,+d,+f,+m,+relax,-save-restore" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3}
!llvm.ident = !{!4}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"target-abi", !"ilp32d"}
!2 = !{i32 7, !"frame-pointer", i32 2}
!3 = !{i32 1, !"SmallDataLimit", i32 8}
!4 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
