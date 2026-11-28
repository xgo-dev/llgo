; ModuleID = '../../wrap/struct_int16.c'
source_filename = "../../wrap/struct_int16.c"
target datalayout = "e-m:e-p:32:32-i64:64-n32-S128"
target triple = "riscv32-unknown-unknown-elf"

%struct.point1 = type { i16 }
%struct.point2 = type { i16, i16 }
%struct.point3 = type { i16, i16, i16 }
%struct.point4 = type { i16, i16, i16, i16 }
%struct.point5 = type { i16, i16, i16, i16, i16 }
%struct.point6 = type { i16, i16, i16, i16, i16, i16 }
%struct.point7 = type { i16, i16, i16, i16, i16, i16, i16 }
%struct.point8 = type { i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point9 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point10 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point11 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point12 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point13 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point14 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point15 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point16 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point17 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point18 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point19 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point20 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo1(i32 %0) #0 {
  %2 = alloca %struct.point1, align 2
  %3 = alloca %struct.point1, align 2
  %4 = getelementptr inbounds %struct.point1, %struct.point1* %3, i32 0, i32 0
  %5 = trunc i32 %0 to i16
  store i16 %5, i16* %4, align 2
  %6 = bitcast %struct.point1* %2 to i8*
  %7 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %6, i8* align 2 %7, i32 2, i1 false)
  %8 = getelementptr inbounds %struct.point1, %struct.point1* %2, i32 0, i32 0
  %9 = load i16, i16* %8, align 2
  %10 = zext i16 %9 to i32
  ret i32 %10
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo2(i32 %0) #0 {
  %2 = alloca %struct.point2, align 2
  %3 = alloca %struct.point2, align 2
  %4 = bitcast %struct.point2* %3 to i32*
  store i32 %0, i32* %4, align 2
  %5 = bitcast %struct.point2* %2 to i8*
  %6 = bitcast %struct.point2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 4, i1 false)
  %7 = bitcast %struct.point2* %2 to i32*
  %8 = load i32, i32* %7, align 2
  ret i32 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo3([2 x i32] %0) #0 {
  %2 = alloca %struct.point3, align 2
  %3 = alloca %struct.point3, align 2
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, [2 x i32]* %4, align 4
  %6 = bitcast %struct.point3* %3 to i8*
  %7 = bitcast [2 x i32]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %6, i8* align 4 %7, i32 6, i1 false)
  %8 = bitcast %struct.point3* %2 to i8*
  %9 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %8, i8* align 2 %9, i32 6, i1 false)
  %10 = bitcast [2 x i32]* %5 to i8*
  %11 = bitcast %struct.point3* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %10, i8* align 2 %11, i32 6, i1 false)
  %12 = load [2 x i32], [2 x i32]* %5, align 4
  ret [2 x i32] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo4([2 x i32] %0) #0 {
  %2 = alloca %struct.point4, align 2
  %3 = alloca %struct.point4, align 2
  %4 = bitcast %struct.point4* %3 to [2 x i32]*
  store [2 x i32] %0, [2 x i32]* %4, align 2
  %5 = bitcast %struct.point4* %2 to i8*
  %6 = bitcast %struct.point4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 8, i1 false)
  %7 = bitcast %struct.point4* %2 to [2 x i32]*
  %8 = load [2 x i32], [2 x i32]* %7, align 2
  ret [2 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo5(%struct.point5* noalias sret(%struct.point5) align 2 %0, %struct.point5* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point5* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point5* %0 to i8*
  %6 = bitcast %struct.point5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo6(%struct.point6* noalias sret(%struct.point6) align 2 %0, %struct.point6* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point6* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point6* %0 to i8*
  %6 = bitcast %struct.point6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(%struct.point7* noalias sret(%struct.point7) align 2 %0, %struct.point7* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point7* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point7* %0 to i8*
  %6 = bitcast %struct.point7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 14, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo8(%struct.point8* noalias sret(%struct.point8) align 2 %0, %struct.point8* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point8* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point8* %0 to i8*
  %6 = bitcast %struct.point8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(%struct.point9* noalias sret(%struct.point9) align 2 %0, %struct.point9* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point9* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point9* %0 to i8*
  %6 = bitcast %struct.point9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(%struct.point10* noalias sret(%struct.point10) align 2 %0, %struct.point10* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point10* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point10* %0 to i8*
  %6 = bitcast %struct.point10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(%struct.point11* noalias sret(%struct.point11) align 2 %0, %struct.point11* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point11* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point11* %0 to i8*
  %6 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 22, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(%struct.point12* noalias sret(%struct.point12) align 2 %0, %struct.point12* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point12* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point12* %0 to i8*
  %6 = bitcast %struct.point12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(%struct.point13* noalias sret(%struct.point13) align 2 %0, %struct.point13* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point13* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point13* %0 to i8*
  %6 = bitcast %struct.point13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 26, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(%struct.point14* noalias sret(%struct.point14) align 2 %0, %struct.point14* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point14* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point14* %0 to i8*
  %6 = bitcast %struct.point14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(%struct.point15* noalias sret(%struct.point15) align 2 %0, %struct.point15* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point15* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point15* %0 to i8*
  %6 = bitcast %struct.point15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 30, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(%struct.point16* noalias sret(%struct.point16) align 2 %0, %struct.point16* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point16* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point16* %0 to i8*
  %6 = bitcast %struct.point16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(%struct.point17* noalias sret(%struct.point17) align 2 %0, %struct.point17* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point17* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point17* %0 to i8*
  %6 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 34, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(%struct.point18* noalias sret(%struct.point18) align 2 %0, %struct.point18* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point18* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point18* %0 to i8*
  %6 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(%struct.point19* noalias sret(%struct.point19) align 2 %0, %struct.point19* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point19* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point19* %0 to i8*
  %6 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 38, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(%struct.point20* noalias sret(%struct.point20) align 2 %0, %struct.point20* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point20* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point20* %0 to i8*
  %6 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %5, i8* align 2 %6, i32 40, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+a,+c,+d,+f,+m,+relax,-save-restore" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3}
!llvm.ident = !{!4}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"target-abi", !"ilp32"}
!2 = !{i32 7, !"frame-pointer", i32 2}
!3 = !{i32 1, !"SmallDataLimit", i32 8}
!4 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
