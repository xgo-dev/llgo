; ModuleID = '../../wrap/demo.c'
source_filename = "../../wrap/demo.c"
target datalayout = "e-m:e-p:32:32-p270:32:32-p271:32:32-p272:64:64-f64:32:64-f80:32-n8:16:32-S128"
target triple = "i386-unknown-linux-gnu"

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
define dso_local void @pt64(%struct.point64* noalias sret(%struct.point64) align 4 %0, i64 %1, i64 %2) #0 {
  %4 = alloca i8*, align 4
  %5 = alloca %struct.point64, align 4
  %6 = bitcast %struct.point64* %0 to i8*
  store i8* %6, i8** %4, align 4
  %7 = getelementptr inbounds %struct.point64, %struct.point64* %5, i32 0, i32 0
  store i64 %1, i64* %7, align 4
  %8 = getelementptr inbounds %struct.point64, %struct.point64* %5, i32 0, i32 1
  store i64 %2, i64* %8, align 4
  %9 = getelementptr inbounds %struct.point64, %struct.point64* %5, i32 0, i32 0
  %10 = load i64, i64* %9, align 4
  %11 = getelementptr inbounds %struct.point64, %struct.point64* %5, i32 0, i32 1
  %12 = load i64, i64* %11, align 4
  %13 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([20 x i8], [20 x i8]* @.str, i32 0, i32 0), i64 noundef %10, i64 noundef %12)
  %14 = bitcast %struct.point64* %0 to i8*
  %15 = bitcast %struct.point64* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %14, i8* align 4 %15, i32 16, i1 false)
  ret void
}

declare i32 @printf(i8* noundef, ...) #1

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #2

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo32s(%struct.struct32* noalias sret(%struct.struct32) align 4 %0, i32 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = alloca %struct.struct32, align 4
  %5 = bitcast %struct.struct32* %0 to i8*
  store i8* %5, i8** %3, align 4
  %6 = getelementptr inbounds %struct.struct32, %struct.struct32* %4, i32 0, i32 0
  store i32 %1, i32* %6, align 4
  %7 = getelementptr inbounds %struct.struct32, %struct.struct32* %4, i32 0, i32 0
  %8 = load i32, i32* %7, align 4
  %9 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([14 x i8], [14 x i8]* @.str.1, i32 0, i32 0), i32 noundef %8)
  %10 = getelementptr inbounds %struct.struct32, %struct.struct32* %0, i32 0, i32 0
  %11 = getelementptr inbounds %struct.struct32, %struct.struct32* %4, i32 0, i32 0
  %12 = load i32, i32* %11, align 4
  %13 = add nsw i32 %12, 100
  store i32 %13, i32* %10, align 4
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @pt(%struct.point* noalias sret(%struct.point) align 4 %0, i32 %1, i32 %2) #0 {
  %4 = alloca i8*, align 4
  %5 = alloca %struct.point, align 4
  %6 = bitcast %struct.point* %0 to i8*
  store i8* %6, i8** %4, align 4
  %7 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 0
  store i32 %1, i32* %7, align 4
  %8 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 1
  store i32 %2, i32* %8, align 4
  %9 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 0
  %10 = load i32, i32* %9, align 4
  %11 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 1
  %12 = load i32, i32* %11, align 4
  %13 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([14 x i8], [14 x i8]* @.str.2, i32 0, i32 0), i32 noundef %10, i32 noundef %12)
  %14 = bitcast %struct.point* %0 to i8*
  %15 = bitcast %struct.point* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %14, i8* align 4 %15, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @pt1(%struct.point1* noalias sret(%struct.point1) align 4 %0, i32 %1, i32 %2, i32 %3) #0 {
  %5 = alloca i8*, align 4
  %6 = alloca %struct.point1, align 4
  %7 = bitcast %struct.point1* %0 to i8*
  store i8* %7, i8** %5, align 4
  %8 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 0
  store i32 %1, i32* %8, align 4
  %9 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 1
  store i32 %2, i32* %9, align 4
  %10 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 2
  store i32 %3, i32* %10, align 4
  %11 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 0
  %12 = load i32, i32* %11, align 4
  %13 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 1
  %14 = load i32, i32* %13, align 4
  %15 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 2
  %16 = load i32, i32* %15, align 4
  %17 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.3, i32 0, i32 0), i32 noundef %12, i32 noundef %14, i32 noundef %16)
  %18 = bitcast %struct.point1* %0 to i8*
  %19 = bitcast %struct.point1* %6 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %18, i8* align 4 %19, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @pt2(%struct.point2* noalias sret(%struct.point2) align 4 %0, %struct.point2* noundef byval(%struct.point2) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point2* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.point2, %struct.point2* %1, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point2, %struct.point2* %1, i32 0, i32 1
  %9 = load i32, i32* %8, align 4
  %10 = getelementptr inbounds %struct.point2, %struct.point2* %1, i32 0, i32 2
  %11 = load i32, i32* %10, align 4
  %12 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.4, i32 0, i32 0), i32 noundef %7, i32 noundef %9, i32 noundef %11)
  %13 = bitcast %struct.point2* %0 to i8*
  %14 = bitcast %struct.point2* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %13, i8* align 4 %14, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @pt3(%struct.point3* noalias sret(%struct.point3) align 1 %0, %struct.point3* noundef byval(%struct.point3) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point3* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.point3, %struct.point3* %1, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point3, %struct.point3* %1, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point3, %struct.point3* %1, i32 0, i32 2
  %12 = load i8, i8* %11, align 2
  %13 = sext i8 %12 to i32
  %14 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.5, i32 0, i32 0), i32 noundef %7, i32 noundef %10, i32 noundef %13)
  %15 = bitcast %struct.point3* %0 to i8*
  %16 = bitcast %struct.point3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %15, i8* align 4 %16, i32 3, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @pt4(%struct.point4* noalias sret(%struct.point4) align 4 %0, %struct.point4* noundef byval(%struct.point4) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point4* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.point4, %struct.point4* %1, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point4, %struct.point4* %1, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point4, %struct.point4* %1, i32 0, i32 2
  %12 = load i8, i8* %11, align 2
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point4, %struct.point4* %1, i32 0, i32 3
  %15 = load i32, i32* %14, align 4
  %16 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([21 x i8], [21 x i8]* @.str.6, i32 0, i32 0), i32 noundef %7, i32 noundef %10, i32 noundef %13, i32 noundef %15)
  %17 = bitcast %struct.point4* %0 to i8*
  %18 = bitcast %struct.point4* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %17, i8* align 4 %18, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @pt5(%struct.point5* noalias sret(%struct.point5) align 1 %0, %struct.point5* noundef byval(%struct.point5) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point5* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.point5, %struct.point5* %1, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point5, %struct.point5* %1, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point5, %struct.point5* %1, i32 0, i32 2
  %12 = load i8, i8* %11, align 2
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point5, %struct.point5* %1, i32 0, i32 3
  %15 = load i8, i8* %14, align 1
  %16 = sext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point5, %struct.point5* %1, i32 0, i32 4
  %18 = load i8, i8* %17, align 4
  %19 = sext i8 %18 to i32
  %20 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([24 x i8], [24 x i8]* @.str.7, i32 0, i32 0), i32 noundef %7, i32 noundef %10, i32 noundef %13, i32 noundef %16, i32 noundef %19)
  %21 = bitcast %struct.point5* %0 to i8*
  %22 = bitcast %struct.point5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %21, i8* align 4 %22, i32 5, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @pt6(%struct.point6* noalias sret(%struct.point6) align 4 %0, %struct.point6* noundef byval(%struct.point6) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point6* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.point6, %struct.point6* %1, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point6, %struct.point6* %1, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point6, %struct.point6* %1, i32 0, i32 2
  %12 = load i8, i8* %11, align 2
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point6, %struct.point6* %1, i32 0, i32 3
  %15 = load i8, i8* %14, align 1
  %16 = sext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point6, %struct.point6* %1, i32 0, i32 4
  %18 = load i8, i8* %17, align 4
  %19 = sext i8 %18 to i32
  %20 = getelementptr inbounds %struct.point6, %struct.point6* %1, i32 0, i32 5
  %21 = load i32, i32* %20, align 4
  %22 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([27 x i8], [27 x i8]* @.str.8, i32 0, i32 0), i32 noundef %7, i32 noundef %10, i32 noundef %13, i32 noundef %16, i32 noundef %19, i32 noundef %21)
  %23 = bitcast %struct.point6* %0 to i8*
  %24 = bitcast %struct.point6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %23, i8* align 4 %24, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @pt7(%struct.point7* noalias sret(%struct.point7) align 4 %0, %struct.point7* noundef byval(%struct.point7) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point7* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.point7, %struct.point7* %1, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.point7, %struct.point7* %1, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.point7, %struct.point7* %1, i32 0, i32 2
  %12 = load i8, i8* %11, align 2
  %13 = sext i8 %12 to i32
  %14 = getelementptr inbounds %struct.point7, %struct.point7* %1, i32 0, i32 3
  %15 = load i8, i8* %14, align 1
  %16 = sext i8 %15 to i32
  %17 = getelementptr inbounds %struct.point7, %struct.point7* %1, i32 0, i32 4
  %18 = load i8, i8* %17, align 4
  %19 = sext i8 %18 to i32
  %20 = getelementptr inbounds %struct.point7, %struct.point7* %1, i32 0, i32 5
  %21 = load i32, i32* %20, align 4
  %22 = getelementptr inbounds %struct.point7, %struct.point7* %1, i32 0, i32 6
  %23 = load i8, i8* %22, align 4
  %24 = sext i8 %23 to i32
  %25 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([30 x i8], [30 x i8]* @.str.9, i32 0, i32 0), i32 noundef %7, i32 noundef %10, i32 noundef %13, i32 noundef %16, i32 noundef %19, i32 noundef %21, i32 noundef %24)
  %26 = bitcast %struct.point7* %0 to i8*
  %27 = bitcast %struct.point7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %26, i8* align 4 %27, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @fn1(%struct.data1* noalias sret(%struct.data1) align 4 %0, %struct.data1* noundef byval(%struct.data1) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.data1* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.data1, %struct.data1* %1, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.data1, %struct.data1* %1, i32 0, i32 1
  %9 = load i64, i64* %8, align 4
  %10 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.10, i32 0, i32 0), i32 noundef %7, i64 noundef %9)
  %11 = bitcast %struct.data1* %0 to i8*
  %12 = bitcast %struct.data1* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %11, i8* align 4 %12, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @fn2(%struct.data2* noalias sret(%struct.data2) align 4 %0, i32 %1, i64 %2) #0 {
  %4 = alloca i8*, align 4
  %5 = alloca %struct.data2, align 4
  %6 = bitcast %struct.data2* %0 to i8*
  store i8* %6, i8** %4, align 4
  %7 = getelementptr inbounds %struct.data2, %struct.data2* %5, i32 0, i32 0
  store i32 %1, i32* %7, align 4
  %8 = getelementptr inbounds %struct.data2, %struct.data2* %5, i32 0, i32 1
  store i64 %2, i64* %8, align 4
  %9 = getelementptr inbounds %struct.data2, %struct.data2* %5, i32 0, i32 0
  %10 = load i32, i32* %9, align 4
  %11 = getelementptr inbounds %struct.data2, %struct.data2* %5, i32 0, i32 1
  %12 = load i64, i64* %11, align 4
  %13 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.11, i32 0, i32 0), i32 noundef %10, i64 noundef %12)
  %14 = bitcast %struct.data2* %0 to i8*
  %15 = bitcast %struct.data2* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %14, i8* align 4 %15, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @fn3(%struct.data3* noalias sret(%struct.data3) align 4 %0, %struct.data3* noundef byval(%struct.data3) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.data3* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.data3, %struct.data3* %1, i32 0, i32 0
  %6 = load i64, i64* %5, align 4
  %7 = getelementptr inbounds %struct.data3, %struct.data3* %1, i32 0, i32 1
  %8 = load i8, i8* %7, align 4
  %9 = sext i8 %8 to i32
  %10 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.12, i32 0, i32 0), i64 noundef %6, i32 noundef %9)
  %11 = bitcast %struct.data3* %0 to i8*
  %12 = bitcast %struct.data3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %11, i8* align 4 %12, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff1(%struct.fdata1* noalias sret(%struct.fdata1) align 4 %0, float %1) #0 {
  %3 = alloca i8*, align 4
  %4 = alloca %struct.fdata1, align 4
  %5 = bitcast %struct.fdata1* %0 to i8*
  store i8* %5, i8** %3, align 4
  %6 = getelementptr inbounds %struct.fdata1, %struct.fdata1* %4, i32 0, i32 0
  store float %1, float* %6, align 4
  %7 = getelementptr inbounds %struct.fdata1, %struct.fdata1* %4, i32 0, i32 0
  %8 = load float, float* %7, align 4
  %9 = fpext float %8 to double
  %10 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([9 x i8], [9 x i8]* @.str.13, i32 0, i32 0), double noundef %9)
  %11 = bitcast %struct.fdata1* %0 to i8*
  %12 = bitcast %struct.fdata1* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %11, i8* align 4 %12, i32 4, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @dd1(%struct.ddata1* noalias sret(%struct.ddata1) align 4 %0, double %1) #0 {
  %3 = alloca i8*, align 4
  %4 = alloca %struct.ddata1, align 4
  %5 = bitcast %struct.ddata1* %0 to i8*
  store i8* %5, i8** %3, align 4
  %6 = getelementptr inbounds %struct.ddata1, %struct.ddata1* %4, i32 0, i32 0
  store double %1, double* %6, align 4
  %7 = getelementptr inbounds %struct.ddata1, %struct.ddata1* %4, i32 0, i32 0
  %8 = load double, double* %7, align 4
  %9 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([9 x i8], [9 x i8]* @.str.14, i32 0, i32 0), double noundef %8)
  %10 = bitcast %struct.ddata1* %0 to i8*
  %11 = bitcast %struct.ddata1* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %10, i8* align 4 %11, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @dd2(%struct.ddata2* noalias sret(%struct.ddata2) align 4 %0, double %1, double %2) #0 {
  %4 = alloca i8*, align 4
  %5 = alloca %struct.ddata2, align 4
  %6 = bitcast %struct.ddata2* %0 to i8*
  store i8* %6, i8** %4, align 4
  %7 = getelementptr inbounds %struct.ddata2, %struct.ddata2* %5, i32 0, i32 0
  store double %1, double* %7, align 4
  %8 = getelementptr inbounds %struct.ddata2, %struct.ddata2* %5, i32 0, i32 1
  store double %2, double* %8, align 4
  %9 = getelementptr inbounds %struct.ddata2, %struct.ddata2* %5, i32 0, i32 0
  %10 = load double, double* %9, align 4
  %11 = getelementptr inbounds %struct.ddata2, %struct.ddata2* %5, i32 0, i32 1
  %12 = load double, double* %11, align 4
  %13 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([12 x i8], [12 x i8]* @.str.15, i32 0, i32 0), double noundef %10, double noundef %12)
  %14 = bitcast %struct.ddata2* %0 to i8*
  %15 = bitcast %struct.ddata2* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %14, i8* align 4 %15, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @dd3(%struct.ddata3* noalias sret(%struct.ddata3) align 4 %0, %struct.ddata3* noundef byval(%struct.ddata3) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.ddata3* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 0
  %6 = load double, double* %5, align 4
  %7 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 1
  %8 = load double, double* %7, align 4
  %9 = getelementptr inbounds %struct.ddata3, %struct.ddata3* %1, i32 0, i32 2
  %10 = load double, double* %9, align 4
  %11 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.16, i32 0, i32 0), double noundef %6, double noundef %8, double noundef %10)
  %12 = bitcast %struct.ddata3* %0 to i8*
  %13 = bitcast %struct.ddata3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %12, i8* align 4 %13, i32 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff2i(%struct.fdata2i* noalias sret(%struct.fdata2i) align 4 %0, float %1, i32 %2) #0 {
  %4 = alloca i8*, align 4
  %5 = alloca %struct.fdata2i, align 4
  %6 = bitcast %struct.fdata2i* %0 to i8*
  store i8* %6, i8** %4, align 4
  %7 = getelementptr inbounds %struct.fdata2i, %struct.fdata2i* %5, i32 0, i32 0
  store float %1, float* %7, align 4
  %8 = getelementptr inbounds %struct.fdata2i, %struct.fdata2i* %5, i32 0, i32 1
  store i32 %2, i32* %8, align 4
  %9 = getelementptr inbounds %struct.fdata2i, %struct.fdata2i* %5, i32 0, i32 0
  %10 = load float, float* %9, align 4
  %11 = fpext float %10 to double
  %12 = getelementptr inbounds %struct.fdata2i, %struct.fdata2i* %5, i32 0, i32 1
  %13 = load i32, i32* %12, align 4
  %14 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([13 x i8], [13 x i8]* @.str.17, i32 0, i32 0), double noundef %11, i32 noundef %13)
  %15 = bitcast %struct.fdata2i* %0 to i8*
  %16 = bitcast %struct.fdata2i* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %15, i8* align 4 %16, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff2(%struct.fdata2* noalias sret(%struct.fdata2) align 4 %0, float %1, float %2) #0 {
  %4 = alloca i8*, align 4
  %5 = alloca %struct.fdata2, align 4
  %6 = bitcast %struct.fdata2* %0 to i8*
  store i8* %6, i8** %4, align 4
  %7 = getelementptr inbounds %struct.fdata2, %struct.fdata2* %5, i32 0, i32 0
  store float %1, float* %7, align 4
  %8 = getelementptr inbounds %struct.fdata2, %struct.fdata2* %5, i32 0, i32 1
  store float %2, float* %8, align 4
  %9 = getelementptr inbounds %struct.fdata2, %struct.fdata2* %5, i32 0, i32 0
  %10 = load float, float* %9, align 4
  %11 = fpext float %10 to double
  %12 = getelementptr inbounds %struct.fdata2, %struct.fdata2* %5, i32 0, i32 1
  %13 = load float, float* %12, align 4
  %14 = fpext float %13 to double
  %15 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([12 x i8], [12 x i8]* @.str.18, i32 0, i32 0), double noundef %11, double noundef %14)
  %16 = bitcast %struct.fdata2* %0 to i8*
  %17 = bitcast %struct.fdata2* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %16, i8* align 4 %17, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff3(%struct.fdata3* noalias sret(%struct.fdata3) align 4 %0, float %1, float %2, float %3) #0 {
  %5 = alloca i8*, align 4
  %6 = alloca %struct.fdata3, align 4
  %7 = bitcast %struct.fdata3* %0 to i8*
  store i8* %7, i8** %5, align 4
  %8 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %6, i32 0, i32 0
  store float %1, float* %8, align 4
  %9 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %6, i32 0, i32 1
  store float %2, float* %9, align 4
  %10 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %6, i32 0, i32 2
  store float %3, float* %10, align 4
  %11 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %6, i32 0, i32 0
  %12 = load float, float* %11, align 4
  %13 = fpext float %12 to double
  %14 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %6, i32 0, i32 1
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = getelementptr inbounds %struct.fdata3, %struct.fdata3* %6, i32 0, i32 2
  %18 = load float, float* %17, align 4
  %19 = fpext float %18 to double
  %20 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.19, i32 0, i32 0), double noundef %13, double noundef %16, double noundef %19)
  %21 = bitcast %struct.fdata3* %0 to i8*
  %22 = bitcast %struct.fdata3* %6 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %21, i8* align 4 %22, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff4(%struct.fdata4* noalias sret(%struct.fdata4) align 4 %0, float %1, float %2, float %3, float %4) #0 {
  %6 = alloca i8*, align 4
  %7 = alloca %struct.fdata4, align 4
  %8 = bitcast %struct.fdata4* %0 to i8*
  store i8* %8, i8** %6, align 4
  %9 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %7, i32 0, i32 0
  store float %1, float* %9, align 4
  %10 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %7, i32 0, i32 1
  store float %2, float* %10, align 4
  %11 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %7, i32 0, i32 2
  store float %3, float* %11, align 4
  %12 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %7, i32 0, i32 3
  store float %4, float* %12, align 4
  %13 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %7, i32 0, i32 0
  %14 = load float, float* %13, align 4
  %15 = fpext float %14 to double
  %16 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %7, i32 0, i32 1
  %17 = load float, float* %16, align 4
  %18 = fpext float %17 to double
  %19 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %7, i32 0, i32 2
  %20 = load float, float* %19, align 4
  %21 = fpext float %20 to double
  %22 = getelementptr inbounds %struct.fdata4, %struct.fdata4* %7, i32 0, i32 3
  %23 = load float, float* %22, align 4
  %24 = fpext float %23 to double
  %25 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([18 x i8], [18 x i8]* @.str.20, i32 0, i32 0), double noundef %15, double noundef %18, double noundef %21, double noundef %24)
  %26 = bitcast %struct.fdata4* %0 to i8*
  %27 = bitcast %struct.fdata4* %7 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %26, i8* align 4 %27, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff5(%struct.fdata5* noalias sret(%struct.fdata5) align 4 %0, %struct.fdata5* noundef byval(%struct.fdata5) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.fdata5* %0 to i8*
  store i8* %4, i8** %3, align 4
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
  %20 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([21 x i8], [21 x i8]* @.str.21, i32 0, i32 0), double noundef %7, double noundef %10, double noundef %13, double noundef %16, double noundef %19)
  %21 = bitcast %struct.fdata5* %0 to i8*
  %22 = bitcast %struct.fdata5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %21, i8* align 4 %22, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff2id(%struct.fdata2id* noalias sret(%struct.fdata2id) align 4 %0, %struct.fdata2id* noundef byval(%struct.fdata2id) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.fdata2id* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %1, i32 0, i32 0
  %6 = load i8, i8* %5, align 4
  %7 = sext i8 %6 to i32
  %8 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %1, i32 0, i32 1
  %9 = load i8, i8* %8, align 1
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.fdata2id, %struct.fdata2id* %1, i32 0, i32 2
  %12 = load double, double* %11, align 4
  %13 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([15 x i8], [15 x i8]* @.str.22, i32 0, i32 0), i32 noundef %7, i32 noundef %10, double noundef %12)
  %14 = bitcast %struct.fdata2id* %0 to i8*
  %15 = bitcast %struct.fdata2id* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %14, i8* align 4 %15, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff7if(%struct.fdata7if* noalias sret(%struct.fdata7if) align 4 %0, %struct.fdata7if* noundef byval(%struct.fdata7if) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.fdata7if* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %1, i32 0, i32 0
  %6 = getelementptr inbounds [7 x i8], [7 x i8]* %5, i32 0, i32 0
  %7 = load i8, i8* %6, align 4
  %8 = sext i8 %7 to i32
  %9 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %1, i32 0, i32 0
  %10 = getelementptr inbounds [7 x i8], [7 x i8]* %9, i32 0, i32 1
  %11 = load i8, i8* %10, align 1
  %12 = sext i8 %11 to i32
  %13 = getelementptr inbounds %struct.fdata7if, %struct.fdata7if* %1, i32 0, i32 1
  %14 = load float, float* %13, align 4
  %15 = fpext float %14 to double
  %16 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([17 x i8], [17 x i8]* @.str.23, i32 0, i32 0), i32 noundef %8, i32 noundef %12, double noundef %15)
  %17 = bitcast %struct.fdata7if* %0 to i8*
  %18 = bitcast %struct.fdata7if* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %17, i8* align 4 %18, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @ff4if(%struct.fdata4if* noalias sret(%struct.fdata4if) align 4 %0, %struct.fdata4if* noundef byval(%struct.fdata4if) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.fdata4if* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %1, i32 0, i32 0
  %6 = load float, float* %5, align 4
  %7 = fpext float %6 to double
  %8 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %1, i32 0, i32 1
  %9 = load i8, i8* %8, align 4
  %10 = sext i8 %9 to i32
  %11 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %1, i32 0, i32 2
  %12 = load float, float* %11, align 4
  %13 = fpext float %12 to double
  %14 = getelementptr inbounds %struct.fdata4if, %struct.fdata4if* %1, i32 0, i32 3
  %15 = load float, float* %14, align 4
  %16 = fpext float %15 to double
  %17 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([20 x i8], [20 x i8]* @.str.24, i32 0, i32 0), double noundef %7, i32 noundef %10, double noundef %13, double noundef %16)
  %18 = bitcast %struct.fdata4if* %0 to i8*
  %19 = bitcast %struct.fdata4if* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %18, i8* align 4 %19, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo(%struct.array* noalias sret(%struct.array) align 4 %0, %struct.array* noundef byval(%struct.array) align 4 %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.array* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %6 = getelementptr inbounds [8 x i32], [8 x i32]* %5, i32 0, i32 0
  %7 = load i32, i32* %6, align 4
  %8 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %9 = getelementptr inbounds [8 x i32], [8 x i32]* %8, i32 0, i32 1
  %10 = load i32, i32* %9, align 4
  %11 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %12 = getelementptr inbounds [8 x i32], [8 x i32]* %11, i32 0, i32 2
  %13 = load i32, i32* %12, align 4
  %14 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([16 x i8], [16 x i8]* @.str.25, i32 0, i32 0), i32 noundef %7, i32 noundef %10, i32 noundef %13)
  %15 = bitcast %struct.array* %0 to i8*
  %16 = bitcast %struct.array* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %15, i8* align 4 %16, i32 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo2(%struct.array* noalias sret(%struct.array) align 4 %0, i32 noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = alloca i32, align 4
  %5 = alloca i32, align 4
  %6 = bitcast %struct.array* %0 to i8*
  store i8* %6, i8** %3, align 4
  store i32 %1, i32* %4, align 4
  store i32 0, i32* %5, align 4
  br label %7

7:                                                ; preds = %17, %2
  %8 = load i32, i32* %5, align 4
  %9 = icmp slt i32 %8, 8
  br i1 %9, label %10, label %20

10:                                               ; preds = %7
  %11 = load i32, i32* %5, align 4
  %12 = load i32, i32* %4, align 4
  %13 = add nsw i32 %11, %12
  %14 = getelementptr inbounds %struct.array, %struct.array* %0, i32 0, i32 0
  %15 = load i32, i32* %5, align 4
  %16 = getelementptr inbounds [8 x i32], [8 x i32]* %14, i32 0, i32 %15
  store i32 %13, i32* %16, align 4
  br label %17

17:                                               ; preds = %10
  %18 = load i32, i32* %5, align 4
  %19 = add nsw i32 %18, 1
  store i32 %19, i32* %5, align 4
  br label %7, !llvm.loop !7

20:                                               ; preds = %7
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @callback(void (%struct.array*, %struct.array*, i32, i32, i32, i32, i32)* noundef %0, %struct.array* noundef byval(%struct.array) align 4 %1) #0 {
  %3 = alloca void (%struct.array*, %struct.array*, i32, i32, i32, i32, i32)*, align 4
  %4 = alloca %struct.array, align 4
  %5 = alloca %struct.point, align 4
  %6 = alloca %struct.point1, align 4
  %7 = alloca %struct.array, align 4
  %8 = alloca %struct.array, align 4
  store void (%struct.array*, %struct.array*, i32, i32, i32, i32, i32)* %0, void (%struct.array*, %struct.array*, i32, i32, i32, i32, i32)** %3, align 4
  call void @demo(%struct.array* sret(%struct.array) align 4 %4, %struct.array* noundef byval(%struct.array) align 4 %1)
  %9 = bitcast %struct.point* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %9, i8* align 4 bitcast (%struct.point* @__const.callback.pt to i8*), i32 8, i1 false)
  %10 = bitcast %struct.point1* %6 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %10, i8* align 4 bitcast (%struct.point1* @__const.callback.pt1 to i8*), i32 12, i1 false)
  %11 = load void (%struct.array*, %struct.array*, i32, i32, i32, i32, i32)*, void (%struct.array*, %struct.array*, i32, i32, i32, i32, i32)** %3, align 4
  %12 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 0
  %13 = load i32, i32* %12, align 4
  %14 = getelementptr inbounds %struct.point, %struct.point* %5, i32 0, i32 1
  %15 = load i32, i32* %14, align 4
  %16 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 0
  %17 = load i32, i32* %16, align 4
  %18 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 1
  %19 = load i32, i32* %18, align 4
  %20 = getelementptr inbounds %struct.point1, %struct.point1* %6, i32 0, i32 2
  %21 = load i32, i32* %20, align 4
  call void %11(%struct.array* sret(%struct.array) align 4 %7, %struct.array* noundef byval(%struct.array) align 4 %1, i32 %13, i32 %15, i32 %17, i32 %19, i32 %21)
  call void @demo(%struct.array* sret(%struct.array) align 4 %8, %struct.array* noundef byval(%struct.array) align 4 %7)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @callback1(void (%struct.point*, %struct.array*, i32, i32, i32, i32, i32)* noundef %0, %struct.array* noundef byval(%struct.array) align 4 %1) #0 {
  %3 = alloca void (%struct.point*, %struct.array*, i32, i32, i32, i32, i32)*, align 4
  %4 = alloca %struct.point, align 4
  %5 = alloca %struct.point1, align 4
  %6 = alloca %struct.point, align 4
  store void (%struct.point*, %struct.array*, i32, i32, i32, i32, i32)* %0, void (%struct.point*, %struct.array*, i32, i32, i32, i32, i32)** %3, align 4
  %7 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %8 = getelementptr inbounds [8 x i32], [8 x i32]* %7, i32 0, i32 0
  %9 = load i32, i32* %8, align 4
  %10 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %11 = getelementptr inbounds [8 x i32], [8 x i32]* %10, i32 0, i32 1
  %12 = load i32, i32* %11, align 4
  %13 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %14 = getelementptr inbounds [8 x i32], [8 x i32]* %13, i32 0, i32 7
  %15 = load i32, i32* %14, align 4
  %16 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([27 x i8], [27 x i8]* @.str.26, i32 0, i32 0), i32 noundef %9, i32 noundef %12, i32 noundef %15)
  %17 = bitcast %struct.point* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %17, i8* align 4 bitcast (%struct.point* @__const.callback1.pt to i8*), i32 8, i1 false)
  %18 = bitcast %struct.point1* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %18, i8* align 4 bitcast (%struct.point1* @__const.callback1.pt1 to i8*), i32 12, i1 false)
  %19 = load void (%struct.point*, %struct.array*, i32, i32, i32, i32, i32)*, void (%struct.point*, %struct.array*, i32, i32, i32, i32, i32)** %3, align 4
  %20 = getelementptr inbounds %struct.point, %struct.point* %4, i32 0, i32 0
  %21 = load i32, i32* %20, align 4
  %22 = getelementptr inbounds %struct.point, %struct.point* %4, i32 0, i32 1
  %23 = load i32, i32* %22, align 4
  %24 = getelementptr inbounds %struct.point1, %struct.point1* %5, i32 0, i32 0
  %25 = load i32, i32* %24, align 4
  %26 = getelementptr inbounds %struct.point1, %struct.point1* %5, i32 0, i32 1
  %27 = load i32, i32* %26, align 4
  %28 = getelementptr inbounds %struct.point1, %struct.point1* %5, i32 0, i32 2
  %29 = load i32, i32* %28, align 4
  call void %19(%struct.point* sret(%struct.point) align 4 %6, %struct.array* noundef byval(%struct.array) align 4 %1, i32 %21, i32 %23, i32 %25, i32 %27, i32 %29)
  %30 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 0
  %31 = load i32, i32* %30, align 4
  %32 = getelementptr inbounds %struct.point, %struct.point* %6, i32 0, i32 1
  %33 = load i32, i32* %32, align 4
  %34 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([22 x i8], [22 x i8]* @.str.27, i32 0, i32 0), i32 noundef %31, i32 noundef %33)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @mycallback(%struct.point* noalias sret(%struct.point) align 4 %0, %struct.array* noundef byval(%struct.array) align 4 %1, i32 %2, i32 %3, i32 %4, i32 %5, i32 %6) #0 {
  %8 = alloca i8*, align 4
  %9 = alloca %struct.point, align 4
  %10 = alloca %struct.point1, align 4
  %11 = bitcast %struct.point* %0 to i8*
  store i8* %11, i8** %8, align 4
  %12 = getelementptr inbounds %struct.point, %struct.point* %9, i32 0, i32 0
  store i32 %2, i32* %12, align 4
  %13 = getelementptr inbounds %struct.point, %struct.point* %9, i32 0, i32 1
  store i32 %3, i32* %13, align 4
  %14 = getelementptr inbounds %struct.point1, %struct.point1* %10, i32 0, i32 0
  store i32 %4, i32* %14, align 4
  %15 = getelementptr inbounds %struct.point1, %struct.point1* %10, i32 0, i32 1
  store i32 %5, i32* %15, align 4
  %16 = getelementptr inbounds %struct.point1, %struct.point1* %10, i32 0, i32 2
  store i32 %6, i32* %16, align 4
  %17 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %18 = getelementptr inbounds [8 x i32], [8 x i32]* %17, i32 0, i32 0
  %19 = load i32, i32* %18, align 4
  %20 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %21 = getelementptr inbounds [8 x i32], [8 x i32]* %20, i32 0, i32 1
  %22 = load i32, i32* %21, align 4
  %23 = getelementptr inbounds %struct.array, %struct.array* %1, i32 0, i32 0
  %24 = getelementptr inbounds [8 x i32], [8 x i32]* %23, i32 0, i32 7
  %25 = load i32, i32* %24, align 4
  %26 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([28 x i8], [28 x i8]* @.str.28, i32 0, i32 0), i32 noundef %19, i32 noundef %22, i32 noundef %25)
  %27 = getelementptr inbounds %struct.point, %struct.point* %9, i32 0, i32 0
  %28 = load i32, i32* %27, align 4
  %29 = getelementptr inbounds %struct.point, %struct.point* %9, i32 0, i32 1
  %30 = load i32, i32* %29, align 4
  %31 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([22 x i8], [22 x i8]* @.str.29, i32 0, i32 0), i32 noundef %28, i32 noundef %30)
  %32 = getelementptr inbounds %struct.point1, %struct.point1* %10, i32 0, i32 0
  %33 = load i32, i32* %32, align 4
  %34 = getelementptr inbounds %struct.point1, %struct.point1* %10, i32 0, i32 1
  %35 = load i32, i32* %34, align 4
  %36 = getelementptr inbounds %struct.point1, %struct.point1* %10, i32 0, i32 2
  %37 = load i32, i32* %36, align 4
  %38 = call i32 (i8*, ...) @printf(i8* noundef getelementptr inbounds ([26 x i8], [26 x i8]* @.str.30, i32 0, i32 0), i32 noundef %33, i32 noundef %35, i32 noundef %37)
  %39 = getelementptr inbounds %struct.point, %struct.point* %0, i32 0, i32 0
  %40 = getelementptr inbounds %struct.point, %struct.point* %9, i32 0, i32 0
  %41 = load i32, i32* %40, align 4
  %42 = getelementptr inbounds %struct.point1, %struct.point1* %10, i32 0, i32 0
  %43 = load i32, i32* %42, align 4
  %44 = add nsw i32 %41, %43
  store i32 %44, i32* %39, align 4
  %45 = getelementptr inbounds %struct.point, %struct.point* %0, i32 0, i32 1
  %46 = getelementptr inbounds %struct.point, %struct.point* %9, i32 0, i32 1
  %47 = load i32, i32* %46, align 4
  %48 = getelementptr inbounds %struct.point1, %struct.point1* %10, i32 0, i32 1
  %49 = load i32, i32* %48, align 4
  %50 = add nsw i32 %47, %49
  store i32 %50, i32* %45, align 4
  ret void
}

attributes #0 = { noinline nounwind optnone uwtable "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="pentium4" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #1 = { "frame-pointer"="all" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="pentium4" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #2 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3, !4, !5}
!llvm.ident = !{!6}

!0 = !{i32 1, !"NumRegisterParameters", i32 0}
!1 = !{i32 1, !"wchar_size", i32 4}
!2 = !{i32 7, !"PIC Level", i32 2}
!3 = !{i32 7, !"PIE Level", i32 2}
!4 = !{i32 7, !"uwtable", i32 2}
!5 = !{i32 7, !"frame-pointer", i32 2}
!6 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
!7 = distinct !{!7, !8}
!8 = !{!"llvm.loop.mustprogress"}
