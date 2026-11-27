; ModuleID = '../../wrap/demo.c'
source_filename = "../../wrap/demo.c"
target datalayout = "e-m:e-p270:32:32-p271:32:32-p272:64:64-i64:64-f80:128-n8:16:32:64-S128"
target triple = "amd64-unknown-linux-gnu"

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

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo32(i32 noundef %0) #0 {
  %2 = alloca i32, align 4
  store i32 %0, i32* %2, align 4
  %3 = load i32, i32* %2, align 4
  %4 = add nsw i32 %3, 100
  ret i32 %4
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @demo64(i64 noundef %0) #0 {
  %2 = alloca i64, align 8
  store i64 %0, i64* %2, align 8
  %3 = load i64, i64* %2, align 8
  %4 = add nsw i64 %3, 100
  ret i64 %4
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i64 } @pt64(i64 %0, i64 %1) #0 {
  %3 = alloca %struct.point64, align 8
  %4 = alloca %struct.point64, align 8
  %5 = bitcast %struct.point64* %4 to { i64, i64 }*
  %6 = getelementptr inbounds { i64, i64 }, { i64, i64 }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 8
  %7 = getelementptr inbounds { i64, i64 }, { i64, i64 }* %5, i32 0, i32 1
  store i64 %1, i64* %7, align 8
  %8 = getelementptr inbounds %struct.point64, %struct.point64* %4, i32 0, i32 0
  %9 = load i64, i64* %8, align 8
  %10 = getelementptr inbounds %struct.point64, %struct.point64* %4, i32 0, i32 1
  %11 = load i64, i64* %10, align 8
  %12 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([20 x i8], [20 x i8]* @.str, i64 0, i64 0), i64 noundef %9, i64 noundef %11)
  %13 = bitcast %struct.point64* %3 to i8*
  %14 = bitcast %struct.point64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %13, i8* align 8 %14, i64 16, i1 false)
  %15 = bitcast %struct.point64* %3 to { i64, i64 }*
  %16 = load { i64, i64 }, { i64, i64 }* %15, align 8
  ret { i64, i64 } %16
}

declare i32 @printf(i8* noundef, ...) #1

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #2

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo32s(i32 %0) #0 {
  %2 = alloca %struct.struct32, align 4
  %3 = alloca %struct.struct32, align 4
  %4 = getelementptr inbounds %struct.struct32, %struct.struct32* %3, i32 0, i32 0
  store i32 %0, i32* %4, align 4
  %5 = getelementptr inbounds %struct.struct32, %struct.struct32* %3, i32 0, i32 0
  %6 = load i32, i32* %5, align 4
  %7 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([14 x i8], [14 x i8]* @.str.1, i64 0, i64 0), i32 noundef %6)
  %8 = getelementptr inbounds %struct.struct32, %struct.struct32* %2, i32 0, i32 0
  %9 = getelementptr inbounds %struct.struct32, %struct.struct32* %3, i32 0, i32 0
  %10 = load i32, i32* %9, align 4
  %11 = add nsw i32 %10, 100
  store i32 %11, i32* %8, align 4
  %12 = getelementptr inbounds %struct.struct32, %struct.struct32* %2, i32 0, i32 0
  %13 = load i32, i32* %12, align 4
  ret i32 %13
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @pt(i64 %0) #0 {
  %2 = alloca %struct.point, align 4
  %3 = alloca %struct.point, align 4
  %4 = bitcast %struct.point* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = getelementptr inbounds %struct.point, %struct.point* %3, i32 0, i32 0
  %6 = load i32, i32* %5, align 4
  %7 = getelementptr inbounds %struct.point, %struct.point* %3, i32 0, i32 1
  %8 = load i32, i32* %7, align 4
  %9 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([14 x i8], [14 x i8]* @.str.2, i64 0, i64 0), i32 noundef %6, i32 noundef %8)
  %10 = bitcast %struct.point* %2 to i8*
  %11 = bitcast %struct.point* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %10, i8* align 4 %11, i64 8, i1 false)
  %12 = bitcast %struct.point* %2 to i64*
  %13 = load i64, i64* %12, align 4
  ret i64 %13
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i32 } @pt1(i64 %0, i32 %1) #0 {
  %3 = alloca %struct.point1, align 4
  %4 = alloca %struct.point1, align 4
  %5 = alloca { i64, i32 }, align 4
  %6 = alloca { i64, i32 }, align 8
  %7 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 4
  %8 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 1
  store i32 %1, i32* %8, align 4
  %9 = bitcast %struct.point1* %4 to i8*
  %10 = bitcast { i64, i32 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 4 %10, i64 12, i1 false)
  %11 = getelementptr inbounds %struct.point1, %struct.point1* %4, i32 0, i32 0
  %12 = load i32, i32* %11, align 4
  %13 = getelementptr inbounds %struct.point1, %struct.point1* %4, i32 0, i32 1
  %14 = load i32, i32* %13, align 4
  %15 = getelementptr inbounds %struct.point1, %struct.point1* %4, i32 0, i32 2
  %16 = load i32, i32* %15, align 4
  %17 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.3, i64 0, i64 0), i32 noundef %12, i32 noundef %14, i32 noundef %16)
  %18 = bitcast %struct.point1* %3 to i8*
  %19 = bitcast %struct.point1* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %18, i8* align 4 %19, i64 12, i1 false)
  %20 = bitcast { i64, i32 }* %6 to i8*
  %21 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %20, i8* align 4 %21, i64 12, i1 false)
  %22 = load { i64, i32 }, { i64, i32 }* %6, align 8
  ret { i64, i32 } %22
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i32 } @pt2(i64 %0, i32 %1) #0 {
  %3 = alloca %struct.point2, align 4
  %4 = alloca %struct.point2, align 4
  %5 = alloca { i64, i32 }, align 4
  %6 = alloca { i64, i32 }, align 8
  %7 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 4
  %8 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 1
  store i32 %1, i32* %8, align 4
  %9 = bitcast %struct.point2* %4 to i8*
  %10 = bitcast { i64, i32 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 4 %10, i64 12, i1 false)
  %11 = getelementptr inbounds %struct.point2, %struct.point2* %4, i32 0, i32 0
  %12 = load i8, i8* %11, align 4
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point2, %struct.point2* %4, i32 0, i32 1
  %15 = load i32, i32* %14, align 4
  %16 = getelementptr inbounds %struct.point2, %struct.point2* %4, i32 0, i32 2
  %17 = load i32, i32* %16, align 4
  %18 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.4, i64 0, i64 0), i32 noundef %13, i32 noundef %15, i32 noundef %17)
  %19 = bitcast %struct.point2* %3 to i8*
  %20 = bitcast %struct.point2* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %19, i8* align 4 %20, i64 12, i1 false)
  %21 = bitcast { i64, i32 }* %6 to i8*
  %22 = bitcast %struct.point2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %21, i8* align 4 %22, i64 12, i1 false)
  %23 = load { i64, i32 }, { i64, i32 }* %6, align 8
  ret { i64, i32 } %23
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i24 @pt3(i24 %0) #0 {
  %2 = alloca %struct.point3, align 1
  %3 = alloca %struct.point3, align 1
  %4 = alloca i24, align 4
  %5 = alloca i24, align 4
  store i24 %0, i24* %4, align 4
  %6 = bitcast %struct.point3* %3 to i8*
  %7 = bitcast i24* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 4 %7, i64 3, i1 false)
  %8 = getelementptr inbounds %struct.point3, %struct.point3* %3, i32 0, i32 0
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point3, %struct.point3* %3, i32 0, i32 1
  %12 = load i8, i8* %11, align 1
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point3, %struct.point3* %3, i32 0, i32 2
  %15 = load i8, i8* %14, align 1
  %16 = sext i8 %15 to i32
  %17 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.5, i64 0, i64 0), i32 noundef %10, i32 noundef %13, i32 noundef %16)
  %18 = bitcast %struct.point3* %2 to i8*
  %19 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %18, i8* align 1 %19, i64 3, i1 false)
  %20 = bitcast i24* %5 to i8*
  %21 = bitcast %struct.point3* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %20, i8* align 1 %21, i64 3, i1 false)
  %22 = load i24, i24* %5, align 4
  ret i24 %22
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @pt4(i64 %0) #0 {
  %2 = alloca %struct.point4, align 4
  %3 = alloca %struct.point4, align 4
  %4 = bitcast %struct.point4* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = getelementptr inbounds %struct.point4, %struct.point4* %3, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point4, %struct.point4* %3, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point4, %struct.point4* %3, i32 0, i32 2
  %12 = load i8, i8* %11, align 2
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point4, %struct.point4* %3, i32 0, i32 3
  %15 = load i32, i32* %14, align 4
  %16 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([21 x i8], [21 x i8]* @.str.6, i64 0, i64 0), i32 noundef %7, i32 noundef %10, i32 noundef %13, i32 noundef %15)
  %17 = bitcast %struct.point4* %2 to i8*
  %18 = bitcast %struct.point4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %17, i8* align 4 %18, i64 8, i1 false)
  %19 = bitcast %struct.point4* %2 to i64*
  %20 = load i64, i64* %19, align 4
  ret i64 %20
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i40 @pt5(i40 %0) #0 {
  %2 = alloca %struct.point5, align 1
  %3 = alloca %struct.point5, align 1
  %4 = alloca i40, align 8
  %5 = alloca i40, align 8
  store i40 %0, i40* %4, align 8
  %6 = bitcast %struct.point5* %3 to i8*
  %7 = bitcast i40* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 8 %7, i64 5, i1 false)
  %8 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 0
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 1
  %12 = load i8, i8* %11, align 1
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 2
  %15 = load i8, i8* %14, align 1
  %16 = sext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 3
  %18 = load i8, i8* %17, align 1
  %19 = sext i8 %18 to i32
  %20 = getelementptr inbounds %struct.point5, %struct.point5* %3, i32 0, i32 4
  %21 = load i8, i8* %20, align 1
  %22 = sext i8 %21 to i32
  %23 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([24 x i8], [24 x i8]* @.str.7, i64 0, i64 0), i32 noundef %10, i32 noundef %13, i32 noundef %16, i32 noundef %19, i32 noundef %22)
  %24 = bitcast %struct.point5* %2 to i8*
  %25 = bitcast %struct.point5* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %24, i8* align 1 %25, i64 5, i1 false)
  %26 = bitcast i40* %5 to i8*
  %27 = bitcast %struct.point5* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %26, i8* align 1 %27, i64 5, i1 false)
  %28 = load i40, i40* %5, align 8
  ret i40 %28
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i32 } @pt6(i64 %0, i32 %1) #0 {
  %3 = alloca %struct.point6, align 4
  %4 = alloca %struct.point6, align 4
  %5 = alloca { i64, i32 }, align 4
  %6 = alloca { i64, i32 }, align 8
  %7 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 4
  %8 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 1
  store i32 %1, i32* %8, align 4
  %9 = bitcast %struct.point6* %4 to i8*
  %10 = bitcast { i64, i32 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 4 %10, i64 12, i1 false)
  %11 = getelementptr inbounds %struct.point6, %struct.point6* %4, i32 0, i32 0
  %12 = load i8, i8* %11, align 4
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point6, %struct.point6* %4, i32 0, i32 1
  %15 = load i8, i8* %14, align 1
  %16 = sext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point6, %struct.point6* %4, i32 0, i32 2
  %18 = load i8, i8* %17, align 2
  %19 = sext i8 %18 to i32
  %20 = getelementptr inbounds %struct.point6, %struct.point6* %4, i32 0, i32 3
  %21 = load i8, i8* %20, align 1
  %22 = sext i8 %21 to i32
  %23 = getelementptr inbounds %struct.point6, %struct.point6* %4, i32 0, i32 4
  %24 = load i8, i8* %23, align 4
  %25 = sext i8 %24 to i32
  %26 = getelementptr inbounds %struct.point6, %struct.point6* %4, i32 0, i32 5
  %27 = load i32, i32* %26, align 4
  %28 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([27 x i8], [27 x i8]* @.str.8, i64 0, i64 0), i32 noundef %13, i32 noundef %16, i32 noundef %19, i32 noundef %22, i32 noundef %25, i32 noundef %27)
  %29 = bitcast %struct.point6* %3 to i8*
  %30 = bitcast %struct.point6* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %29, i8* align 4 %30, i64 12, i1 false)
  %31 = bitcast { i64, i32 }* %6 to i8*
  %32 = bitcast %struct.point6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %31, i8* align 4 %32, i64 12, i1 false)
  %33 = load { i64, i32 }, { i64, i32 }* %6, align 8
  ret { i64, i32 } %33
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i64 } @pt7(i64 %0, i64 %1) #0 {
  %3 = alloca %struct.point7, align 4
  %4 = alloca %struct.point7, align 4
  %5 = bitcast %struct.point7* %4 to { i64, i64 }*
  %6 = getelementptr inbounds { i64, i64 }, { i64, i64 }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 4
  %7 = getelementptr inbounds { i64, i64 }, { i64, i64 }* %5, i32 0, i32 1
  store i64 %1, i64* %7, align 4
  %8 = getelementptr inbounds %struct.point7, %struct.point7* %4, i32 0, i32 0
  %9 = load i8, i8* %8, align 4
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point7, %struct.point7* %4, i32 0, i32 1
  %12 = load i8, i8* %11, align 1
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point7, %struct.point7* %4, i32 0, i32 2
  %15 = load i8, i8* %14, align 2
  %16 = sext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point7, %struct.point7* %4, i32 0, i32 3
  %18 = load i8, i8* %17, align 1
  %19 = sext i8 %18 to i32
  %20 = getelementptr inbounds %struct.point7, %struct.point7* %4, i32 0, i32 4
  %21 = load i8, i8* %20, align 4
  %22 = sext i8 %21 to i32
  %23 = getelementptr inbounds %struct.point7, %struct.point7* %4, i32 0, i32 5
  %24 = load i32, i32* %23, align 4
  %25 = getelementptr inbounds %struct.point7, %struct.point7* %4, i32 0, i32 6
  %26 = load i8, i8* %25, align 4
  %27 = sext i8 %26 to i32
  %28 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([30 x i8], [30 x i8]* @.str.9, i64 0, i64 0), i32 noundef %10, i32 noundef %13, i32 noundef %16, i32 noundef %19, i32 noundef %22, i32 noundef %24, i32 noundef %27)
  %29 = bitcast %struct.point7* %3 to i8*
  %30 = bitcast %struct.point7* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %29, i8* align 4 %30, i64 16, i1 false)
  %31 = bitcast %struct.point7* %3 to { i64, i64 }*
  %32 = load { i64, i64 }, { i64, i64 }* %31, align 4
  ret { i64, i64 } %32
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i8, i64 } @fn1(i8 %0, i64 %1) #0 {
  %3 = alloca %struct.data1, align 8
  %4 = alloca %struct.data1, align 8
  %5 = bitcast %struct.data1* %4 to { i8, i64 }*
  %6 = getelementptr inbounds { i8, i64 }, { i8, i64 }* %5, i32 0, i32 0
  store i8 %0, i8* %6, align 8
  %7 = getelementptr inbounds { i8, i64 }, { i8, i64 }* %5, i32 0, i32 1
  store i64 %1, i64* %7, align 8
  %8 = getelementptr inbounds %struct.data1, %struct.data1* %4, i32 0, i32 0
  %9 = load i8, i8* %8, align 8
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.data1, %struct.data1* %4, i32 0, i32 1
  %12 = load i64, i64* %11, align 8
  %13 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.10, i64 0, i64 0), i32 noundef %10, i64 noundef %12)
  %14 = bitcast %struct.data1* %3 to i8*
  %15 = bitcast %struct.data1* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %14, i8* align 8 %15, i64 16, i1 false)
  %16 = bitcast %struct.data1* %3 to { i8, i64 }*
  %17 = load { i8, i64 }, { i8, i64 }* %16, align 8
  ret { i8, i64 } %17
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i32, i64 } @fn2(i32 %0, i64 %1) #0 {
  %3 = alloca %struct.data2, align 8
  %4 = alloca %struct.data2, align 8
  %5 = bitcast %struct.data2* %4 to { i32, i64 }*
  %6 = getelementptr inbounds { i32, i64 }, { i32, i64 }* %5, i32 0, i32 0
  store i32 %0, i32* %6, align 8
  %7 = getelementptr inbounds { i32, i64 }, { i32, i64 }* %5, i32 0, i32 1
  store i64 %1, i64* %7, align 8
  %8 = getelementptr inbounds %struct.data2, %struct.data2* %4, i32 0, i32 0
  %9 = load i32, i32* %8, align 8
  %10 = getelementptr inbounds %struct.data2, %struct.data2* %4, i32 0, i32 1
  %11 = load i64, i64* %10, align 8
  %12 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.11, i64 0, i64 0), i32 noundef %9, i64 noundef %11)
  %13 = bitcast %struct.data2* %3 to i8*
  %14 = bitcast %struct.data2* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %13, i8* align 8 %14, i64 16, i1 false)
  %15 = bitcast %struct.data2* %3 to { i32, i64 }*
  %16 = load { i32, i64 }, { i32, i64 }* %15, align 8
  ret { i32, i64 } %16
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i8 } @fn3(i64 %0, i8 %1) #0 {
  %3 = alloca %struct.data3, align 8
  %4 = alloca %struct.data3, align 8
  %5 = bitcast %struct.data3* %4 to { i64, i8 }*
  %6 = getelementptr inbounds { i64, i8 }, { i64, i8 }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 8
  %7 = getelementptr inbounds { i64, i8 }, { i64, i8 }* %5, i32 0, i32 1
  store i8 %1, i8* %7, align 8
  %8 = getelementptr inbounds %struct.data3, %struct.data3* %4, i32 0, i32 0
  %9 = load i64, i64* %8, align 8
  %10 = getelementptr inbounds %struct.data3, %struct.data3* %4, i32 0, i32 1
  %11 = load i8, i8* %10, align 8
  %12 = sext i8 %11 to i32
  %13 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.12, i64 0, i64 0), i64 noundef %9, i32 noundef %12)
  %14 = bitcast %struct.data3* %3 to i8*
  %15 = bitcast %struct.data3* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %14, i8* align 8 %15, i64 16, i1 false)
  %16 = bitcast %struct.data3* %3 to { i64, i8 }*
  %17 = load { i64, i8 }, { i64, i8 }* %16, align 8
  ret { i64, i8 } %17
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local float @ff1(float %0) #0 {
  %2 = alloca %struct.fdata1, align 4
  %3 = alloca %struct.fdata1, align 4
  %4 = getelementptr inbounds %struct.fdata1, %struct.fdata1* %3, i32 0, i32 0
  store float %0, float* %4, align 4
  %5 = getelementptr inbounds %struct.fdata1, %struct.fdata1* %3, i32 0, i32 0
  %6 = load float, float* %5, align 4
  %7 = fpext float %6 to double
  %8 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([9 x i8], [9 x i8]* @.str.13, i64 0, i64 0), double noundef %7)
  %9 = bitcast %struct.fdata1* %2 to i8*
  %10 = bitcast %struct.fdata1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 4 %10, i64 4, i1 false)
  %11 = getelementptr inbounds %struct.fdata1, %struct.fdata1* %2, i32 0, i32 0
  %12 = load float, float* %11, align 4
  ret float %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local double @dd1(double %0) #0 {
  %2 = alloca %struct.ddata1, align 8
  %3 = alloca %struct.ddata1, align 8
  %4 = getelementptr inbounds %struct.ddata1, %struct.ddata1* %3, i32 0, i32 0
  store double %0, double* %4, align 8
  %5 = getelementptr inbounds %struct.ddata1, %struct.ddata1* %3, i32 0, i32 0
  %6 = load double, double* %5, align 8
  %7 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([9 x i8], [9 x i8]* @.str.14, i64 0, i64 0), double noundef %6)
  %8 = bitcast %struct.ddata1* %2 to i8*
  %9 = bitcast %struct.ddata1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 8, i1 false)
  %10 = getelementptr inbounds %struct.ddata1, %struct.ddata1* %2, i32 0, i32 0
  %11 = load double, double* %10, align 8
  ret double %11
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { double, double } @dd2(double %0, double %1) #0 {
  %3 = alloca %struct.ddata2, align 8
  %4 = alloca %struct.ddata2, align 8
  %5 = bitcast %struct.ddata2* %4 to { double, double }*
  %6 = getelementptr inbounds { double, double }, { double, double }* %5, i32 0, i32 0
  store double %0, double* %6, align 8
  %7 = getelementptr inbounds { double, double }, { double, double }* %5, i32 0, i32 1
  store double %1, double* %7, align 8
  %8 = getelementptr inbounds %struct.ddata2, %struct.ddata2* %4, i32 0, i32 0
  %9 = load double, double* %8, align 8
  %10 = getelementptr inbounds %struct.ddata2, %struct.ddata2* %4, i32 0, i32 1
  %11 = load double, double* %10, align 8
  %12 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([12 x i8], [12 x i8]* @.str.15, i64 0, i64 0), double noundef %9, double noundef %11)
  %13 = bitcast %struct.ddata2* %3 to i8*
  %14 = bitcast %struct.ddata2* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %13, i8* align 8 %14, i64 16, i1 false)
  %15 = bitcast %struct.ddata2* %3 to { double, double }*
  %16 = load { double, double }, { double, double }* %15, align 8
  ret { double, double } %16
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @dd3(%struct.ddata3* noalias sret(%struct.ddata3) align 8 %0, %struct.ddata3* noundef byval(%struct.ddata3) align 8 %1) #0 {
  %3 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 0
  %4 = load double, double* %3, align 8
  %5 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 1
  %6 = load double, double* %5, align 8
  %7 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 2
  %8 = load double, double* %7, align 8
  %9 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.16, i64 0, i64 0), double noundef %4, double noundef %6, double noundef %8)
  %10 = bitcast %struct.ddata3* %0 to i8*
  %11 = bitcast %struct.ddata3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 8 %11, i64 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
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
  %10 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([13 x i8], [13 x i8]* @.str.17, i64 0, i64 0), double noundef %7, i32 noundef %9)
  %11 = bitcast %struct.fdata2i* %2 to i8*
  %12 = bitcast %struct.fdata2i* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %11, i8* align 4 %12, i64 8, i1 false)
  %13 = bitcast %struct.fdata2i* %2 to i64*
  %14 = load i64, i64* %13, align 4
  ret i64 %14
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local <2 x float> @ff2(<2 x float> %0) #3 {
  %2 = alloca %struct.fdata2, align 4
  %3 = alloca %struct.fdata2, align 4
  %4 = bitcast %struct.fdata2* %3 to <2 x float>*
  store <2 x float> %0, <2 x float>* %4, align 4
  %5 = getelementptr inbounds %struct.fdata2, %struct.fdata2* %3, i32 0, i32 0
  %6 = load float, float* %5, align 4
  %7 = fpext float %6 to double
  %8 = getelementptr inbounds %struct.fdata2, %struct.fdata2* %3, i32 0, i32 1
  %9 = load float, float* %8, align 4
  %10 = fpext float %9 to double
  %11 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([12 x i8], [12 x i8]* @.str.18, i64 0, i64 0), double noundef %7, double noundef %10)
  %12 = bitcast %struct.fdata2* %2 to i8*
  %13 = bitcast %struct.fdata2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %12, i8* align 4 %13, i64 8, i1 false)
  %14 = bitcast %struct.fdata2* %2 to <2 x float>*
  %15 = load <2 x float>, <2 x float>* %14, align 4
  ret <2 x float> %15
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { <2 x float>, float } @ff3(<2 x float> %0, float %1) #3 {
  %3 = alloca %struct.fdata3, align 4
  %4 = alloca %struct.fdata3, align 4
  %5 = alloca { <2 x float>, float }, align 4
  %6 = alloca { <2 x float>, float }, align 8
  %7 = getelementptr inbounds { <2 x float>, float }, { <2 x float>, float }* %5, i32 0, i32 0
  store <2 x float> %0, <2 x float>* %7, align 4
  %8 = getelementptr inbounds { <2 x float>, float }, { <2 x float>, float }* %5, i32 0, i32 1
  store float %1, float* %8, align 4
  %9 = bitcast %struct.fdata3* %4 to i8*
  %10 = bitcast { <2 x float>, float }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 4 %10, i64 12, i1 false)
  %11 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %4, i32 0, i32 0
  %12 = load float, float* %11, align 4
  %13 = fpext float %12 to double
  %14 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %4, i32 0, i32 1
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %4, i32 0, i32 2
  %18 = load float, float* %17, align 4
  %19 = fpext float %18 to double
  %20 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.19, i64 0, i64 0), double noundef %13, double noundef %16, double noundef %19)
  %21 = bitcast %struct.fdata3* %3 to i8*
  %22 = bitcast %struct.fdata3* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %21, i8* align 4 %22, i64 12, i1 false)
  %23 = bitcast { <2 x float>, float }* %6 to i8*
  %24 = bitcast %struct.fdata3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %23, i8* align 4 %24, i64 12, i1 false)
  %25 = load { <2 x float>, float }, { <2 x float>, float }* %6, align 8
  ret { <2 x float>, float } %25
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { <2 x float>, <2 x float> } @ff4(<2 x float> %0, <2 x float> %1) #3 {
  %3 = alloca %struct.fdata4, align 4
  %4 = alloca %struct.fdata4, align 4
  %5 = bitcast %struct.fdata4* %4 to { <2 x float>, <2 x float> }*
  %6 = getelementptr inbounds { <2 x float>, <2 x float> }, { <2 x float>, <2 x float> }* %5, i32 0, i32 0
  store <2 x float> %0, <2 x float>* %6, align 4
  %7 = getelementptr inbounds { <2 x float>, <2 x float> }, { <2 x float>, <2 x float> }* %5, i32 0, i32 1
  store <2 x float> %1, <2 x float>* %7, align 4
  %8 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %4, i32 0, i32 0
  %9 = load float, float* %8, align 4
  %10 = fpext float %9 to double
  %11 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %4, i32 0, i32 1
  %12 = load float, float* %11, align 4
  %13 = fpext float %12 to double
  %14 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %4, i32 0, i32 2
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %4, i32 0, i32 3
  %18 = load float, float* %17, align 4
  %19 = fpext float %18 to double
  %20 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.20, i64 0, i64 0), double noundef %10, double noundef %13, double noundef %16, double noundef %19)
  %21 = bitcast %struct.fdata4* %3 to i8*
  %22 = bitcast %struct.fdata4* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %21, i8* align 4 %22, i64 16, i1 false)
  %23 = bitcast %struct.fdata4* %3 to { <2 x float>, <2 x float> }*
  %24 = load { <2 x float>, <2 x float> }, { <2 x float>, <2 x float> }* %23, align 4
  ret { <2 x float>, <2 x float> } %24
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff5(%struct.fdata5* noalias sret(%struct.fdata5) align 4 %0, %struct.fdata5* noundef byval(%struct.fdata5) align 8 %1) #0 {
  %3 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 0
  %4 = load float, float* %3, align 8
  %5 = fpext float %4 to double
  %6 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 1
  %7 = load float, float* %6, align 4
  %8 = fpext float %7 to double
  %9 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 2
  %10 = load float, float* %9, align 8
  %11 = fpext float %10 to double
  %12 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 3
  %13 = load float, float* %12, align 4
  %14 = fpext float %13 to double
  %15 = getelementptr inbounds %struct.fdata5, %struct.fdata5* %1, i32 0, i32 4
  %16 = load float, float* %15, align 8
  %17 = fpext float %16 to double
  %18 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([21 x i8], [21 x i8]* @.str.21, i64 0, i64 0), double noundef %5, double noundef %8, double noundef %11, double noundef %14, double noundef %17)
  %19 = bitcast %struct.fdata5* %0 to i8*
  %20 = bitcast %struct.fdata5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %19, i8* align 8 %20, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, double } @ff2id(i64 %0, double %1) #0 {
  %3 = alloca %struct.fdata2id, align 8
  %4 = alloca %struct.fdata2id, align 8
  %5 = bitcast %struct.fdata2id* %4 to { i64, double }*
  %6 = getelementptr inbounds { i64, double }, { i64, double }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 8
  %7 = getelementptr inbounds { i64, double }, { i64, double }* %5, i32 0, i32 1
  store double %1, double* %7, align 8
  %8 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %4, i32 0, i32 0
  %9 = load i8, i8* %8, align 8
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %4, i32 0, i32 1
  %12 = load i8, i8* %11, align 1
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %4, i32 0, i32 2
  %15 = load double, double* %14, align 8
  %16 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.22, i64 0, i64 0), i32 noundef %10, i32 noundef %13, double noundef %15)
  %17 = bitcast %struct.fdata2id* %3 to i8*
  %18 = bitcast %struct.fdata2id* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %17, i8* align 8 %18, i64 16, i1 false)
  %19 = bitcast %struct.fdata2id* %3 to { i64, double }*
  %20 = load { i64, double }, { i64, double }* %19, align 8
  ret { i64, double } %20
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, float } @ff7if(i64 %0, float %1) #0 {
  %3 = alloca %struct.fdata7if, align 4
  %4 = alloca %struct.fdata7if, align 4
  %5 = alloca { i64, float }, align 4
  %6 = alloca { i64, float }, align 8
  %7 = getelementptr inbounds { i64, float }, { i64, float }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 4
  %8 = getelementptr inbounds { i64, float }, { i64, float }* %5, i32 0, i32 1
  store float %1, float* %8, align 4
  %9 = bitcast %struct.fdata7if* %4 to i8*
  %10 = bitcast { i64, float }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 4 %10, i64 12, i1 false)
  %11 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %4, i32 0, i32 0
  %12 = getelementptr inbounds [7 x i8], [7 x i8]* %11, i64 0, i64 0
  %13 = load i8, i8* %12, align 4
  %14 = sext i8 %13 to i32
  %15 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %4, i32 0, i32 0
  %16 = getelementptr inbounds [7 x i8], [7 x i8]* %15, i64 0, i64 1
  %17 = load i8, i8* %16, align 1
  %18 = sext i8 %17 to i32
  %19 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %4, i32 0, i32 1
  %20 = load float, float* %19, align 4
  %21 = fpext float %20 to double
  %22 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([17 x i8], [17 x i8]* @.str.23, i64 0, i64 0), i32 noundef %14, i32 noundef %18, double noundef %21)
  %23 = bitcast %struct.fdata7if* %3 to i8*
  %24 = bitcast %struct.fdata7if* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %23, i8* align 4 %24, i64 12, i1 false)
  %25 = bitcast { i64, float }* %6 to i8*
  %26 = bitcast %struct.fdata7if* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %25, i8* align 4 %26, i64 12, i1 false)
  %27 = load { i64, float }, { i64, float }* %6, align 8
  ret { i64, float } %27
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, <2 x float> } @ff4if(i64 %0, <2 x float> %1) #3 {
  %3 = alloca %struct.fdata4if, align 4
  %4 = alloca %struct.fdata4if, align 4
  %5 = bitcast %struct.fdata4if* %4 to { i64, <2 x float> }*
  %6 = getelementptr inbounds { i64, <2 x float> }, { i64, <2 x float> }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 4
  %7 = getelementptr inbounds { i64, <2 x float> }, { i64, <2 x float> }* %5, i32 0, i32 1
  store <2 x float> %1, <2 x float>* %7, align 4
  %8 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %4, i32 0, i32 0
  %9 = load float, float* %8, align 4
  %10 = fpext float %9 to double
  %11 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %4, i32 0, i32 1
  %12 = load i8, i8* %11, align 4
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %4, i32 0, i32 2
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %4, i32 0, i32 3
  %18 = load float, float* %17, align 4
  %19 = fpext float %18 to double
  %20 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([20 x i8], [20 x i8]* @.str.24, i64 0, i64 0), double noundef %10, i32 noundef %13, double noundef %16, double noundef %19)
  %21 = bitcast %struct.fdata4if* %3 to i8*
  %22 = bitcast %struct.fdata4if* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %21, i8* align 4 %22, i64 16, i1 false)
  %23 = bitcast %struct.fdata4if* %3 to { i64, <2 x float> }*
  %24 = load { i64, <2 x float> }, { i64, <2 x float> }* %23, align 4
  ret { i64, <2 x float> } %24
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo(%struct.array* noalias sret(%struct.array) align 4 %0, %struct.array* noundef byval(%struct.array) align 8 %1) #0 {
  %3 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %4 = getelementptr inbounds [8 x i32], [8 x i32]* %3, i64 0, i64 0
  %5 = load i32, i32* %4, align 8
  %6 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %7 = getelementptr inbounds [8 x i32], [8 x i32]* %6, i64 0, i64 1
  %8 = load i32, i32* %7, align 4
  %9 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %10 = getelementptr inbounds [8 x i32], [8 x i32]* %9, i64 0, i64 2
  %11 = load i32, i32* %10, align 8
  %12 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.25, i64 0, i64 0), i32 noundef %5, i32 noundef %8, i32 noundef %11)
  %13 = bitcast %struct.array* %0 to i8*
  %14 = bitcast %struct.array* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %13, i8* align 8 %14, i64 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo2(%struct.array* noalias sret(%struct.array) align 4 %0, i32 noundef %1) #0 {
  %3 = alloca i32, align 4
  %4 = alloca i32, align 4
  store i32 %1, i32* %3, align 4
  store i32 0, i32* %4, align 4
  br label %5

5:                                                ; preds = %16, %2
  %6 = load i32, i32* %4, align 4
  %7 = icmp slt i32 %6, 8
  br i1 %7, label %8, label %19

8:                                                ; preds = %5
  %9 = load i32, i32* %4, align 4
  %10 = load i32, i32* %3, align 4
  %11 = add nsw i32 %9, %10
  %12 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %13 = load i32, i32* %4, align 4
  %14 = sext i32 %13 to i64
  %15 = getelementptr inbounds [8 x i32], [8 x i32]* %12, i64 0, i64 %14
  store i32 %11, i32* %15, align 4
  br label %16

16:                                               ; preds = %8
  %17 = load i32, i32* %4, align 4
  %18 = add nsw i32 %17, 1
  store i32 %18, i32* %4, align 4
  br label %5, !llvm.loop !6

19:                                               ; preds = %5
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @callback(void (%struct.array*, %struct.array*, i64, i64, i32)* noundef %0, %struct.array* noundef byval(%struct.array) align 8 %1) #0 {
  %3 = alloca void (%struct.array*, %struct.array*, i64, i64, i32)*, align 8
  %4 = alloca %struct.array, align 4
  %5 = alloca %struct.point, align 4
  %6 = alloca %struct.point1, align 4
  %7 = alloca %struct.array, align 8
  %8 = alloca { i64, i32 }, align 4
  %9 = alloca %struct.array, align 4
  store void (%struct.array*, %struct.array*, i64, i64, i32)* %0, void (%struct.array*, %struct.array*, i64, i64, i32)** %3, align 8
  call void @demo(%struct.array* sret(%struct.array) align 4 %4, %struct.array* noundef byval(%struct.array) align 8 %1)
  %10 = bitcast %struct.point* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %10, i8* align 4 bitcast (%struct.point* @__const.callback.pt to i8*), i64 8, i1 false)
  %11 = bitcast %struct.point1* %6 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %11, i8* align 4 bitcast (%struct.point1* @__const.callback.pt1 to i8*), i64 12, i1 false)
  %12 = load void (%struct.array*, %struct.array*, i64, i64, i32)*, void (%struct.array*, %struct.array*, i64, i64, i32)** %3, align 8
  %13 = bitcast %struct.point* %5 to i64*
  %14 = load i64, i64* %13, align 4
  %15 = bitcast { i64, i32 }* %8 to i8*
  %16 = bitcast %struct.point1* %6 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %15, i8* align 4 %16, i64 12, i1 false)
  %17 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %8, i32 0, i32 0
  %18 = load i64, i64* %17, align 4
  %19 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %8, i32 0, i32 1
  %20 = load i32, i32* %19, align 4
  call void %12(%struct.array* sret(%struct.array) align 4 %7, %struct.array* noundef byval(%struct.array) align 8 %1, i64 %14, i64 %18, i32 %20)
  call void @demo(%struct.array* sret(%struct.array) align 4 %9, %struct.array* noundef byval(%struct.array) align 8 %7)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @callback1(i64 (%struct.array*, i64, i64, i32)* noundef %0, %struct.array* noundef byval(%struct.array) align 8 %1) #0 {
  %3 = alloca i64 (%struct.array*, i64, i64, i32)*, align 8
  %4 = alloca %struct.point, align 4
  %5 = alloca %struct.point1, align 4
  %6 = alloca %struct.point, align 4
  %7 = alloca { i64, i32 }, align 4
  store i64 (%struct.array*, i64, i64, i32)* %0, i64 (%struct.array*, i64, i64, i32)** %3, align 8
  %8 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %9 = getelementptr inbounds [8 x i32], [8 x i32]* %8, i64 0, i64 0
  %10 = load i32, i32* %9, align 8
  %11 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %12 = getelementptr inbounds [8 x i32], [8 x i32]* %11, i64 0, i64 1
  %13 = load i32, i32* %12, align 4
  %14 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %15 = getelementptr inbounds [8 x i32], [8 x i32]* %14, i64 0, i64 7
  %16 = load i32, i32* %15, align 4
  %17 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([27 x i8], [27 x i8]* @.str.26, i64 0, i64 0), i32 noundef %10, i32 noundef %13, i32 noundef %16)
  %18 = bitcast %struct.point* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %18, i8* align 4 bitcast (%struct.point* @__const.callback1.pt to i8*), i64 8, i1 false)
  %19 = bitcast %struct.point1* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %19, i8* align 4 bitcast (%struct.point1* @__const.callback1.pt1 to i8*), i64 12, i1 false)
  %20 = load i64 (%struct.array*, i64, i64, i32)*, i64 (%struct.array*, i64, i64, i32)** %3, align 8
  %21 = bitcast %struct.point* %4 to i64*
  %22 = load i64, i64* %21, align 4
  %23 = bitcast { i64, i32 }* %7 to i8*
  %24 = bitcast %struct.point1* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %23, i8* align 4 %24, i64 12, i1 false)
  %25 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %7, i32 0, i32 0
  %26 = load i64, i64* %25, align 4
  %27 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %7, i32 0, i32 1
  %28 = load i32, i32* %27, align 4
  %29 = call i64 %20(%struct.array* noundef byval(%struct.array) align 8 %1, i64 %22, i64 %26, i32 %28)
  %30 = bitcast %struct.point* %6 to i64*
  store i64 %29, i64* %30, align 4
  %31 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 0
  %32 = load i32, i32* %31, align 4
  %33 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 1
  %34 = load i32, i32* %33, align 4
  %35 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([22 x i8], [22 x i8]* @.str.27, i64 0, i64 0), i32 noundef %32, i32 noundef %34)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @mycallback(%struct.array* noundef byval(%struct.array) align 8 %0, i64 %1, i64 %2, i32 %3) #0 {
  %5 = alloca %struct.point, align 4
  %6 = alloca %struct.point, align 4
  %7 = alloca %struct.point1, align 4
  %8 = alloca { i64, i32 }, align 4
  %9 = bitcast %struct.point* %6 to i64*
  store i64 %1, i64* %9, align 4
  %10 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %8, i32 0, i32 0
  store i64 %2, i64* %10, align 4
  %11 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %8, i32 0, i32 1
  store i32 %3, i32* %11, align 4
  %12 = bitcast %struct.point1* %7 to i8*
  %13 = bitcast { i64, i32 }* %8 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %12, i8* align 4 %13, i64 12, i1 false)
  %14 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %15 = getelementptr inbounds [8 x i32], [8 x i32]* %14, i64 0, i64 0
  %16 = load i32, i32* %15, align 8
  %17 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %18 = getelementptr inbounds [8 x i32], [8 x i32]* %17, i64 0, i64 1
  %19 = load i32, i32* %18, align 4
  %20 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %21 = getelementptr inbounds [8 x i32], [8 x i32]* %20, i64 0, i64 7
  %22 = load i32, i32* %21, align 4
  %23 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([28 x i8], [28 x i8]* @.str.28, i64 0, i64 0), i32 noundef %16, i32 noundef %19, i32 noundef %22)
  %24 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 0
  %25 = load i32, i32* %24, align 4
  %26 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 1
  %27 = load i32, i32* %26, align 4
  %28 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([22 x i8], [22 x i8]* @.str.29, i64 0, i64 0), i32 noundef %25, i32 noundef %27)
  %29 = getelementptr inbounds %struct.point1, %struct.point1* %7, i32 0, i32 0
  %30 = load i32, i32* %29, align 4
  %31 = getelementptr inbounds %struct.point1, %struct.point1* %7, i32 0, i32 1
  %32 = load i32, i32* %31, align 4
  %33 = getelementptr inbounds %struct.point1, %struct.point1* %7, i32 0, i32 2
  %34 = load i32, i32* %33, align 4
  %35 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([26 x i8], [26 x i8]* @.str.30, i64 0, i64 0), i32 noundef %30, i32 noundef %32, i32 noundef %34)
  %36 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 0
  %37 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 0
  %38 = load i32, i32* %37, align 4
  %39 = getelementptr inbounds %struct.point1, %struct.point1* %7, i32 0, i32 0
  %40 = load i32, i32* %39, align 4
  %41 = add nsw i32 %38, %40
  store i32 %41, i32* %36, align 4
  %42 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 1
  %43 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 1
  %44 = load i32, i32* %43, align 4
  %45 = getelementptr inbounds %struct.point1, %struct.point1* %7, i32 0, i32 1
  %46 = load i32, i32* %45, align 4
  %47 = add nsw i32 %44, %46
  store i32 %47, i32* %42, align 4
  %48 = bitcast %struct.point* %5 to i64*
  %49 = load i64, i64* %48, align 4
  ret i64 %49
}

attributes #0 = { noinline nounwind optnone uwtable "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="x86-64" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #1 = { "frame-pointer"="all" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="x86-64" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #2 = { argmemonly nofree nounwind willreturn }
attributes #3 = { noinline nounwind optnone uwtable "frame-pointer"="all" "min-legal-vector-width"="64" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="x86-64" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }

!llvm.module.flags = !{!0, !1, !2, !3, !4}
!llvm.ident = !{!5}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 7, !"PIC Level", i32 2}
!2 = !{i32 7, !"PIE Level", i32 2}
!3 = !{i32 7, !"uwtable", i32 2}
!4 = !{i32 7, !"frame-pointer", i32 2}
!5 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
!6 = distinct !{!6, !7}
!7 = !{!"llvm.loop.mustprogress"}
