; ModuleID = '../../wrap/array_float32.c'
source_filename = "../../wrap/array_float32.c"
target datalayout = "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
target triple = "aarch64-unknown-linux-gnu"

%struct.array1 = type { [1 x float] }
%struct.array2 = type { [2 x float] }
%struct.array3 = type { [3 x float] }
%struct.array4 = type { [4 x float] }
%struct.array5 = type { [5 x float] }
%struct.array6 = type { [6 x float] }
%struct.array7 = type { [7 x float] }
%struct.array8 = type { [8 x float] }
%struct.array9 = type { [9 x float] }
%struct.array10 = type { [10 x float] }
%struct.array11 = type { [11 x float] }
%struct.array12 = type { [12 x float] }
%struct.array13 = type { [13 x float] }
%struct.array14 = type { [14 x float] }
%struct.array15 = type { [15 x float] }
%struct.array16 = type { [16 x float] }
%struct.array17 = type { [17 x float] }
%struct.array18 = type { [18 x float] }
%struct.array19 = type { [19 x float] }
%struct.array20 = type { [20 x float] }

; Function Attrs: noinline nounwind optnone uwtable
define dso_local %struct.array1 @demo1([1 x float] %0) #0 {
  %2 = alloca %struct.array1, align 4
  %3 = alloca %struct.array1, align 4
  %4 = getelementptr inbounds %struct.array1, %struct.array1* %3, i32 0, i32 0
  store [1 x float] %0, [1 x float]* %4, align 4
  %5 = bitcast %struct.array1* %2 to i8*
  %6 = bitcast %struct.array1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 4, i1 false)
  %7 = load %struct.array1, %struct.array1* %2, align 4
  ret %struct.array1 %7
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone uwtable
define dso_local %struct.array2 @demo2([2 x float] %0) #0 {
  %2 = alloca %struct.array2, align 4
  %3 = alloca %struct.array2, align 4
  %4 = getelementptr inbounds %struct.array2, %struct.array2* %3, i32 0, i32 0
  store [2 x float] %0, [2 x float]* %4, align 4
  %5 = bitcast %struct.array2* %2 to i8*
  %6 = bitcast %struct.array2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = load %struct.array2, %struct.array2* %2, align 4
  ret %struct.array2 %7
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local %struct.array3 @demo3([3 x float] %0) #0 {
  %2 = alloca %struct.array3, align 4
  %3 = alloca %struct.array3, align 4
  %4 = getelementptr inbounds %struct.array3, %struct.array3* %3, i32 0, i32 0
  store [3 x float] %0, [3 x float]* %4, align 4
  %5 = bitcast %struct.array3* %2 to i8*
  %6 = bitcast %struct.array3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 12, i1 false)
  %7 = load %struct.array3, %struct.array3* %2, align 4
  ret %struct.array3 %7
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local %struct.array4 @demo4([4 x float] %0) #0 {
  %2 = alloca %struct.array4, align 4
  %3 = alloca %struct.array4, align 4
  %4 = getelementptr inbounds %struct.array4, %struct.array4* %3, i32 0, i32 0
  store [4 x float] %0, [4 x float]* %4, align 4
  %5 = bitcast %struct.array4* %2 to i8*
  %6 = bitcast %struct.array4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = load %struct.array4, %struct.array4* %2, align 4
  ret %struct.array4 %7
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo5(%struct.array5* noalias sret(%struct.array5) align 4 %0, %struct.array5* noundef %1) #0 {
  %3 = bitcast %struct.array5* %0 to i8*
  %4 = bitcast %struct.array5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo6(%struct.array6* noalias sret(%struct.array6) align 4 %0, %struct.array6* noundef %1) #0 {
  %3 = bitcast %struct.array6* %0 to i8*
  %4 = bitcast %struct.array6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo7(%struct.array7* noalias sret(%struct.array7) align 4 %0, %struct.array7* noundef %1) #0 {
  %3 = bitcast %struct.array7* %0 to i8*
  %4 = bitcast %struct.array7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo8(%struct.array8* noalias sret(%struct.array8) align 4 %0, %struct.array8* noundef %1) #0 {
  %3 = bitcast %struct.array8* %0 to i8*
  %4 = bitcast %struct.array8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo9(%struct.array9* noalias sret(%struct.array9) align 4 %0, %struct.array9* noundef %1) #0 {
  %3 = bitcast %struct.array9* %0 to i8*
  %4 = bitcast %struct.array9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo10(%struct.array10* noalias sret(%struct.array10) align 4 %0, %struct.array10* noundef %1) #0 {
  %3 = bitcast %struct.array10* %0 to i8*
  %4 = bitcast %struct.array10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo11(%struct.array11* noalias sret(%struct.array11) align 4 %0, %struct.array11* noundef %1) #0 {
  %3 = bitcast %struct.array11* %0 to i8*
  %4 = bitcast %struct.array11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 44, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo12(%struct.array12* noalias sret(%struct.array12) align 4 %0, %struct.array12* noundef %1) #0 {
  %3 = bitcast %struct.array12* %0 to i8*
  %4 = bitcast %struct.array12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo13(%struct.array13* noalias sret(%struct.array13) align 4 %0, %struct.array13* noundef %1) #0 {
  %3 = bitcast %struct.array13* %0 to i8*
  %4 = bitcast %struct.array13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 52, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo14(%struct.array14* noalias sret(%struct.array14) align 4 %0, %struct.array14* noundef %1) #0 {
  %3 = bitcast %struct.array14* %0 to i8*
  %4 = bitcast %struct.array14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo15(%struct.array15* noalias sret(%struct.array15) align 4 %0, %struct.array15* noundef %1) #0 {
  %3 = bitcast %struct.array15* %0 to i8*
  %4 = bitcast %struct.array15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 60, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo16(%struct.array16* noalias sret(%struct.array16) align 4 %0, %struct.array16* noundef %1) #0 {
  %3 = bitcast %struct.array16* %0 to i8*
  %4 = bitcast %struct.array16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo17(%struct.array17* noalias sret(%struct.array17) align 4 %0, %struct.array17* noundef %1) #0 {
  %3 = bitcast %struct.array17* %0 to i8*
  %4 = bitcast %struct.array17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 68, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo18(%struct.array18* noalias sret(%struct.array18) align 4 %0, %struct.array18* noundef %1) #0 {
  %3 = bitcast %struct.array18* %0 to i8*
  %4 = bitcast %struct.array18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo19(%struct.array19* noalias sret(%struct.array19) align 4 %0, %struct.array19* noundef %1) #0 {
  %3 = bitcast %struct.array19* %0 to i8*
  %4 = bitcast %struct.array19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 76, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo20(%struct.array20* noalias sret(%struct.array20) align 4 %0, %struct.array20* noundef %1) #0 {
  %3 = bitcast %struct.array20* %0 to i8*
  %4 = bitcast %struct.array20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %3, i8* align 4 %4, i64 80, i1 false)
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
