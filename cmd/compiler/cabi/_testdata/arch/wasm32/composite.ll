; ModuleID = '../../wrap/composite.c'
source_filename = "../../wrap/composite.c"
target datalayout = "e-m:e-p:32:32-p10:8:8-p20:8:8-i64:64-f128:64-n32:64-S128-ni:1:10:20"
target triple = "wasm32-unknown-emscripten"

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
define hidden i32 @demo_array10(i32 %0) #0 {
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
define hidden void @demo_array11(%struct.array11* noalias sret(%struct.array11) align 4 %0, %struct.array11* noundef byval(%struct.array11) align 4 %1) #0 {
  %3 = bitcast %struct.array11* %0 to i8*
  %4 = bitcast %struct.array11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden i32 @demo_point10(i32 %0) #0 {
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
define hidden void @demo_point11(%struct.point11* noalias sret(%struct.point11) align 4 %0, %struct.point11* noundef byval(%struct.point11) align 4 %1) #0 {
  %3 = bitcast %struct.point11* %0 to i8*
  %4 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo_array20(%struct.array20* noalias sret(%struct.array20) align 4 %0, %struct.array20* noundef byval(%struct.array20) align 4 %1) #0 {
  %3 = bitcast %struct.array20* %0 to i8*
  %4 = bitcast %struct.array20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo_array21(%struct.array21* noalias sret(%struct.array21) align 4 %0, %struct.array21* noundef byval(%struct.array21) align 4 %1) #0 {
  %3 = bitcast %struct.array21* %0 to i8*
  %4 = bitcast %struct.array21* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo_point20(%struct.point20* noalias sret(%struct.point20) align 4 %0, %struct.point20* noundef byval(%struct.point20) align 4 %1) #0 {
  %3 = bitcast %struct.point20* %0 to i8*
  %4 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo_point21(%struct.point21* noalias sret(%struct.point21) align 4 %0, %struct.point21* noundef byval(%struct.point21) align 4 %1) #0 {
  %3 = bitcast %struct.point21* %0 to i8*
  %4 = bitcast %struct.point21* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="none" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0}
!llvm.ident = !{!1}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
