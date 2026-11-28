; ModuleID = '../../wrap/array_float32.c'
source_filename = "../../wrap/array_float32.c"
target datalayout = "e-m:e-p:64:64-i64:64-i128:128-n64-S128"
target triple = "riscv64-unknown-unknown-elf"

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

; Function Attrs: noinline nounwind optnone
define dso_local float @demo1(float %0) #0 {
  %2 = alloca %struct.array1, align 4
  %3 = alloca %struct.array1, align 4
  %4 = bitcast %struct.array1* %3 to { float }*
  %5 = getelementptr inbounds { float }, { float }* %4, i32 0, i32 0
  store float %0, float* %5, align 4
  %6 = bitcast %struct.array1* %2 to i8*
  %7 = bitcast %struct.array1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 4 %7, i64 4, i1 false)
  %8 = bitcast %struct.array1* %2 to { float }*
  %9 = getelementptr inbounds { float }, { float }* %8, i32 0, i32 0
  %10 = load float, float* %9, align 4
  ret float %10
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local { float, float } @demo2(float %0, float %1) #0 {
  %3 = alloca %struct.array2, align 4
  %4 = alloca %struct.array2, align 4
  %5 = bitcast %struct.array2* %4 to { float, float }*
  %6 = getelementptr inbounds { float, float }, { float, float }* %5, i32 0, i32 0
  store float %0, float* %6, align 4
  %7 = getelementptr inbounds { float, float }, { float, float }* %5, i32 0, i32 1
  store float %1, float* %7, align 4
  %8 = bitcast %struct.array2* %3 to i8*
  %9 = bitcast %struct.array2* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 8, i1 false)
  %10 = bitcast %struct.array2* %3 to { float, float }*
  %11 = getelementptr inbounds { float, float }, { float, float }* %10, i32 0, i32 0
  %12 = load float, float* %11, align 4
  %13 = getelementptr inbounds { float, float }, { float, float }* %10, i32 0, i32 1
  %14 = load float, float* %13, align 4
  %15 = insertvalue { float, float } undef, float %12, 0
  %16 = insertvalue { float, float } %15, float %14, 1
  ret { float, float } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo3([2 x i64] %0) #0 {
  %2 = alloca %struct.array3, align 4
  %3 = alloca %struct.array3, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  %6 = getelementptr inbounds %struct.array3, %struct.array3* %3, i32 0, i32 0
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %7 = bitcast [3 x float]* %6 to i8*
  %8 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %7, i8* align 8 %8, i64 12, i1 false)
  %9 = bitcast %struct.array3* %2 to i8*
  %10 = bitcast %struct.array3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %9, i8* align 4 %10, i64 12, i1 false)
  %11 = getelementptr inbounds %struct.array3, %struct.array3* %2, i32 0, i32 0
  %12 = bitcast [2 x i64]* %5 to i8*
  %13 = bitcast [3 x float]* %11 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %12, i8* align 4 %13, i64 12, i1 false)
  %14 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %14
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo4([2 x i64] %0) #0 {
  %2 = alloca %struct.array4, align 4
  %3 = alloca %struct.array4, align 4
  %4 = getelementptr inbounds %struct.array4, %struct.array4* %3, i32 0, i32 0
  %5 = bitcast [4 x float]* %4 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %5, align 4
  %6 = bitcast %struct.array4* %2 to i8*
  %7 = bitcast %struct.array4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 4 %7, i64 16, i1 false)
  %8 = getelementptr inbounds %struct.array4, %struct.array4* %2, i32 0, i32 0
  %9 = bitcast [4 x float]* %8 to [2 x i64]*
  %10 = load [2 x i64], [2 x i64]* %9, align 4
  ret [2 x i64] %10
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo5(%struct.array5* noalias sret(%struct.array5) align 4 %0, %struct.array5* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array5* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array5* %0 to i8*
  %6 = bitcast %struct.array5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo6(%struct.array6* noalias sret(%struct.array6) align 4 %0, %struct.array6* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array6* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array6* %0 to i8*
  %6 = bitcast %struct.array6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(%struct.array7* noalias sret(%struct.array7) align 4 %0, %struct.array7* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array7* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array7* %0 to i8*
  %6 = bitcast %struct.array7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo8(%struct.array8* noalias sret(%struct.array8) align 4 %0, %struct.array8* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array8* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array8* %0 to i8*
  %6 = bitcast %struct.array8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(%struct.array9* noalias sret(%struct.array9) align 4 %0, %struct.array9* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array9* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array9* %0 to i8*
  %6 = bitcast %struct.array9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(%struct.array10* noalias sret(%struct.array10) align 4 %0, %struct.array10* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array10* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array10* %0 to i8*
  %6 = bitcast %struct.array10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(%struct.array11* noalias sret(%struct.array11) align 4 %0, %struct.array11* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array11* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array11* %0 to i8*
  %6 = bitcast %struct.array11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 44, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(%struct.array12* noalias sret(%struct.array12) align 4 %0, %struct.array12* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array12* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array12* %0 to i8*
  %6 = bitcast %struct.array12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(%struct.array13* noalias sret(%struct.array13) align 4 %0, %struct.array13* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array13* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array13* %0 to i8*
  %6 = bitcast %struct.array13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 52, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(%struct.array14* noalias sret(%struct.array14) align 4 %0, %struct.array14* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array14* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array14* %0 to i8*
  %6 = bitcast %struct.array14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(%struct.array15* noalias sret(%struct.array15) align 4 %0, %struct.array15* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array15* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array15* %0 to i8*
  %6 = bitcast %struct.array15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 60, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(%struct.array16* noalias sret(%struct.array16) align 4 %0, %struct.array16* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array16* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array16* %0 to i8*
  %6 = bitcast %struct.array16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(%struct.array17* noalias sret(%struct.array17) align 4 %0, %struct.array17* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array17* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array17* %0 to i8*
  %6 = bitcast %struct.array17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 68, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(%struct.array18* noalias sret(%struct.array18) align 4 %0, %struct.array18* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array18* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array18* %0 to i8*
  %6 = bitcast %struct.array18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(%struct.array19* noalias sret(%struct.array19) align 4 %0, %struct.array19* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array19* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array19* %0 to i8*
  %6 = bitcast %struct.array19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 76, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(%struct.array20* noalias sret(%struct.array20) align 4 %0, %struct.array20* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.array20* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.array20* %0 to i8*
  %6 = bitcast %struct.array20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 80, i1 false)
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
