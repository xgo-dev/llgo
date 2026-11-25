; ModuleID = '../../wrap/composite.c'
source_filename = "../../wrap/composite.c"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-unknown-linux-gnu"

%struct.array10 = type { %struct.basearray1 }
%struct.basearray1 = type { [1 x i32] }
%struct.array11 = type { %struct.basearray1, i32 }
%struct.point10 = type { %struct.basepoint1 }
%struct.basepoint1 = type { i32 }
%struct.point11 = type { %struct.basepoint1, i32 }
%struct.array20 = type { %struct.basearray2 }
%struct.basearray2 = type { [2 x i32] }
%struct.array21 = type { %struct.basearray2, i32 }
%struct.point20 = type { %struct.basepoint2 }
%struct.basepoint2 = type { i32, i32 }
%struct.point21 = type { %struct.basepoint2, i32 }

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo_array10(i64 %0) #0 {
  %2 = alloca %struct.array10, align 4
  %3 = alloca %struct.array10, align 4
  %4 = alloca i64, align 8
  %5 = getelementptr inbounds %struct.array10, %struct.array10* %3, i32 0, i32 0
  %6 = getelementptr inbounds %struct.basearray1, %struct.basearray1* %5, i32 0, i32 0
  store i64 %0, i64* %4, align 8
  %7 = bitcast [1 x i32]* %6 to i8*
  %8 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %7, i8* align 8 %8, i64 4, i1 false)
  %9 = bitcast %struct.array10* %2 to i8*
  %10 = bitcast %struct.array10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 4 %10, i64 4, i1 false)
  %11 = getelementptr inbounds %struct.array10, %struct.array10* %2, i32 0, i32 0
  %12 = getelementptr inbounds %struct.basearray1, %struct.basearray1* %11, i32 0, i32 0
  %13 = bitcast [1 x i32]* %12 to i32*
  %14 = load i32, i32* %13, align 4
  ret i32 %14
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @demo_array11(i64 %0) #0 {
  %2 = alloca %struct.array11, align 4
  %3 = alloca %struct.array11, align 4
  %4 = bitcast %struct.array11* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.array11* %2 to i8*
  %6 = bitcast %struct.array11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.array11* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo_point10(i64 %0) #0 {
  %2 = alloca %struct.point10, align 4
  %3 = alloca %struct.point10, align 4
  %4 = getelementptr inbounds %struct.point10, %struct.point10* %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basepoint1, %struct.basepoint1* %4, i32 0, i32 0
  %6 = trunc i64 %0 to i32
  store i32 %6, i32* %5, align 4
  %7 = bitcast %struct.point10* %2 to i8*
  %8 = bitcast %struct.point10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %7, i8* align 4 %8, i64 4, i1 false)
  %9 = getelementptr inbounds %struct.point10, %struct.point10* %2, i32 0, i32 0
  %10 = getelementptr inbounds %struct.basepoint1, %struct.basepoint1* %9, i32 0, i32 0
  %11 = load i32, i32* %10, align 4
  ret i32 %11
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @demo_point11(i64 %0) #0 {
  %2 = alloca %struct.point11, align 4
  %3 = alloca %struct.point11, align 4
  %4 = bitcast %struct.point11* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point11* %2 to i8*
  %6 = bitcast %struct.point11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point11* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @demo_array20(i64 %0) #0 {
  %2 = alloca %struct.array20, align 4
  %3 = alloca %struct.array20, align 4
  %4 = getelementptr inbounds %struct.array20, %struct.array20* %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basearray2, %struct.basearray2* %4, i32 0, i32 0
  %6 = bitcast [2 x i32]* %5 to i64*
  store i64 %0, i64* %6, align 4
  %7 = bitcast %struct.array20* %2 to i8*
  %8 = bitcast %struct.array20* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %7, i8* align 4 %8, i64 8, i1 false)
  %9 = getelementptr inbounds %struct.array20, %struct.array20* %2, i32 0, i32 0
  %10 = getelementptr inbounds %struct.basearray2, %struct.basearray2* %9, i32 0, i32 0
  %11 = bitcast [2 x i32]* %10 to i64*
  %12 = load i64, i64* %11, align 4
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo_array21([2 x i64] %0) #0 {
  %2 = alloca %struct.array21, align 4
  %3 = alloca %struct.array21, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.array21* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.array21* %2 to i8*
  %9 = bitcast %struct.array21* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.array21* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @demo_point20(i64 %0) #0 {
  %2 = alloca %struct.point20, align 4
  %3 = alloca %struct.point20, align 4
  %4 = getelementptr inbounds %struct.point20, %struct.point20* %3, i32 0, i32 0
  %5 = bitcast %struct.basepoint2* %4 to i64*
  store i64 %0, i64* %5, align 4
  %6 = bitcast %struct.point20* %2 to i8*
  %7 = bitcast %struct.point20* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 4 %7, i64 8, i1 false)
  %8 = getelementptr inbounds %struct.point20, %struct.point20* %2, i32 0, i32 0
  %9 = bitcast %struct.basepoint2* %8 to i64*
  %10 = load i64, i64* %9, align 4
  ret i64 %10
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local [2 x i64] @demo_point21([2 x i64] %0) #0 {
  %2 = alloca %struct.point21, align 4
  %3 = alloca %struct.point21, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point21* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point21* %2 to i8*
  %9 = bitcast %struct.point21* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point21* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
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
