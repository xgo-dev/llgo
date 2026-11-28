; ModuleID = 'array_int8.c'
source_filename = "array_int8.c"
target datalayout = "e-m:w-p270:32:32-p271:32:32-p272:64:64-i64:64-f80:128-n8:16:32:64-S128"
target triple = "x86_64-pc-windows-gnu"

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

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i8 @demo1(i8 %0) #0 {
  %2 = alloca %struct.array1, align 1
  %3 = alloca %struct.array1, align 1
  %4 = getelementptr inbounds %struct.array1, %struct.array1* %3, i32 0, i32 0
  %5 = bitcast [1 x i8]* %4 to i8*
  store i8 %0, i8* %5, align 1
  %6 = bitcast %struct.array1* %2 to i8*
  %7 = bitcast %struct.array1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 1 %7, i64 1, i1 false)
  %8 = getelementptr inbounds %struct.array1, %struct.array1* %2, i32 0, i32 0
  %9 = bitcast [1 x i8]* %8 to i8*
  %10 = load i8, i8* %9, align 1
  ret i8 %10
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i16 @demo2(i16 %0) #0 {
  %2 = alloca %struct.array2, align 1
  %3 = alloca %struct.array2, align 1
  %4 = getelementptr inbounds %struct.array2, %struct.array2* %3, i32 0, i32 0
  %5 = bitcast [2 x i8]* %4 to i16*
  store i16 %0, i16* %5, align 1
  %6 = bitcast %struct.array2* %2 to i8*
  %7 = bitcast %struct.array2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 1 %7, i64 2, i1 false)
  %8 = getelementptr inbounds %struct.array2, %struct.array2* %2, i32 0, i32 0
  %9 = bitcast [2 x i8]* %8 to i16*
  %10 = load i16, i16* %9, align 1
  ret i16 %10
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo3(%struct.array3* noalias sret(%struct.array3) align 1 %0, %struct.array3* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array3* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array3* %0 to i8*
  %6 = bitcast %struct.array3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 3, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo4(i32 %0) #0 {
  %2 = alloca %struct.array4, align 1
  %3 = alloca %struct.array4, align 1
  %4 = getelementptr inbounds %struct.array4, %struct.array4* %3, i32 0, i32 0
  %5 = bitcast [4 x i8]* %4 to i32*
  store i32 %0, i32* %5, align 1
  %6 = bitcast %struct.array4* %2 to i8*
  %7 = bitcast %struct.array4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %6, i8* align 1 %7, i64 4, i1 false)
  %8 = getelementptr inbounds %struct.array4, %struct.array4* %2, i32 0, i32 0
  %9 = bitcast [4 x i8]* %8 to i32*
  %10 = load i32, i32* %9, align 1
  ret i32 %10
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo5(%struct.array5* noalias sret(%struct.array5) align 1 %0, %struct.array5* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array5* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array5* %0 to i8*
  %6 = bitcast %struct.array5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 5, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo6(%struct.array6* noalias sret(%struct.array6) align 1 %0, %struct.array6* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array6* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array6* %0 to i8*
  %6 = bitcast %struct.array6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo7(%struct.array7* noalias sret(%struct.array7) align 1 %0, %struct.array7* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array7* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array7* %0 to i8*
  %6 = bitcast %struct.array7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 7, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
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

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo9(%struct.array9* noalias sret(%struct.array9) align 1 %0, %struct.array9* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array9* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array9* %0 to i8*
  %6 = bitcast %struct.array9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 9, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo10(%struct.array10* noalias sret(%struct.array10) align 1 %0, %struct.array10* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array10* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array10* %0 to i8*
  %6 = bitcast %struct.array10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo11(%struct.array11* noalias sret(%struct.array11) align 1 %0, %struct.array11* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array11* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array11* %0 to i8*
  %6 = bitcast %struct.array11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 11, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo12(%struct.array12* noalias sret(%struct.array12) align 1 %0, %struct.array12* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array12* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array12* %0 to i8*
  %6 = bitcast %struct.array12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo13(%struct.array13* noalias sret(%struct.array13) align 1 %0, %struct.array13* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array13* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array13* %0 to i8*
  %6 = bitcast %struct.array13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 13, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo14(%struct.array14* noalias sret(%struct.array14) align 1 %0, %struct.array14* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array14* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array14* %0 to i8*
  %6 = bitcast %struct.array14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 14, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo15(%struct.array15* noalias sret(%struct.array15) align 1 %0, %struct.array15* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array15* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array15* %0 to i8*
  %6 = bitcast %struct.array15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 15, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo16(%struct.array16* noalias sret(%struct.array16) align 1 %0, %struct.array16* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array16* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array16* %0 to i8*
  %6 = bitcast %struct.array16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo17(%struct.array17* noalias sret(%struct.array17) align 1 %0, %struct.array17* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array17* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array17* %0 to i8*
  %6 = bitcast %struct.array17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 17, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo18(%struct.array18* noalias sret(%struct.array18) align 1 %0, %struct.array18* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array18* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array18* %0 to i8*
  %6 = bitcast %struct.array18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo19(%struct.array19* noalias sret(%struct.array19) align 1 %0, %struct.array19* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array19* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array19* %0 to i8*
  %6 = bitcast %struct.array19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 19, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo20(%struct.array20* noalias sret(%struct.array20) align 1 %0, %struct.array20* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array20* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array20* %0 to i8*
  %6 = bitcast %struct.array20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 20, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone uwtable "frame-pointer"="none" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="x86-64" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2}
!llvm.ident = !{!3}

!0 = !{i32 1, !"wchar_size", i32 2}
!1 = !{i32 7, !"PIC Level", i32 2}
!2 = !{i32 7, !"uwtable", i32 2}
!3 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
