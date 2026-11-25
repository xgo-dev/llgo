; ModuleID = '../../wrap/struct_int8.c'
source_filename = "../../wrap/struct_int8.c"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-unknown-linux-gnu"

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

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i8 @demo1(i64 %0) #0 {
  %2 = alloca %struct.point1, align 1
  %3 = alloca %struct.point1, align 1
  %4 = getelementptr inbounds %struct.point1, %struct.point1* %3, i32 0, i32 0
  %5 = trunc i64 %0 to i8
  store i8 %5, i8* %4, align 1
  %6 = bitcast %struct.point1* %2 to i8*
  %7 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 1 %7, i64 1, i1 false)
  %8 = getelementptr inbounds %struct.point1, %struct.point1* %2, i32 0, i32 0
  %9 = load i8, i8* %8, align 1
  ret i8 %9
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i16 @demo2(i64 %0) #0 {
  %2 = alloca %struct.point2, align 1
  %3 = alloca %struct.point2, align 1
  %4 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %5 = bitcast %struct.point2* %3 to i8*
  %6 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 8 %6, i64 2, i1 false)
  %7 = bitcast %struct.point2* %2 to i8*
  %8 = bitcast %struct.point2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 1 %8, i64 2, i1 false)
  %9 = bitcast %struct.point2* %2 to i16*
  %10 = load i16, i16* %9, align 1
  ret i16 %10
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i24 @demo3(i64 %0) #0 {
  %2 = alloca %struct.point3, align 1
  %3 = alloca %struct.point3, align 1
  %4 = alloca i64, align 8
  %5 = alloca i24, align 4
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point3* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 3, i1 false)
  %8 = bitcast %struct.point3* %2 to i8*
  %9 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 3, i1 false)
  %10 = bitcast i24* %5 to i8*
  %11 = bitcast %struct.point3* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %10, i8* align 1 %11, i64 3, i1 false)
  %12 = load i24, i24* %5, align 4
  ret i24 %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo4(i64 %0) #0 {
  %2 = alloca %struct.point4, align 1
  %3 = alloca %struct.point4, align 1
  %4 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %5 = bitcast %struct.point4* %3 to i8*
  %6 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 8 %6, i64 4, i1 false)
  %7 = bitcast %struct.point4* %2 to i8*
  %8 = bitcast %struct.point4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 1 %8, i64 4, i1 false)
  %9 = bitcast %struct.point4* %2 to i32*
  %10 = load i32, i32* %9, align 1
  ret i32 %10
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i40 @demo5(i64 %0) #0 {
  %2 = alloca %struct.point5, align 1
  %3 = alloca %struct.point5, align 1
  %4 = alloca i64, align 8
  %5 = alloca i40, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point5* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 5, i1 false)
  %8 = bitcast %struct.point5* %2 to i8*
  %9 = bitcast %struct.point5* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 5, i1 false)
  %10 = bitcast i40* %5 to i8*
  %11 = bitcast %struct.point5* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 5, i1 false)
  %12 = load i40, i40* %5, align 8
  ret i40 %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i48 @demo6(i64 %0) #0 {
  %2 = alloca %struct.point6, align 1
  %3 = alloca %struct.point6, align 1
  %4 = alloca i64, align 8
  %5 = alloca i48, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point6* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point6* %2 to i8*
  %9 = bitcast %struct.point6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 6, i1 false)
  %10 = bitcast i48* %5 to i8*
  %11 = bitcast %struct.point6* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 6, i1 false)
  %12 = load i48, i48* %5, align 8
  ret i48 %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i56 @demo7(i64 %0) #0 {
  %2 = alloca %struct.point7, align 1
  %3 = alloca %struct.point7, align 1
  %4 = alloca i64, align 8
  %5 = alloca i56, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point7* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 7, i1 false)
  %8 = bitcast %struct.point7* %2 to i8*
  %9 = bitcast %struct.point7* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 7, i1 false)
  %10 = bitcast i56* %5 to i8*
  %11 = bitcast %struct.point7* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 7, i1 false)
  %12 = load i56, i56* %5, align 8
  ret i56 %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @demo8(i64 %0) #0 {
  %2 = alloca %struct.point8, align 1
  %3 = alloca %struct.point8, align 1
  %4 = bitcast %struct.point8* %3 to i64*
  store i64 %0, i64* %4, align 1
  %5 = bitcast %struct.point8* %2 to i8*
  %6 = bitcast %struct.point8* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 8, i1 false)
  %7 = bitcast %struct.point8* %2 to i64*
  %8 = load i64, i64* %7, align 1
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo9([2 x i64] %0) #0 {
  %2 = alloca %struct.point9, align 1
  %3 = alloca %struct.point9, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point9* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 9, i1 false)
  %8 = bitcast %struct.point9* %2 to i8*
  %9 = bitcast %struct.point9* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 9, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point9* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 9, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo10([2 x i64] %0) #0 {
  %2 = alloca %struct.point10, align 1
  %3 = alloca %struct.point10, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point10* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point10* %2 to i8*
  %9 = bitcast %struct.point10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point10* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo11([2 x i64] %0) #0 {
  %2 = alloca %struct.point11, align 1
  %3 = alloca %struct.point11, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point11* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 11, i1 false)
  %8 = bitcast %struct.point11* %2 to i8*
  %9 = bitcast %struct.point11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 11, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point11* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 11, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo12([2 x i64] %0) #0 {
  %2 = alloca %struct.point12, align 1
  %3 = alloca %struct.point12, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point12* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point12* %2 to i8*
  %9 = bitcast %struct.point12* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point12* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo13([2 x i64] %0) #0 {
  %2 = alloca %struct.point13, align 1
  %3 = alloca %struct.point13, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point13* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 13, i1 false)
  %8 = bitcast %struct.point13* %2 to i8*
  %9 = bitcast %struct.point13* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 13, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point13* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 13, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo14([2 x i64] %0) #0 {
  %2 = alloca %struct.point14, align 1
  %3 = alloca %struct.point14, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point14* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 14, i1 false)
  %8 = bitcast %struct.point14* %2 to i8*
  %9 = bitcast %struct.point14* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 14, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point14* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 14, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo15([2 x i64] %0) #0 {
  %2 = alloca %struct.point15, align 1
  %3 = alloca %struct.point15, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point15* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 15, i1 false)
  %8 = bitcast %struct.point15* %2 to i8*
  %9 = bitcast %struct.point15* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %8, i8* align 1 %9, i64 15, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point15* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 1 %11, i64 15, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo16([2 x i64] %0) #0 {
  %2 = alloca %struct.point16, align 1
  %3 = alloca %struct.point16, align 1
  %4 = bitcast %struct.point16* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 1
  %5 = bitcast %struct.point16* %2 to i8*
  %6 = bitcast %struct.point16* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 16, i1 false)
  %7 = bitcast %struct.point16* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 1
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo17(%struct.point17* noalias sret(%struct.point17) align 1 %0, %struct.point17* noundef %1) #0 {
  %3 = bitcast %struct.point17* %0 to i8*
  %4 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %3, i8* align 1 %4, i64 17, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo18(%struct.point18* noalias sret(%struct.point18) align 1 %0, %struct.point18* noundef %1) #0 {
  %3 = bitcast %struct.point18* %0 to i8*
  %4 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %3, i8* align 1 %4, i64 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo19(%struct.point19* noalias sret(%struct.point19) align 1 %0, %struct.point19* noundef %1) #0 {
  %3 = bitcast %struct.point19* %0 to i8*
  %4 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %3, i8* align 1 %4, i64 19, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo20(%struct.point20* noalias sret(%struct.point20) align 1 %0, %struct.point20* noundef %1) #0 {
  %3 = bitcast %struct.point20* %0 to i8*
  %4 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %3, i8* align 1 %4, i64 20, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone uwtable "frame-pointer"="non-leaf" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="generic" "target-features"="+neon,+v8a" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3, !4, !5, !6, !7, !8}
!llvm.ident = !{!9}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 8, !"branch-target-enforcement", i32 0}
!2 = !{i32 8, !"sign-return-address", i32 0}
!3 = !{i32 8, !"sign-return-address-all", i32 0}
!4 = !{i32 8, !"sign-return-address-with-bkey", i32 0}
!5 = !{i32 7, !"PIC Level", i32 2}
!6 = !{i32 7, !"PIE Level", i32 2}
!7 = !{i32 7, !"uwtable", i32 2}
!8 = !{i32 7, !"frame-pointer", i32 1}
!9 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
