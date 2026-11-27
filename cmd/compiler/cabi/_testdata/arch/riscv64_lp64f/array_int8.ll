; ModuleID = '../../wrap/array_int8.c'
source_filename = "../../wrap/array_int8.c"
target datalayout = "e-m:e-p:64:64-i64:64-i128:128-n64-S128"
target triple = "riscv64-unknown-unknown-elf"

%struct.array1 = type { [1 x i8] }
%struct.array2 = type { [2 x i8] }
%struct.array3 = type { [3 x i8] }
%struct.array4 = type { [4 x i8] }
%struct.array5 = type { [5 x i8] }
%struct.array6 = type { [6 x i8] }
%struct.array7 = type { [7 x i8] }
%struct.array8 = type { [8 x i8] }
%struct.array9 = type { [9 x i8] }
%struct.array10 = type { [10 x i8] }
%struct.array11 = type { [11 x i8] }
%struct.array12 = type { [12 x i8] }
%struct.array13 = type { [13 x i8] }
%struct.array14 = type { [14 x i8] }
%struct.array15 = type { [15 x i8] }
%struct.array16 = type { [16 x i8] }
%struct.array17 = type { [17 x i8] }
%struct.array18 = type { [18 x i8] }
%struct.array19 = type { [19 x i8] }
%struct.array20 = type { [20 x i8] }

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo1(i64 %0) #0 {
  %2 = alloca %struct.array1, align 1
  %3 = alloca %struct.array1, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  %6 = getelementptr inbounds %struct.array1, %struct.array1* %3, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast [1 x i8]* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 1, i1 false)
  %9 = bitcast %struct.array1* %2 to i8*
  %10 = bitcast %struct.array1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 1, i1 false)
  %11 = getelementptr inbounds %struct.array1, %struct.array1* %2, i32 0, i32 0
  %12 = bitcast i64* %5 to i8*
  %13 = bitcast [1 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 1, i1 false)
  %14 = load i64, i64* %5, align 8
  ret i64 %14
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo2(i64 %0) #0 {
  %2 = alloca %struct.array2, align 1
  %3 = alloca %struct.array2, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  %6 = getelementptr inbounds %struct.array2, %struct.array2* %3, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast [2 x i8]* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 2, i1 false)
  %9 = bitcast %struct.array2* %2 to i8*
  %10 = bitcast %struct.array2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 2, i1 false)
  %11 = getelementptr inbounds %struct.array2, %struct.array2* %2, i32 0, i32 0
  %12 = bitcast i64* %5 to i8*
  %13 = bitcast [2 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 2, i1 false)
  %14 = load i64, i64* %5, align 8
  ret i64 %14
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo3(i64 %0) #0 {
  %2 = alloca %struct.array3, align 1
  %3 = alloca %struct.array3, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  %6 = getelementptr inbounds %struct.array3, %struct.array3* %3, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast [3 x i8]* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 3, i1 false)
  %9 = bitcast %struct.array3* %2 to i8*
  %10 = bitcast %struct.array3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 3, i1 false)
  %11 = getelementptr inbounds %struct.array3, %struct.array3* %2, i32 0, i32 0
  %12 = bitcast i64* %5 to i8*
  %13 = bitcast [3 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 3, i1 false)
  %14 = load i64, i64* %5, align 8
  ret i64 %14
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo4(i64 %0) #0 {
  %2 = alloca %struct.array4, align 1
  %3 = alloca %struct.array4, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  %6 = getelementptr inbounds %struct.array4, %struct.array4* %3, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast [4 x i8]* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 4, i1 false)
  %9 = bitcast %struct.array4* %2 to i8*
  %10 = bitcast %struct.array4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 4, i1 false)
  %11 = getelementptr inbounds %struct.array4, %struct.array4* %2, i32 0, i32 0
  %12 = bitcast i64* %5 to i8*
  %13 = bitcast [4 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 4, i1 false)
  %14 = load i64, i64* %5, align 8
  ret i64 %14
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo5(i64 %0) #0 {
  %2 = alloca %struct.array5, align 1
  %3 = alloca %struct.array5, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  %6 = getelementptr inbounds %struct.array5, %struct.array5* %3, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast [5 x i8]* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 5, i1 false)
  %9 = bitcast %struct.array5* %2 to i8*
  %10 = bitcast %struct.array5* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 5, i1 false)
  %11 = getelementptr inbounds %struct.array5, %struct.array5* %2, i32 0, i32 0
  %12 = bitcast i64* %5 to i8*
  %13 = bitcast [5 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 5, i1 false)
  %14 = load i64, i64* %5, align 8
  ret i64 %14
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo6(i64 %0) #0 {
  %2 = alloca %struct.array6, align 1
  %3 = alloca %struct.array6, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  %6 = getelementptr inbounds %struct.array6, %struct.array6* %3, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast [6 x i8]* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 6, i1 false)
  %9 = bitcast %struct.array6* %2 to i8*
  %10 = bitcast %struct.array6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 6, i1 false)
  %11 = getelementptr inbounds %struct.array6, %struct.array6* %2, i32 0, i32 0
  %12 = bitcast i64* %5 to i8*
  %13 = bitcast [6 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 6, i1 false)
  %14 = load i64, i64* %5, align 8
  ret i64 %14
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo7(i64 %0) #0 {
  %2 = alloca %struct.array7, align 1
  %3 = alloca %struct.array7, align 1
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  %6 = getelementptr inbounds %struct.array7, %struct.array7* %3, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast [7 x i8]* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 7, i1 false)
  %9 = bitcast %struct.array7* %2 to i8*
  %10 = bitcast %struct.array7* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 7, i1 false)
  %11 = getelementptr inbounds %struct.array7, %struct.array7* %2, i32 0, i32 0
  %12 = bitcast i64* %5 to i8*
  %13 = bitcast [7 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 7, i1 false)
  %14 = load i64, i64* %5, align 8
  ret i64 %14
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo8(i64 %0) #0 {
  %2 = alloca %struct.array8, align 1
  %3 = alloca %struct.array8, align 1
  %4 = getelementptr inbounds %struct.array8, %struct.array8* %3, i32 0, i32 0
  %5 = bitcast [8 x i8]* %4 to i64*
  store i64 %0, i64* %5, align 1
  %6 = bitcast %struct.array8* %2 to i8*
  %7 = bitcast %struct.array8* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 1 %7, i64 8, i1 false)
  %8 = getelementptr inbounds %struct.array8, %struct.array8* %2, i32 0, i32 0
  %9 = bitcast [8 x i8]* %8 to i64*
  %10 = load i64, i64* %9, align 1
  ret i64 %10
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo9([2 x i64] %0) #0 {
  %2 = alloca %struct.array9, align 1
  %3 = alloca %struct.array9, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  %6 = getelementptr inbounds %struct.array9, %struct.array9* %3, i32 0, i32 0
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %7 = bitcast [9 x i8]* %6 to i8*
  %8 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 9, i1 false)
  %9 = bitcast %struct.array9* %2 to i8*
  %10 = bitcast %struct.array9* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 9, i1 false)
  %11 = getelementptr inbounds %struct.array9, %struct.array9* %2, i32 0, i32 0
  %12 = bitcast [2 x i64]* %5 to i8*
  %13 = bitcast [9 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 9, i1 false)
  %14 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo10([2 x i64] %0) #0 {
  %2 = alloca %struct.array10, align 1
  %3 = alloca %struct.array10, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  %6 = getelementptr inbounds %struct.array10, %struct.array10* %3, i32 0, i32 0
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %7 = bitcast [10 x i8]* %6 to i8*
  %8 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 10, i1 false)
  %9 = bitcast %struct.array10* %2 to i8*
  %10 = bitcast %struct.array10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 10, i1 false)
  %11 = getelementptr inbounds %struct.array10, %struct.array10* %2, i32 0, i32 0
  %12 = bitcast [2 x i64]* %5 to i8*
  %13 = bitcast [10 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 10, i1 false)
  %14 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo11([2 x i64] %0) #0 {
  %2 = alloca %struct.array11, align 1
  %3 = alloca %struct.array11, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  %6 = getelementptr inbounds %struct.array11, %struct.array11* %3, i32 0, i32 0
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %7 = bitcast [11 x i8]* %6 to i8*
  %8 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 11, i1 false)
  %9 = bitcast %struct.array11* %2 to i8*
  %10 = bitcast %struct.array11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 11, i1 false)
  %11 = getelementptr inbounds %struct.array11, %struct.array11* %2, i32 0, i32 0
  %12 = bitcast [2 x i64]* %5 to i8*
  %13 = bitcast [11 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 11, i1 false)
  %14 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo12([2 x i64] %0) #0 {
  %2 = alloca %struct.array12, align 1
  %3 = alloca %struct.array12, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  %6 = getelementptr inbounds %struct.array12, %struct.array12* %3, i32 0, i32 0
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %7 = bitcast [12 x i8]* %6 to i8*
  %8 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 12, i1 false)
  %9 = bitcast %struct.array12* %2 to i8*
  %10 = bitcast %struct.array12* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 12, i1 false)
  %11 = getelementptr inbounds %struct.array12, %struct.array12* %2, i32 0, i32 0
  %12 = bitcast [2 x i64]* %5 to i8*
  %13 = bitcast [12 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 12, i1 false)
  %14 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo13([2 x i64] %0) #0 {
  %2 = alloca %struct.array13, align 1
  %3 = alloca %struct.array13, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  %6 = getelementptr inbounds %struct.array13, %struct.array13* %3, i32 0, i32 0
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %7 = bitcast [13 x i8]* %6 to i8*
  %8 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 13, i1 false)
  %9 = bitcast %struct.array13* %2 to i8*
  %10 = bitcast %struct.array13* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 13, i1 false)
  %11 = getelementptr inbounds %struct.array13, %struct.array13* %2, i32 0, i32 0
  %12 = bitcast [2 x i64]* %5 to i8*
  %13 = bitcast [13 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 13, i1 false)
  %14 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo14([2 x i64] %0) #0 {
  %2 = alloca %struct.array14, align 1
  %3 = alloca %struct.array14, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  %6 = getelementptr inbounds %struct.array14, %struct.array14* %3, i32 0, i32 0
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %7 = bitcast [14 x i8]* %6 to i8*
  %8 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 14, i1 false)
  %9 = bitcast %struct.array14* %2 to i8*
  %10 = bitcast %struct.array14* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 14, i1 false)
  %11 = getelementptr inbounds %struct.array14, %struct.array14* %2, i32 0, i32 0
  %12 = bitcast [2 x i64]* %5 to i8*
  %13 = bitcast [14 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 14, i1 false)
  %14 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo15([2 x i64] %0) #0 {
  %2 = alloca %struct.array15, align 1
  %3 = alloca %struct.array15, align 1
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  %6 = getelementptr inbounds %struct.array15, %struct.array15* %3, i32 0, i32 0
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %7 = bitcast [15 x i8]* %6 to i8*
  %8 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %7, i8* align 8 %8, i64 15, i1 false)
  %9 = bitcast %struct.array15* %2 to i8*
  %10 = bitcast %struct.array15* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %9, i8* align 1 %10, i64 15, i1 false)
  %11 = getelementptr inbounds %struct.array15, %struct.array15* %2, i32 0, i32 0
  %12 = bitcast [2 x i64]* %5 to i8*
  %13 = bitcast [15 x i8]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 1 %13, i64 15, i1 false)
  %14 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo16([2 x i64] %0) #0 {
  %2 = alloca %struct.array16, align 1
  %3 = alloca %struct.array16, align 1
  %4 = getelementptr inbounds %struct.array16, %struct.array16* %3, i32 0, i32 0
  %5 = bitcast [16 x i8]* %4 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %5, align 1
  %6 = bitcast %struct.array16* %2 to i8*
  %7 = bitcast %struct.array16* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 1 %7, i64 16, i1 false)
  %8 = getelementptr inbounds %struct.array16, %struct.array16* %2, i32 0, i32 0
  %9 = bitcast [16 x i8]* %8 to [2 x i64]*
  %10 = load [2 x i64], [2 x i64]* %9, align 1
  ret [2 x i64] %10
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(%struct.array17* noalias sret(%struct.array17) align 1 %0, %struct.array17* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array17* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array17* %0 to i8*
  %6 = bitcast %struct.array17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 17, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(%struct.array18* noalias sret(%struct.array18) align 1 %0, %struct.array18* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array18* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array18* %0 to i8*
  %6 = bitcast %struct.array18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(%struct.array19* noalias sret(%struct.array19) align 1 %0, %struct.array19* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array19* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array19* %0 to i8*
  %6 = bitcast %struct.array19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 19, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(%struct.array20* noalias sret(%struct.array20) align 1 %0, %struct.array20* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array20* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array20* %0 to i8*
  %6 = bitcast %struct.array20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 20, i1 false)
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
