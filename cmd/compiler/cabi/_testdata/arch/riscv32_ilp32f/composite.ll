; ModuleID = '../../wrap/composite.c'
source_filename = "../../wrap/composite.c"
target datalayout = "e-m:e-p:32:32-i64:64-n32-S128"
target triple = "riscv32-unknown-unknown-elf"

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

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo_array10(i32 %0) #0 {
  %2 = alloca %struct.array10, align 4
  %3 = alloca %struct.array10, align 4
  %4 = getelementptr inbounds %struct.array10, %struct.array10* %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basearray1, %struct.basearray1* %4, i32 0, i32 0
  %6 = bitcast [1 x i32]* %5 to i32*
  store i32 %0, i32* %6, align 4
  %7 = bitcast %struct.array10* %2 to i8*
  %8 = bitcast %struct.array10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %7, i8* align 4 %8, i32 4, i1 false)
  %9 = getelementptr inbounds %struct.array10, %struct.array10* %2, i32 0, i32 0
  %10 = getelementptr inbounds %struct.basearray1, %struct.basearray1* %9, i32 0, i32 0
  %11 = bitcast [1 x i32]* %10 to i32*
  %12 = load i32, i32* %11, align 4
  ret i32 %12
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo_array11([2 x i32] %0) #0 {
  %2 = alloca %struct.array11, align 4
  %3 = alloca %struct.array11, align 4
  %4 = bitcast %struct.array11* %3 to [2 x i32]*
  store [2 x i32] %0, [2 x i32]* %4, align 4
  %5 = bitcast %struct.array11* %2 to i8*
  %6 = bitcast %struct.array11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 8, i1 false)
  %7 = bitcast %struct.array11* %2 to [2 x i32]*
  %8 = load [2 x i32], [2 x i32]* %7, align 4
  ret [2 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo_point10(i32 %0) #0 {
  %2 = alloca %struct.point10, align 4
  %3 = alloca %struct.point10, align 4
  %4 = getelementptr inbounds %struct.point10, %struct.point10* %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basepoint1, %struct.basepoint1* %4, i32 0, i32 0
  store i32 %0, i32* %5, align 4
  %6 = bitcast %struct.point10* %2 to i8*
  %7 = bitcast %struct.point10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %6, i8* align 4 %7, i32 4, i1 false)
  %8 = getelementptr inbounds %struct.point10, %struct.point10* %2, i32 0, i32 0
  %9 = getelementptr inbounds %struct.basepoint1, %struct.basepoint1* %8, i32 0, i32 0
  %10 = load i32, i32* %9, align 4
  ret i32 %10
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo_point11([2 x i32] %0) #0 {
  %2 = alloca %struct.point11, align 4
  %3 = alloca %struct.point11, align 4
  %4 = bitcast %struct.point11* %3 to [2 x i32]*
  store [2 x i32] %0, [2 x i32]* %4, align 4
  %5 = bitcast %struct.point11* %2 to i8*
  %6 = bitcast %struct.point11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 8, i1 false)
  %7 = bitcast %struct.point11* %2 to [2 x i32]*
  %8 = load [2 x i32], [2 x i32]* %7, align 4
  ret [2 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo_array20([2 x i32] %0) #0 {
  %2 = alloca %struct.array20, align 4
  %3 = alloca %struct.array20, align 4
  %4 = getelementptr inbounds %struct.array20, %struct.array20* %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basearray2, %struct.basearray2* %4, i32 0, i32 0
  store [2 x i32] %0, [2 x i32]* %5, align 4
  %6 = bitcast %struct.array20* %2 to i8*
  %7 = bitcast %struct.array20* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %6, i8* align 4 %7, i32 8, i1 false)
  %8 = getelementptr inbounds %struct.array20, %struct.array20* %2, i32 0, i32 0
  %9 = getelementptr inbounds %struct.basearray2, %struct.basearray2* %8, i32 0, i32 0
  %10 = load [2 x i32], [2 x i32]* %9, align 4
  ret [2 x i32] %10
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_array21(%struct.array21* noalias sret(%struct.array21) align 4 %0, %struct.array21* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.array21* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.array21* %0 to i8*
  %6 = bitcast %struct.array21* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo_point20([2 x i32] %0) #0 {
  %2 = alloca %struct.point20, align 4
  %3 = alloca %struct.point20, align 4
  %4 = getelementptr inbounds %struct.point20, %struct.point20* %3, i32 0, i32 0
  %5 = bitcast %struct.basepoint2* %4 to [2 x i32]*
  store [2 x i32] %0, [2 x i32]* %5, align 4
  %6 = bitcast %struct.point20* %2 to i8*
  %7 = bitcast %struct.point20* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %6, i8* align 4 %7, i32 8, i1 false)
  %8 = getelementptr inbounds %struct.point20, %struct.point20* %2, i32 0, i32 0
  %9 = bitcast %struct.basepoint2* %8 to [2 x i32]*
  %10 = load [2 x i32], [2 x i32]* %9, align 4
  ret [2 x i32] %10
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_point21(%struct.point21* noalias sret(%struct.point21) align 4 %0, %struct.point21* noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = bitcast %struct.point21* %0 to i8*
  store i8* %4, i8** %3, align 4
  %5 = bitcast %struct.point21* %0 to i8*
  %6 = bitcast %struct.point21* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 12, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+a,+c,+d,+f,+m,+relax,-save-restore" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3}
!llvm.ident = !{!4}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"target-abi", !"ilp32f"}
!2 = !{i32 7, !"frame-pointer", i32 2}
!3 = !{i32 1, !"SmallDataLimit", i32 8}
!4 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
