; ModuleID = '../../wrap/demo.c'
source_filename = "../../wrap/demo.c"
target datalayout = "e-m:e-p:64:64-i64:64-i128:128-n64-S128"
target triple = "riscv64-unknown-unknown-elf"

%struct.point = type { i32, i32 }
%struct.point1 = type { i32, i32, i32 }
%struct.point64 = type { i64, i64 }
%struct.struct32 = type { i32 }
%struct.point2 = type { i8, i32, i32 }
%struct.point3 = type { i8, i8, i8 }
%struct.point4 = type { i8, i8, i8, i32 }
%struct.point5 = type { i8, i8, i8, i8, i8 }
%struct.point6 = type { i8, i8, i8, i8, i8, i32 }
%struct.point7 = type { i8, i8, i8, i8, i8, i32, i8 }
%struct.data1 = type { i8, i64 }
%struct.data2 = type { i32, i64 }
%struct.data3 = type { i64, i8 }
%struct.fdata1 = type { float }
%struct.ddata1 = type { double }
%struct.ddata2 = type { double, double }
%struct.ddata3 = type { double, double, double }
%struct.fdata2i = type { float, i32 }
%struct.fdata2 = type { float, float }
%struct.fdata3 = type { float, float, float }
%struct.fdata4 = type { float, float, float, float }
%struct.fdata5 = type { float, float, float, float, float }
%struct.fdata2id = type { i8, i8, double }
%struct.fdata7if = type { [7 x i8], float }
%struct.fdata4if = type { float, i8, float, float }
%struct.array = type { [8 x i32] }

@.str = private unnamed_addr constant [20 x i8] c"point64: %lld %lld\0A\00", align 1
@.str.1 = private unnamed_addr constant [14 x i8] c"struct32: %d\0A\00", align 1
@.str.2 = private unnamed_addr constant [14 x i8] c"point: %d %d\0A\00", align 1
@.str.3 = private unnamed_addr constant [18 x i8] c"point1: %d %d %d\0A\00", align 1
@.str.4 = private unnamed_addr constant [18 x i8] c"point2: %d %d %d\0A\00", align 1
@.str.5 = private unnamed_addr constant [18 x i8] c"point3: %d %d %d\0A\00", align 1
@.str.6 = private unnamed_addr constant [21 x i8] c"point4: %d %d %d %d\0A\00", align 1
@.str.7 = private unnamed_addr constant [24 x i8] c"point5: %d %d %d %d %d\0A\00", align 1
@.str.8 = private unnamed_addr constant [27 x i8] c"point6: %d %d %d %d %d %d\0A\00", align 1
@.str.9 = private unnamed_addr constant [30 x i8] c"point7: %d %d %d %d %d %d %d\0A\00", align 1
@.str.10 = private unnamed_addr constant [16 x i8] c"data1: %d %lld\0A\00", align 1
@.str.11 = private unnamed_addr constant [16 x i8] c"data2: %d %lld\0A\00", align 1
@.str.12 = private unnamed_addr constant [16 x i8] c"data3: %lld %d\0A\00", align 1
@.str.13 = private unnamed_addr constant [9 x i8] c"ff1: %f\0A\00", align 1
@.str.14 = private unnamed_addr constant [9 x i8] c"dd1: %f\0A\00", align 1
@.str.15 = private unnamed_addr constant [12 x i8] c"dd2: %f %f\0A\00", align 1
@.str.16 = private unnamed_addr constant [15 x i8] c"dd3: %f %f %f\0A\00", align 1
@.str.17 = private unnamed_addr constant [13 x i8] c"ff2i: %f %d\0A\00", align 1
@.str.18 = private unnamed_addr constant [12 x i8] c"ff2: %f %f\0A\00", align 1
@.str.19 = private unnamed_addr constant [15 x i8] c"ff3: %f %f %f\0A\00", align 1
@.str.20 = private unnamed_addr constant [18 x i8] c"ff4: %f %f %f %f\0A\00", align 1
@.str.21 = private unnamed_addr constant [21 x i8] c"ff5: %f %f %f %f %f\0A\00", align 1
@.str.22 = private unnamed_addr constant [15 x i8] c"ff6: %d %d %f\0A\00", align 1
@.str.23 = private unnamed_addr constant [17 x i8] c"ff7if: %d %d %f\0A\00", align 1
@.str.24 = private unnamed_addr constant [20 x i8] c"ff4if: %f %d %f %f\0A\00", align 1
@.str.25 = private unnamed_addr constant [16 x i8] c"demo: %d %d %d\0A\00", align 1
@__const.callback.pt = private unnamed_addr constant %struct.point { i32 1, i32 2 }, align 4
@__const.callback.pt1 = private unnamed_addr constant %struct.point1 { i32 1, i32 2, i32 3 }, align 4
@.str.26 = private unnamed_addr constant [27 x i8] c"callback1 array: %d %d %d\0A\00", align 1
@__const.callback1.pt = private unnamed_addr constant %struct.point { i32 1, i32 2 }, align 4
@__const.callback1.pt1 = private unnamed_addr constant %struct.point1 { i32 1, i32 2, i32 3 }, align 4
@.str.27 = private unnamed_addr constant [22 x i8] c"callback1 ret: %d,%d\0A\00", align 1
@.str.28 = private unnamed_addr constant [28 x i8] c"mycallback array: %d %d %d\0A\00", align 1
@.str.29 = private unnamed_addr constant [22 x i8] c"mycallback pt: %d %d\0A\00", align 1
@.str.30 = private unnamed_addr constant [26 x i8] c"mycallback pt1: %d %d %d\0A\00", align 1

; Function Attrs: noinline nounwind optnone
define dso_local signext i32 @demo32(i32 noundef signext %0) #0 {
  %2 = alloca i32, align 4
  store i32 %0, i32* %2, align 4
  %3 = load i32, i32* %2, align 4
  %4 = add nsw i32 %3, 100
  ret i32 %4
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo64(i64 noundef %0) #0 {
  %2 = alloca i64, align 8
  store i64 %0, i64* %2, align 8
  %3 = load i64, i64* %2, align 8
  %4 = add nsw i64 %3, 100
  ret i64 %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @pt64([2 x i64] %0) #0 {
  %2 = alloca %struct.point64, align 8
  %3 = alloca %struct.point64, align 8
  %4 = bitcast %struct.point64* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = getelementptr inbounds %struct.point64, %struct.point64* %3, i32 0, i32 0
  %6 = load i64, i64* %5, align 8
  %7 = getelementptr inbounds %struct.point64, %struct.point64* %3, i32 0, i32 1
  %8 = load i64, i64* %7, align 8
  %9 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([20 x i8], [20 x i8]* @.str, i64 0, i64 0), i64 noundef %6, i64 noundef %8)
  %10 = bitcast %struct.point64* %2 to i8*
  %11 = bitcast %struct.point64* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 8 %11, i64 16, i1 false)
  %12 = bitcast %struct.point64* %2 to [2 x i64]*
  %13 = load [2 x i64], [2 x i64]* %12, align 8
  ret [2 x i64] %13
}

declare dso_local signext i32 @printf(i8* noundef, ...) #1

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #2

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo32s(i64 %0) #0 {
  %2 = alloca %struct.struct32, align 4
  %3 = alloca %struct.struct32, align 4
  %4 = getelementptr inbounds %struct.struct32, %struct.struct32* %3, i32 0, i32 0
  %5 = trunc i64 %0 to i32
  store i32 %5, i32* %4, align 4
  %6 = getelementptr inbounds %struct.struct32, %struct.struct32* %3, i32 0, i32 0
  %7 = load i32, i32* %6, align 4
  %8 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([14 x i8], [14 x i8]* @.str.1, i64 0, i64 0), i32 noundef signext %7)
  %9 = getelementptr inbounds %struct.struct32, %struct.struct32* %2, i32 0, i32 0
  %10 = getelementptr inbounds %struct.struct32, %struct.struct32* %3, i32 0, i32 0
  %11 = load i32, i32* %10, align 4
  %12 = add nsw i32 %11, 100
  store i32 %12, i32* %9, align 4
  %13 = getelementptr inbounds %struct.struct32, %struct.struct32* %2, i32 0, i32 0
  %14 = load i32, i32* %13, align 4
  %15 = zext i32 %14 to i64
  ret i64 %15
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @pt(i64 %0) #0 {
  %2 = alloca %struct.point, align 4
  %3 = alloca %struct.point, align 4
  %4 = bitcast %struct.point* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = getelementptr inbounds %struct.point, %struct.point* %3, i32 0, i32 0
  %6 = load i32, i32* %5, align 4
  %7 = getelementptr inbounds %struct.point, %struct.point* %3, i32 0, i32 1
  %8 = load i32, i32* %7, align 4
  %9 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([14 x i8], [14 x i8]* @.str.2, i64 0, i64 0), i32 noundef signext %6, i32 noundef signext %8)
  %10 = bitcast %struct.point* %2 to i8*
  %11 = bitcast %struct.point* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %10, i8* align 4 %11, i64 8, i1 false)
  %12 = bitcast %struct.point* %2 to i64*
  %13 = load i64, i64* %12, align 4
  ret i64 %13
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @pt1([2 x i64] %0) #0 {
  %2 = alloca %struct.point1, align 4
  %3 = alloca %struct.point1, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point1* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = getelementptr inbounds %struct.point1, %struct.point1* %3, i32 0, i32 0
  %9 = load i32, i32* %8, align 4
  %10 = getelementptr inbounds %struct.point1, %struct.point1* %3, i32 0, i32 1
  %11 = load i32, i32* %10, align 4
  %12 = getelementptr inbounds %struct.point1, %struct.point1* %3, i32 0, i32 2
  %13 = load i32, i32* %12, align 4
  %14 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.3, i64 0, i64 0), i32 noundef signext %9, i32 noundef signext %11, i32 noundef signext %13)
  %15 = bitcast %struct.point1* %2 to i8*
  %16 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %15, i8* align 4 %16, i64 12, i1 false)
  %17 = bitcast [2 x i64]* %5 to i8*
  %18 = bitcast %struct.point1* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %17, i8* align 4 %18, i64 12, i1 false)
  %19 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %19
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @pt2([2 x i64] %0) #0 {
  %2 = alloca %struct.point2, align 4
  %3 = alloca %struct.point2, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point2* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = getelementptr inbounds %struct.point2, %struct.point2* %3, i32 0, i32 0
  %9 = load i8, i8* %8, align 4
  %10 = zext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point2, %struct.point2* %3, i32 0, i32 1
  %12 = load i32, i32* %11, align 4
  %13 = getelementptr inbounds %struct.point2, %struct.point2* %3, i32 0, i32 2
  %14 = load i32, i32* %13, align 4
  %15 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.4, i64 0, i64 0), i32 noundef signext %10, i32 noundef signext %12, i32 noundef signext %14)
  %16 = bitcast %struct.point2* %2 to i8*
  %17 = bitcast %struct.point2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %16, i8* align 4 %17, i64 12, i1 false)
  %18 = bitcast [2 x i64]* %5 to i8*
  %19 = bitcast %struct.point2* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %18, i8* align 4 %19, i64 12, i1 false)
  %20 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %20
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @pt3(i64 %0) #0 {
  %2 = alloca %struct.point3, align 1
  %3 = alloca %struct.point3, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point3* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 3, i1 false)
  %8 = getelementptr inbounds %struct.point3, %struct.point3* %3, i32 0, i32 0
  %9 = load i8, i8* %8, align 1
  %10 = zext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point3, %struct.point3* %3, i32 0, i32 1
  %12 = load i8, i8* %11, align 1
  %13 = zext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point3, %struct.point3* %3, i32 0, i32 2
  %15 = load i8, i8* %14, align 1
  %16 = zext i8 %15 to i32
  %17 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.5, i64 0, i64 0), i32 noundef signext %10, i32 noundef signext %13, i32 noundef signext %16)
  %18 = bitcast %struct.point3* %2 to i8*
  %19 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %18, i8* align 1 %19, i64 3, i1 false)
  %20 = bitcast i64* %5 to i8*
  %21 = bitcast %struct.point3* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %20, i8* align 1 %21, i64 3, i1 false)
  %22 = load i64, i64* %5, align 8
  ret i64 %22
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @pt4(i64 %0) #0 {
  %2 = alloca %struct.point4, align 4
  %3 = alloca %struct.point4, align 4
  %4 = bitcast %struct.point4* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = getelementptr inbounds %struct.point4, %struct.point4* %3, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = zext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point4, %struct.point4* %3, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = zext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point4, %struct.point4* %3, i32 0, i32 2
  %12 = load i8, i8* %11, align 2
  %13 = zext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point4, %struct.point4* %3, i32 0, i32 3
  %15 = load i32, i32* %14, align 4
  %16 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([21 x i8], [21 x i8]* @.str.6, i64 0, i64 0), i32 noundef signext %7, i32 noundef signext %10, i32 noundef signext %13, i32 noundef signext %15)
  %17 = bitcast %struct.point4* %2 to i8*
  %18 = bitcast %struct.point4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %17, i8* align 4 %18, i64 8, i1 false)
  %19 = bitcast %struct.point4* %2 to i64*
  %20 = load i64, i64* %19, align 4
  ret i64 %20
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @pt5(i64 %0) #0 {
  %2 = alloca %struct.point5, align 1
  %3 = alloca %struct.point5, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point5* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 5, i1 false)
  %8 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 0
  %9 = load i8, i8* %8, align 1
  %10 = zext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 1
  %12 = load i8, i8* %11, align 1
  %13 = zext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 2
  %15 = load i8, i8* %14, align 1
  %16 = zext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 3
  %18 = load i8, i8* %17, align 1
  %19 = zext i8 %18 to i32
  %20 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 4
  %21 = load i8, i8* %20, align 1
  %22 = zext i8 %21 to i32
  %23 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([24 x i8], [24 x i8]* @.str.7, i64 0, i64 0), i32 noundef signext %10, i32 noundef signext %13, i32 noundef signext %16, i32 noundef signext %19, i32 noundef signext %22)
  %24 = bitcast %struct.point5* %2 to i8*
  %25 = bitcast %struct.point5* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %24, i8* align 1 %25, i64 5, i1 false)
  %26 = bitcast i64* %5 to i8*
  %27 = bitcast %struct.point5* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %26, i8* align 1 %27, i64 5, i1 false)
  %28 = load i64, i64* %5, align 8
  ret i64 %28
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @pt6([2 x i64] %0) #0 {
  %2 = alloca %struct.point6, align 4
  %3 = alloca %struct.point6, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point6* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = getelementptr inbounds %struct.point6, %struct.point6* %3, i32 0, i32 0
  %9 = load i8, i8* %8, align 4
  %10 = zext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point6, %struct.point6* %3, i32 0, i32 1
  %12 = load i8, i8* %11, align 1
  %13 = zext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point6, %struct.point6* %3, i32 0, i32 2
  %15 = load i8, i8* %14, align 2
  %16 = zext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point6, %struct.point6* %3, i32 0, i32 3
  %18 = load i8, i8* %17, align 1
  %19 = zext i8 %18 to i32
  %20 = getelementptr inbounds %struct.point6, %struct.point6* %3, i32 0, i32 4
  %21 = load i8, i8* %20, align 4
  %22 = zext i8 %21 to i32
  %23 = getelementptr inbounds %struct.point6, %struct.point6* %3, i32 0, i32 5
  %24 = load i32, i32* %23, align 4
  %25 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([27 x i8], [27 x i8]* @.str.8, i64 0, i64 0), i32 noundef signext %10, i32 noundef signext %13, i32 noundef signext %16, i32 noundef signext %19, i32 noundef signext %22, i32 noundef signext %24)
  %26 = bitcast %struct.point6* %2 to i8*
  %27 = bitcast %struct.point6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %26, i8* align 4 %27, i64 12, i1 false)
  %28 = bitcast [2 x i64]* %5 to i8*
  %29 = bitcast %struct.point6* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %28, i8* align 4 %29, i64 12, i1 false)
  %30 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %30
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @pt7([2 x i64] %0) #0 {
  %2 = alloca %struct.point7, align 4
  %3 = alloca %struct.point7, align 4
  %4 = bitcast %struct.point7* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = getelementptr inbounds %struct.point7, %struct.point7* %3, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = zext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point7, %struct.point7* %3, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = zext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point7, %struct.point7* %3, i32 0, i32 2
  %12 = load i8, i8* %11, align 2
  %13 = zext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point7, %struct.point7* %3, i32 0, i32 3
  %15 = load i8, i8* %14, align 1
  %16 = zext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point7, %struct.point7* %3, i32 0, i32 4
  %18 = load i8, i8* %17, align 4
  %19 = zext i8 %18 to i32
  %20 = getelementptr inbounds %struct.point7, %struct.point7* %3, i32 0, i32 5
  %21 = load i32, i32* %20, align 4
  %22 = getelementptr inbounds %struct.point7, %struct.point7* %3, i32 0, i32 6
  %23 = load i8, i8* %22, align 4
  %24 = zext i8 %23 to i32
  %25 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([30 x i8], [30 x i8]* @.str.9, i64 0, i64 0), i32 noundef signext %7, i32 noundef signext %10, i32 noundef signext %13, i32 noundef signext %16, i32 noundef signext %19, i32 noundef signext %21, i32 noundef signext %24)
  %26 = bitcast %struct.point7* %2 to i8*
  %27 = bitcast %struct.point7* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %26, i8* align 4 %27, i64 16, i1 false)
  %28 = bitcast %struct.point7* %2 to [2 x i64]*
  %29 = load [2 x i64], [2 x i64]* %28, align 4
  ret [2 x i64] %29
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @fn1([2 x i64] %0) #0 {
  %2 = alloca %struct.data1, align 8
  %3 = alloca %struct.data1, align 8
  %4 = bitcast %struct.data1* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = getelementptr inbounds %struct.data1, %struct.data1* %3, i32 0, i32 0
  %6 = load i8, i8* %5, align 8
  %7 = zext i8 %6 to i32
  %8 = getelementptr inbounds %struct.data1, %struct.data1* %3, i32 0, i32 1
  %9 = load i64, i64* %8, align 8
  %10 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.10, i64 0, i64 0), i32 noundef signext %7, i64 noundef %9)
  %11 = bitcast %struct.data1* %2 to i8*
  %12 = bitcast %struct.data1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %11, i8* align 8 %12, i64 16, i1 false)
  %13 = bitcast %struct.data1* %2 to [2 x i64]*
  %14 = load [2 x i64], [2 x i64]* %13, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @fn2([2 x i64] %0) #0 {
  %2 = alloca %struct.data2, align 8
  %3 = alloca %struct.data2, align 8
  %4 = bitcast %struct.data2* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = getelementptr inbounds %struct.data2, %struct.data2* %3, i32 0, i32 0
  %6 = load i32, i32* %5, align 8
  %7 = getelementptr inbounds %struct.data2, %struct.data2* %3, i32 0, i32 1
  %8 = load i64, i64* %7, align 8
  %9 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.11, i64 0, i64 0), i32 noundef signext %6, i64 noundef %8)
  %10 = bitcast %struct.data2* %2 to i8*
  %11 = bitcast %struct.data2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 8 %11, i64 16, i1 false)
  %12 = bitcast %struct.data2* %2 to [2 x i64]*
  %13 = load [2 x i64], [2 x i64]* %12, align 8
  ret [2 x i64] %13
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @fn3([2 x i64] %0) #0 {
  %2 = alloca %struct.data3, align 8
  %3 = alloca %struct.data3, align 8
  %4 = bitcast %struct.data3* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = getelementptr inbounds %struct.data3, %struct.data3* %3, i32 0, i32 0
  %6 = load i64, i64* %5, align 8
  %7 = getelementptr inbounds %struct.data3, %struct.data3* %3, i32 0, i32 1
  %8 = load i8, i8* %7, align 8
  %9 = zext i8 %8 to i32
  %10 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.12, i64 0, i64 0), i64 noundef %6, i32 noundef signext %9)
  %11 = bitcast %struct.data3* %2 to i8*
  %12 = bitcast %struct.data3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %11, i8* align 8 %12, i64 16, i1 false)
  %13 = bitcast %struct.data3* %2 to [2 x i64]*
  %14 = load [2 x i64], [2 x i64]* %13, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @ff1(i64 %0) #0 {
  %2 = alloca %struct.fdata1, align 4
  %3 = alloca %struct.fdata1, align 4
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  %6 = getelementptr inbounds %struct.fdata1, %struct.fdata1* %3, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast float* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %7, i8* align 8 %8, i64 4, i1 false)
  %9 = getelementptr inbounds %struct.fdata1, %struct.fdata1* %3, i32 0, i32 0
  %10 = load float, float* %9, align 4
  %11 = fpext float %10 to double
  %12 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([9 x i8], [9 x i8]* @.str.13, i64 0, i64 0), double noundef %11)
  %13 = bitcast %struct.fdata1* %2 to i8*
  %14 = bitcast %struct.fdata1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %13, i8* align 4 %14, i64 4, i1 false)
  %15 = getelementptr inbounds %struct.fdata1, %struct.fdata1* %2, i32 0, i32 0
  %16 = bitcast i64* %5 to i8*
  %17 = bitcast float* %15 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %16, i8* align 4 %17, i64 4, i1 false)
  %18 = load i64, i64* %5, align 8
  ret i64 %18
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @dd1(i64 %0) #0 {
  %2 = alloca %struct.ddata1, align 8
  %3 = alloca %struct.ddata1, align 8
  %4 = getelementptr inbounds %struct.ddata1, %struct.ddata1* %3, i32 0, i32 0
  %5 = bitcast double* %4 to i64*
  store i64 %0, i64* %5, align 8
  %6 = getelementptr inbounds %struct.ddata1, %struct.ddata1* %3, i32 0, i32 0
  %7 = load double, double* %6, align 8
  %8 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([9 x i8], [9 x i8]* @.str.14, i64 0, i64 0), double noundef %7)
  %9 = bitcast %struct.ddata1* %2 to i8*
  %10 = bitcast %struct.ddata1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %9, i8* align 8 %10, i64 8, i1 false)
  %11 = getelementptr inbounds %struct.ddata1, %struct.ddata1* %2, i32 0, i32 0
  %12 = bitcast double* %11 to i64*
  %13 = load i64, i64* %12, align 8
  ret i64 %13
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @dd2([2 x i64] %0) #0 {
  %2 = alloca %struct.ddata2, align 8
  %3 = alloca %struct.ddata2, align 8
  %4 = bitcast %struct.ddata2* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = getelementptr inbounds %struct.ddata2, %struct.ddata2* %3, i32 0, i32 0
  %6 = load double, double* %5, align 8
  %7 = getelementptr inbounds %struct.ddata2, %struct.ddata2* %3, i32 0, i32 1
  %8 = load double, double* %7, align 8
  %9 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([12 x i8], [12 x i8]* @.str.15, i64 0, i64 0), double noundef %6, double noundef %8)
  %10 = bitcast %struct.ddata2* %2 to i8*
  %11 = bitcast %struct.ddata2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 8 %11, i64 16, i1 false)
  %12 = bitcast %struct.ddata2* %2 to [2 x i64]*
  %13 = load [2 x i64], [2 x i64]* %12, align 8
  ret [2 x i64] %13
}

; Function Attrs: noinline nounwind optnone
define dso_local void @dd3(%struct.ddata3* noalias sret(%struct.ddata3) align 8 %0, %struct.ddata3* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.ddata3* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 0
  %6 = load double, double* %5, align 8
  %7 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 1
  %8 = load double, double* %7, align 8
  %9 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 2
  %10 = load double, double* %9, align 8
  %11 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.16, i64 0, i64 0), double noundef %6, double noundef %8, double noundef %10)
  %12 = bitcast %struct.ddata3* %0 to i8*
  %13 = bitcast %struct.ddata3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 8 %13, i64 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @ff2i(i64 %0) #0 {
  %2 = alloca %struct.fdata2i, align 4
  %3 = alloca %struct.fdata2i, align 4
  %4 = bitcast %struct.fdata2i* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = getelementptr inbounds %struct.fdata2i, %struct.fdata2i* %3, i32 0, i32 0
  %6 = load float, float* %5, align 4
  %7 = fpext float %6 to double
  %8 = getelementptr inbounds %struct.fdata2i, %struct.fdata2i* %3, i32 0, i32 1
  %9 = load i32, i32* %8, align 4
  %10 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([13 x i8], [13 x i8]* @.str.17, i64 0, i64 0), double noundef %7, i32 noundef signext %9)
  %11 = bitcast %struct.fdata2i* %2 to i8*
  %12 = bitcast %struct.fdata2i* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %11, i8* align 4 %12, i64 8, i1 false)
  %13 = bitcast %struct.fdata2i* %2 to i64*
  %14 = load i64, i64* %13, align 4
  ret i64 %14
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @ff2(i64 %0) #0 {
  %2 = alloca %struct.fdata2, align 4
  %3 = alloca %struct.fdata2, align 4
  %4 = bitcast %struct.fdata2* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = getelementptr inbounds %struct.fdata2, %struct.fdata2* %3, i32 0, i32 0
  %6 = load float, float* %5, align 4
  %7 = fpext float %6 to double
  %8 = getelementptr inbounds %struct.fdata2, %struct.fdata2* %3, i32 0, i32 1
  %9 = load float, float* %8, align 4
  %10 = fpext float %9 to double
  %11 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([12 x i8], [12 x i8]* @.str.18, i64 0, i64 0), double noundef %7, double noundef %10)
  %12 = bitcast %struct.fdata2* %2 to i8*
  %13 = bitcast %struct.fdata2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %12, i8* align 4 %13, i64 8, i1 false)
  %14 = bitcast %struct.fdata2* %2 to i64*
  %15 = load i64, i64* %14, align 4
  ret i64 %15
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @ff3([2 x i64] %0) #0 {
  %2 = alloca %struct.fdata3, align 4
  %3 = alloca %struct.fdata3, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.fdata3* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %3, i32 0, i32 0
  %9 = load float, float* %8, align 4
  %10 = fpext float %9 to double
  %11 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %3, i32 0, i32 1
  %12 = load float, float* %11, align 4
  %13 = fpext float %12 to double
  %14 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %3, i32 0, i32 2
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.19, i64 0, i64 0), double noundef %10, double noundef %13, double noundef %16)
  %18 = bitcast %struct.fdata3* %2 to i8*
  %19 = bitcast %struct.fdata3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %18, i8* align 4 %19, i64 12, i1 false)
  %20 = bitcast [2 x i64]* %5 to i8*
  %21 = bitcast %struct.fdata3* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %20, i8* align 4 %21, i64 12, i1 false)
  %22 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %22
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @ff4([2 x i64] %0) #0 {
  %2 = alloca %struct.fdata4, align 4
  %3 = alloca %struct.fdata4, align 4
  %4 = bitcast %struct.fdata4* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %3, i32 0, i32 0
  %6 = load float, float* %5, align 4
  %7 = fpext float %6 to double
  %8 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %3, i32 0, i32 1
  %9 = load float, float* %8, align 4
  %10 = fpext float %9 to double
  %11 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %3, i32 0, i32 2
  %12 = load float, float* %11, align 4
  %13 = fpext float %12 to double
  %14 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %3, i32 0, i32 3
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.20, i64 0, i64 0), double noundef %7, double noundef %10, double noundef %13, double noundef %16)
  %18 = bitcast %struct.fdata4* %2 to i8*
  %19 = bitcast %struct.fdata4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %18, i8* align 4 %19, i64 16, i1 false)
  %20 = bitcast %struct.fdata4* %2 to [2 x i64]*
  %21 = load [2 x i64], [2 x i64]* %20, align 4
  ret [2 x i64] %21
}

; Function Attrs: noinline nounwind optnone
define dso_local void @ff5(%struct.fdata5* noalias sret(%struct.fdata5) align 4 %0, %struct.fdata5* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.fdata5* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 0
  %6 = load float, float* %5, align 4
  %7 = fpext float %6 to double
  %8 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 1
  %9 = load float, float* %8, align 4
  %10 = fpext float %9 to double
  %11 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 2
  %12 = load float, float* %11, align 4
  %13 = fpext float %12 to double
  %14 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 3
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 4
  %18 = load float, float* %17, align 4
  %19 = fpext float %18 to double
  %20 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([21 x i8], [21 x i8]* @.str.21, i64 0, i64 0), double noundef %7, double noundef %10, double noundef %13, double noundef %16, double noundef %19)
  %21 = bitcast %struct.fdata5* %0 to i8*
  %22 = bitcast %struct.fdata5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %21, i8* align 4 %22, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @ff2id([2 x i64] %0) #0 {
  %2 = alloca %struct.fdata2id, align 8
  %3 = alloca %struct.fdata2id, align 8
  %4 = bitcast %struct.fdata2id* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %3, i32 0, i32 0
  %6 = load i8, i8* %5, align 8
  %7 = zext i8 %6 to i32
  %8 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %3, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = zext i8 %9 to i32
  %11 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %3, i32 0, i32 2
  %12 = load double, double* %11, align 8
  %13 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.22, i64 0, i64 0), i32 noundef signext %7, i32 noundef signext %10, double noundef %12)
  %14 = bitcast %struct.fdata2id* %2 to i8*
  %15 = bitcast %struct.fdata2id* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %14, i8* align 8 %15, i64 16, i1 false)
  %16 = bitcast %struct.fdata2id* %2 to [2 x i64]*
  %17 = load [2 x i64], [2 x i64]* %16, align 8
  ret [2 x i64] %17
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @ff7if([2 x i64] %0) #0 {
  %2 = alloca %struct.fdata7if, align 4
  %3 = alloca %struct.fdata7if, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.fdata7if* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %3, i32 0, i32 0
  %9 = getelementptr inbounds [7 x i8], [7 x i8]* %8, i64 0, i64 0
  %10 = load i8, i8* %9, align 4
  %11 = zext i8 %10 to i32
  %12 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %3, i32 0, i32 0
  %13 = getelementptr inbounds [7 x i8], [7 x i8]* %12, i64 0, i64 1
  %14 = load i8, i8* %13, align 1
  %15 = zext i8 %14 to i32
  %16 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %3, i32 0, i32 1
  %17 = load float, float* %16, align 4
  %18 = fpext float %17 to double
  %19 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([17 x i8], [17 x i8]* @.str.23, i64 0, i64 0), i32 noundef signext %11, i32 noundef signext %15, double noundef %18)
  %20 = bitcast %struct.fdata7if* %2 to i8*
  %21 = bitcast %struct.fdata7if* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %20, i8* align 4 %21, i64 12, i1 false)
  %22 = bitcast [2 x i64]* %5 to i8*
  %23 = bitcast %struct.fdata7if* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %22, i8* align 4 %23, i64 12, i1 false)
  %24 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %24
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @ff4if([2 x i64] %0) #0 {
  %2 = alloca %struct.fdata4if, align 4
  %3 = alloca %struct.fdata4if, align 4
  %4 = bitcast %struct.fdata4if* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %3, i32 0, i32 0
  %6 = load float, float* %5, align 4
  %7 = fpext float %6 to double
  %8 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %3, i32 0, i32 1
  %9 = load i8, i8* %8, align 4
  %10 = zext i8 %9 to i32
  %11 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %3, i32 0, i32 2
  %12 = load float, float* %11, align 4
  %13 = fpext float %12 to double
  %14 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %3, i32 0, i32 3
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([20 x i8], [20 x i8]* @.str.24, i64 0, i64 0), double noundef %7, i32 noundef signext %10, double noundef %13, double noundef %16)
  %18 = bitcast %struct.fdata4if* %2 to i8*
  %19 = bitcast %struct.fdata4if* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %18, i8* align 4 %19, i64 16, i1 false)
  %20 = bitcast %struct.fdata4if* %2 to [2 x i64]*
  %21 = load [2 x i64], [2 x i64]* %20, align 4
  ret [2 x i64] %21
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo(%struct.array* noalias sret(%struct.array) align 4 %0, %struct.array* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %6 = getelementptr inbounds [8 x i32], [8 x i32]* %5, i64 0, i64 0
  %7 = load i32, i32* %6, align 4
  %8 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %9 = getelementptr inbounds [8 x i32], [8 x i32]* %8, i64 0, i64 1
  %10 = load i32, i32* %9, align 4
  %11 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %12 = getelementptr inbounds [8 x i32], [8 x i32]* %11, i64 0, i64 2
  %13 = load i32, i32* %12, align 4
  %14 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.25, i64 0, i64 0), i32 noundef signext %7, i32 noundef signext %10, i32 noundef signext %13)
  %15 = bitcast %struct.array* %0 to i8*
  %16 = bitcast %struct.array* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %15, i8* align 4 %16, i64 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo2(%struct.array* noalias sret(%struct.array) align 4 %0, i32 noundef signext %1) #0 {
  %3 = alloca i8*, align 8
  %4 = alloca i32, align 4
  %5 = alloca i32, align 4
  %6 = bitcast %struct.array* %0 to i8*
  store i8* %6, i8** %3, align 8
  store i32 %1, i32* %4, align 4
  store i32 0, i32* %5, align 4
  br label %7

7:                                                ; preds = %18, %2
  %8 = load i32, i32* %5, align 4
  %9 = icmp slt i32 %8, 8
  br i1 %9, label %10, label %21

10:                                               ; preds = %7
  %11 = load i32, i32* %5, align 4
  %12 = load i32, i32* %4, align 4
  %13 = add nsw i32 %11, %12
  %14 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %15 = load i32, i32* %5, align 4
  %16 = sext i32 %15 to i64
  %17 = getelementptr inbounds [8 x i32], [8 x i32]* %14, i64 0, i64 %16
  store i32 %13, i32* %17, align 4
  br label %18

18:                                               ; preds = %10
  %19 = load i32, i32* %5, align 4
  %20 = add nsw i32 %19, 1
  store i32 %20, i32* %5, align 4
  br label %7, !llvm.loop !5

21:                                               ; preds = %7
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @callback(void (%struct.array*, %struct.array*, i64, [2 x i64])* noundef %0, %struct.array* noundef %1) #0 {
  %3 = alloca void (%struct.array*, %struct.array*, i64, [2 x i64])*, align 8
  %4 = alloca %struct.array, align 4
  %5 = alloca %struct.array, align 4
  %6 = alloca %struct.point, align 4
  %7 = alloca %struct.point1, align 4
  %8 = alloca %struct.array, align 4
  %9 = alloca %struct.array, align 4
  %10 = alloca [2 x i64], align 8
  %11 = alloca %struct.array, align 4
  %12 = alloca %struct.array, align 4
  store void (%struct.array*, %struct.array*, i64, [2 x i64])* %0, void (%struct.array*, %struct.array*, i64, [2 x i64])** %3, align 8
  %13 = bitcast %struct.array* %5 to i8*
  %14 = bitcast %struct.array* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %13, i8* align 4 %14, i64 32, i1 false)
  call void @demo(%struct.array* sret(%struct.array) align 4 %4, %struct.array* noundef %5)
  %15 = bitcast %struct.point* %6 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %15, i8* align 4 bitcast (%struct.point* @__const.callback.pt to i8*), i64 8, i1 false)
  %16 = bitcast %struct.point1* %7 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %16, i8* align 4 bitcast (%struct.point1* @__const.callback.pt1 to i8*), i64 12, i1 false)
  %17 = load void (%struct.array*, %struct.array*, i64, [2 x i64])*, void (%struct.array*, %struct.array*, i64, [2 x i64])** %3, align 8
  %18 = bitcast %struct.array* %9 to i8*
  %19 = bitcast %struct.array* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %18, i8* align 4 %19, i64 32, i1 false)
  %20 = bitcast %struct.point* %6 to i64*
  %21 = load i64, i64* %20, align 4
  %22 = bitcast [2 x i64]* %10 to i8*
  %23 = bitcast %struct.point1* %7 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %22, i8* align 4 %23, i64 12, i1 false)
  %24 = load [2 x i64], [2 x i64]* %10, align 8
  call void %17(%struct.array* sret(%struct.array) align 4 %8, %struct.array* noundef %9, i64 %21, [2 x i64] %24)
  %25 = bitcast %struct.array* %12 to i8*
  %26 = bitcast %struct.array* %8 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %25, i8* align 4 %26, i64 32, i1 false)
  call void @demo(%struct.array* sret(%struct.array) align 4 %11, %struct.array* noundef %12)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @callback1(i64 (%struct.array*, i64, [2 x i64])* noundef %0, %struct.array* noundef %1) #0 {
  %3 = alloca i64 (%struct.array*, i64, [2 x i64])*, align 8
  %4 = alloca %struct.point, align 4
  %5 = alloca %struct.point1, align 4
  %6 = alloca %struct.point, align 4
  %7 = alloca %struct.array, align 4
  %8 = alloca [2 x i64], align 8
  store i64 (%struct.array*, i64, [2 x i64])* %0, i64 (%struct.array*, i64, [2 x i64])** %3, align 8
  %9 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %10 = getelementptr inbounds [8 x i32], [8 x i32]* %9, i64 0, i64 0
  %11 = load i32, i32* %10, align 4
  %12 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %13 = getelementptr inbounds [8 x i32], [8 x i32]* %12, i64 0, i64 1
  %14 = load i32, i32* %13, align 4
  %15 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %16 = getelementptr inbounds [8 x i32], [8 x i32]* %15, i64 0, i64 7
  %17 = load i32, i32* %16, align 4
  %18 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([27 x i8], [27 x i8]* @.str.26, i64 0, i64 0), i32 noundef signext %11, i32 noundef signext %14, i32 noundef signext %17)
  %19 = bitcast %struct.point* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %19, i8* align 4 bitcast (%struct.point* @__const.callback1.pt to i8*), i64 8, i1 false)
  %20 = bitcast %struct.point1* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %20, i8* align 4 bitcast (%struct.point1* @__const.callback1.pt1 to i8*), i64 12, i1 false)
  %21 = load i64 (%struct.array*, i64, [2 x i64])*, i64 (%struct.array*, i64, [2 x i64])** %3, align 8
  %22 = bitcast %struct.array* %7 to i8*
  %23 = bitcast %struct.array* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %22, i8* align 4 %23, i64 32, i1 false)
  %24 = bitcast %struct.point* %4 to i64*
  %25 = load i64, i64* %24, align 4
  %26 = bitcast [2 x i64]* %8 to i8*
  %27 = bitcast %struct.point1* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %26, i8* align 4 %27, i64 12, i1 false)
  %28 = load [2 x i64], [2 x i64]* %8, align 8
  %29 = call i64 %21(%struct.array* noundef %7, i64 %25, [2 x i64] %28)
  %30 = bitcast %struct.point* %6 to i64*
  store i64 %29, i64* %30, align 4
  %31 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 0
  %32 = load i32, i32* %31, align 4
  %33 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 1
  %34 = load i32, i32* %33, align 4
  %35 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([22 x i8], [22 x i8]* @.str.27, i64 0, i64 0), i32 noundef signext %32, i32 noundef signext %34)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @mycallback(%struct.array* noundef %0, i64 %1, [2 x i64] %2) #0 {
  %4 = alloca %struct.point, align 4
  %5 = alloca %struct.point, align 4
  %6 = alloca %struct.point1, align 4
  %7 = alloca [2 x i64], align 8
  %8 = bitcast %struct.point* %5 to i64*
  store i64 %1, i64* %8, align 4
  store [2 x i64] %2, [2 x i64]* %7, align 8
  %9 = bitcast %struct.point1* %6 to i8*
  %10 = bitcast [2 x i64]* %7 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 8 %10, i64 12, i1 false)
  %11 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %12 = getelementptr inbounds [8 x i32], [8 x i32]* %11, i64 0, i64 0
  %13 = load i32, i32* %12, align 4
  %14 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %15 = getelementptr inbounds [8 x i32], [8 x i32]* %14, i64 0, i64 1
  %16 = load i32, i32* %15, align 4
  %17 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %18 = getelementptr inbounds [8 x i32], [8 x i32]* %17, i64 0, i64 7
  %19 = load i32, i32* %18, align 4
  %20 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([28 x i8], [28 x i8]* @.str.28, i64 0, i64 0), i32 noundef signext %13, i32 noundef signext %16, i32 noundef signext %19)
  %21 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 0
  %22 = load i32, i32* %21, align 4
  %23 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 1
  %24 = load i32, i32* %23, align 4
  %25 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([22 x i8], [22 x i8]* @.str.29, i64 0, i64 0), i32 noundef signext %22, i32 noundef signext %24)
  %26 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 0
  %27 = load i32, i32* %26, align 4
  %28 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 1
  %29 = load i32, i32* %28, align 4
  %30 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 2
  %31 = load i32, i32* %30, align 4
  %32 = call signext i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([26 x i8], [26 x i8]* @.str.30, i64 0, i64 0), i32 noundef signext %27, i32 noundef signext %29, i32 noundef signext %31)
  %33 = getelementptr inbounds %struct.point, %struct.point* %4, i32 0, i32 0
  %34 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 0
  %35 = load i32, i32* %34, align 4
  %36 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 0
  %37 = load i32, i32* %36, align 4
  %38 = add nsw i32 %35, %37
  store i32 %38, i32* %33, align 4
  %39 = getelementptr inbounds %struct.point, %struct.point* %4, i32 0, i32 1
  %40 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 1
  %41 = load i32, i32* %40, align 4
  %42 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 1
  %43 = load i32, i32* %42, align 4
  %44 = add nsw i32 %41, %43
  store i32 %44, i32* %39, align 4
  %45 = bitcast %struct.point* %4 to i64*
  %46 = load i64, i64* %45, align 4
  ret i64 %46
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+64bit,+a,+c,+m,+relax,-save-restore" }
attributes #1 = { "frame-pointer"="all" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+64bit,+a,+c,+m,+relax,-save-restore" }
attributes #2 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3}
!llvm.ident = !{!4}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"target-abi", !"lp64"}
!2 = !{i32 7, !"frame-pointer", i32 2}
!3 = !{i32 1, !"SmallDataLimit", i32 8}
!4 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
!5 = distinct !{!5, !6}
!6 = !{!"llvm.loop.mustprogress"}
