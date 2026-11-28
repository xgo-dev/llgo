; ModuleID = '../../wrap/array_int16.c'
source_filename = "../../wrap/array_int16.c"
target datalayout = "e-m:e-p270:32:32-p271:32:32-p272:64:64-i64:64-f80:128-n8:16:32:64-S128"
target triple = "amd64-unknown-linux-gnu"

%struct.array1 = type { [1 x i16] }
%struct.array2 = type { [2 x i16] }
%struct.array3 = type { [3 x i16] }
%struct.array4 = type { [4 x i16] }
%struct.array5 = type { [5 x i16] }
%struct.array6 = type { [6 x i16] }
%struct.array7 = type { [7 x i16] }
%struct.array8 = type { [8 x i16] }
%struct.array9 = type { [9 x i16] }
%struct.array10 = type { [10 x i16] }
%struct.array11 = type { [11 x i16] }
%struct.array12 = type { [12 x i16] }
%struct.array13 = type { [13 x i16] }
%struct.array14 = type { [14 x i16] }
%struct.array15 = type { [15 x i16] }
%struct.array16 = type { [16 x i16] }
%struct.array17 = type { [17 x i16] }
%struct.array18 = type { [18 x i16] }
%struct.array19 = type { [19 x i16] }
%struct.array20 = type { [20 x i16] }

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i16 @demo1(i16 %0) #0 {
  %2 = alloca %struct.array1, align 2
  %3 = alloca %struct.array1, align 2
  %4 = getelementptr inbounds %struct.array1, %struct.array1* %3, i32 0, i32 0
  %5 = bitcast [1 x i16]* %4 to i16*
  store i16 %0, i16* %5, align 2
  %6 = bitcast %struct.array1* %2 to i8*
  %7 = bitcast %struct.array1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 2 %7, i64 2, i1 false)
  %8 = getelementptr inbounds %struct.array1, %struct.array1* %2, i32 0, i32 0
  %9 = bitcast [1 x i16]* %8 to i16*
  %10 = load i16, i16* %9, align 2
  ret i16 %10
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo2(i32 %0) #0 {
  %2 = alloca %struct.array2, align 2
  %3 = alloca %struct.array2, align 2
  %4 = getelementptr inbounds %struct.array2, %struct.array2* %3, i32 0, i32 0
  %5 = bitcast [2 x i16]* %4 to i32*
  store i32 %0, i32* %5, align 2
  %6 = bitcast %struct.array2* %2 to i8*
  %7 = bitcast %struct.array2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 2 %7, i64 4, i1 false)
  %8 = getelementptr inbounds %struct.array2, %struct.array2* %2, i32 0, i32 0
  %9 = bitcast [2 x i16]* %8 to i32*
  %10 = load i32, i32* %9, align 2
  ret i32 %10
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i48 @demo3(i48 %0) #0 {
  %2 = alloca %struct.array3, align 2
  %3 = alloca %struct.array3, align 2
  %4 = alloca i48, align 8
  %5 = alloca i48, align 8
  %6 = getelementptr inbounds %struct.array3, %struct.array3* %3, i32 0, i32 0
  store i48 %0, i48* %4, align 8
  %7 = bitcast [3 x i16]* %6 to i8*
  %8 = bitcast i48* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %7, i8* align 8 %8, i64 6, i1 false)
  %9 = bitcast %struct.array3* %2 to i8*
  %10 = bitcast %struct.array3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %9, i8* align 2 %10, i64 6, i1 false)
  %11 = getelementptr inbounds %struct.array3, %struct.array3* %2, i32 0, i32 0
  %12 = bitcast i48* %5 to i8*
  %13 = bitcast [3 x i16]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 2 %13, i64 6, i1 false)
  %14 = load i48, i48* %5, align 8
  ret i48 %14
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @demo4(i64 %0) #0 {
  %2 = alloca %struct.array4, align 2
  %3 = alloca %struct.array4, align 2
  %4 = getelementptr inbounds %struct.array4, %struct.array4* %3, i32 0, i32 0
  %5 = bitcast [4 x i16]* %4 to i64*
  store i64 %0, i64* %5, align 2
  %6 = bitcast %struct.array4* %2 to i8*
  %7 = bitcast %struct.array4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 2 %7, i64 8, i1 false)
  %8 = getelementptr inbounds %struct.array4, %struct.array4* %2, i32 0, i32 0
  %9 = bitcast [4 x i16]* %8 to i64*
  %10 = load i64, i64* %9, align 2
  ret i64 %10
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i16 } @demo5(i64 %0, i16 %1) #0 {
  %3 = alloca %struct.array5, align 2
  %4 = alloca %struct.array5, align 2
  %5 = alloca { i64, i16 }, align 2
  %6 = alloca { i64, i16 }, align 8
  %7 = getelementptr inbounds { i64, i16 }, { i64, i16 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 2
  %8 = getelementptr inbounds { i64, i16 }, { i64, i16 }* %5, i32 0, i32 1
  store i16 %1, i16* %8, align 2
  %9 = bitcast %struct.array5* %4 to i8*
  %10 = bitcast { i64, i16 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %9, i8* align 2 %10, i64 10, i1 false)
  %11 = bitcast %struct.array5* %3 to i8*
  %12 = bitcast %struct.array5* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %11, i8* align 2 %12, i64 10, i1 false)
  %13 = getelementptr inbounds %struct.array5, %struct.array5* %3, i32 0, i32 0
  %14 = bitcast { i64, i16 }* %6 to i8*
  %15 = bitcast [5 x i16]* %13 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %14, i8* align 2 %15, i64 10, i1 false)
  %16 = load { i64, i16 }, { i64, i16 }* %6, align 8
  ret { i64, i16 } %16
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i32 } @demo6(i64 %0, i32 %1) #0 {
  %3 = alloca %struct.array6, align 2
  %4 = alloca %struct.array6, align 2
  %5 = alloca { i64, i32 }, align 2
  %6 = alloca { i64, i32 }, align 8
  %7 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 2
  %8 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 1
  store i32 %1, i32* %8, align 2
  %9 = bitcast %struct.array6* %4 to i8*
  %10 = bitcast { i64, i32 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %9, i8* align 2 %10, i64 12, i1 false)
  %11 = bitcast %struct.array6* %3 to i8*
  %12 = bitcast %struct.array6* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %11, i8* align 2 %12, i64 12, i1 false)
  %13 = getelementptr inbounds %struct.array6, %struct.array6* %3, i32 0, i32 0
  %14 = bitcast { i64, i32 }* %6 to i8*
  %15 = bitcast [6 x i16]* %13 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %14, i8* align 2 %15, i64 12, i1 false)
  %16 = load { i64, i32 }, { i64, i32 }* %6, align 8
  ret { i64, i32 } %16
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i48 } @demo7(i64 %0, i48 %1) #0 {
  %3 = alloca %struct.array7, align 2
  %4 = alloca %struct.array7, align 2
  %5 = alloca { i64, i48 }, align 2
  %6 = alloca { i64, i48 }, align 8
  %7 = getelementptr inbounds { i64, i48 }, { i64, i48 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 2
  %8 = getelementptr inbounds { i64, i48 }, { i64, i48 }* %5, i32 0, i32 1
  store i48 %1, i48* %8, align 2
  %9 = bitcast %struct.array7* %4 to i8*
  %10 = bitcast { i64, i48 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %9, i8* align 2 %10, i64 14, i1 false)
  %11 = bitcast %struct.array7* %3 to i8*
  %12 = bitcast %struct.array7* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %11, i8* align 2 %12, i64 14, i1 false)
  %13 = getelementptr inbounds %struct.array7, %struct.array7* %3, i32 0, i32 0
  %14 = bitcast { i64, i48 }* %6 to i8*
  %15 = bitcast [7 x i16]* %13 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %14, i8* align 2 %15, i64 14, i1 false)
  %16 = load { i64, i48 }, { i64, i48 }* %6, align 8
  ret { i64, i48 } %16
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i64 } @demo8(i64 %0, i64 %1) #0 {
  %3 = alloca %struct.array8, align 2
  %4 = alloca %struct.array8, align 2
  %5 = bitcast %struct.array8* %4 to { i64, i64 }*
  %6 = getelementptr inbounds { i64, i64 }, { i64, i64 }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 2
  %7 = getelementptr inbounds { i64, i64 }, { i64, i64 }* %5, i32 0, i32 1
  store i64 %1, i64* %7, align 2
  %8 = bitcast %struct.array8* %3 to i8*
  %9 = bitcast %struct.array8* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 16, i1 false)
  %10 = getelementptr inbounds %struct.array8, %struct.array8* %3, i32 0, i32 0
  %11 = bitcast [8 x i16]* %10 to { i64, i64 }*
  %12 = load { i64, i64 }, { i64, i64 }* %11, align 2
  ret { i64, i64 } %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo9(%struct.array9* noalias sret(%struct.array9) align 2 %0, %struct.array9* noundef byval(%struct.array9) align 8 %1) #0 {
  %3 = bitcast %struct.array9* %0 to i8*
  %4 = bitcast %struct.array9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo10(%struct.array10* noalias sret(%struct.array10) align 2 %0, %struct.array10* noundef byval(%struct.array10) align 8 %1) #0 {
  %3 = bitcast %struct.array10* %0 to i8*
  %4 = bitcast %struct.array10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo11(%struct.array11* noalias sret(%struct.array11) align 2 %0, %struct.array11* noundef byval(%struct.array11) align 8 %1) #0 {
  %3 = bitcast %struct.array11* %0 to i8*
  %4 = bitcast %struct.array11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 22, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo12(%struct.array12* noalias sret(%struct.array12) align 2 %0, %struct.array12* noundef byval(%struct.array12) align 8 %1) #0 {
  %3 = bitcast %struct.array12* %0 to i8*
  %4 = bitcast %struct.array12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo13(%struct.array13* noalias sret(%struct.array13) align 2 %0, %struct.array13* noundef byval(%struct.array13) align 8 %1) #0 {
  %3 = bitcast %struct.array13* %0 to i8*
  %4 = bitcast %struct.array13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 26, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo14(%struct.array14* noalias sret(%struct.array14) align 2 %0, %struct.array14* noundef byval(%struct.array14) align 8 %1) #0 {
  %3 = bitcast %struct.array14* %0 to i8*
  %4 = bitcast %struct.array14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo15(%struct.array15* noalias sret(%struct.array15) align 2 %0, %struct.array15* noundef byval(%struct.array15) align 8 %1) #0 {
  %3 = bitcast %struct.array15* %0 to i8*
  %4 = bitcast %struct.array15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 30, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo16(%struct.array16* noalias sret(%struct.array16) align 2 %0, %struct.array16* noundef byval(%struct.array16) align 8 %1) #0 {
  %3 = bitcast %struct.array16* %0 to i8*
  %4 = bitcast %struct.array16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo17(%struct.array17* noalias sret(%struct.array17) align 2 %0, %struct.array17* noundef byval(%struct.array17) align 8 %1) #0 {
  %3 = bitcast %struct.array17* %0 to i8*
  %4 = bitcast %struct.array17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 34, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo18(%struct.array18* noalias sret(%struct.array18) align 2 %0, %struct.array18* noundef byval(%struct.array18) align 8 %1) #0 {
  %3 = bitcast %struct.array18* %0 to i8*
  %4 = bitcast %struct.array18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo19(%struct.array19* noalias sret(%struct.array19) align 2 %0, %struct.array19* noundef byval(%struct.array19) align 8 %1) #0 {
  %3 = bitcast %struct.array19* %0 to i8*
  %4 = bitcast %struct.array19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 38, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo20(%struct.array20* noalias sret(%struct.array20) align 2 %0, %struct.array20* noundef byval(%struct.array20) align 8 %1) #0 {
  %3 = bitcast %struct.array20* %0 to i8*
  %4 = bitcast %struct.array20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 40, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone uwtable "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="x86-64" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3, !4}
!llvm.ident = !{!5}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 7, !"PIC Level", i32 2}
!2 = !{i32 7, !"PIE Level", i32 2}
!3 = !{i32 7, !"uwtable", i32 2}
!4 = !{i32 7, !"frame-pointer", i32 2}
!5 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
