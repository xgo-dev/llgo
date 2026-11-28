; ModuleID = '../../wrap/composite.c'
source_filename = "../../wrap/composite.c"
target datalayout = "e-m:e-p:32:32-Fi8-i64:64-v128:64:128-a:0:32-n32-S64"
target triple = "armv6kz-unknown-linux-gnueabihf"

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
define dso_local i32 @demo_array10([1 x i32] %0) #0 {
  %2 = alloca %struct.array10, align 4
  %3 = alloca %struct.array10, align 4
  %4 = getelementptr inbounds %struct.array10, %struct.array10* %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basearray1, %struct.basearray1* %4, i32 0, i32 0
  store [1 x i32] %0, [1 x i32]* %5, align 4
  %6 = bitcast %struct.array10* %2 to i8*
  %7 = bitcast %struct.array10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %6, i8* align 4 %7, i32 4, i1 false)
  %8 = getelementptr inbounds %struct.array10, %struct.array10* %2, i32 0, i32 0
  %9 = getelementptr inbounds %struct.basearray1, %struct.basearray1* %8, i32 0, i32 0
  %10 = bitcast [1 x i32]* %9 to i32*
  %11 = load i32, i32* %10, align 4
  ret i32 %11
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_array11(%struct.array11* noalias sret(%struct.array11) align 4 %0, [2 x i32] %1) #0 {
  %3 = alloca %struct.array11, align 4
  %4 = bitcast %struct.array11* %3 to [2 x i32]*
  store [2 x i32] %1, [2 x i32]* %4, align 4
  %5 = bitcast %struct.array11* %0 to i8*
  %6 = bitcast %struct.array11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo_point10([1 x i32] %0) #0 {
  %2 = alloca %struct.point10, align 4
  %3 = alloca %struct.point10, align 4
  %4 = getelementptr inbounds %struct.point10, %struct.point10* %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basepoint1, %struct.basepoint1* %4, i32 0, i32 0
  %6 = bitcast i32* %5 to [1 x i32]*
  store [1 x i32] %0, [1 x i32]* %6, align 4
  %7 = bitcast %struct.point10* %2 to i8*
  %8 = bitcast %struct.point10* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %7, i8* align 4 %8, i32 4, i1 false)
  %9 = getelementptr inbounds %struct.point10, %struct.point10* %2, i32 0, i32 0
  %10 = getelementptr inbounds %struct.basepoint1, %struct.basepoint1* %9, i32 0, i32 0
  %11 = load i32, i32* %10, align 4
  ret i32 %11
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_point11(%struct.point11* noalias sret(%struct.point11) align 4 %0, [2 x i32] %1) #0 {
  %3 = alloca %struct.point11, align 4
  %4 = bitcast %struct.point11* %3 to [2 x i32]*
  store [2 x i32] %1, [2 x i32]* %4, align 4
  %5 = bitcast %struct.point11* %0 to i8*
  %6 = bitcast %struct.point11* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_array20(%struct.array20* noalias sret(%struct.array20) align 4 %0, [2 x i32] %1) #0 {
  %3 = alloca %struct.array20, align 4
  %4 = getelementptr inbounds %struct.array20, %struct.array20* %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basearray2, %struct.basearray2* %4, i32 0, i32 0
  store [2 x i32] %1, [2 x i32]* %5, align 4
  %6 = bitcast %struct.array20* %0 to i8*
  %7 = bitcast %struct.array20* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %6, i8* align 4 %7, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_array21(%struct.array21* noalias sret(%struct.array21) align 4 %0, [3 x i32] %1) #0 {
  %3 = alloca %struct.array21, align 4
  %4 = bitcast %struct.array21* %3 to [3 x i32]*
  store [3 x i32] %1, [3 x i32]* %4, align 4
  %5 = bitcast %struct.array21* %0 to i8*
  %6 = bitcast %struct.array21* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_point20(%struct.point20* noalias sret(%struct.point20) align 4 %0, [2 x i32] %1) #0 {
  %3 = alloca %struct.point20, align 4
  %4 = getelementptr inbounds %struct.point20, %struct.point20* %3, i32 0, i32 0
  %5 = bitcast %struct.basepoint2* %4 to [2 x i32]*
  store [2 x i32] %1, [2 x i32]* %5, align 4
  %6 = bitcast %struct.point20* %0 to i8*
  %7 = bitcast %struct.point20* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %6, i8* align 4 %7, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_point21(%struct.point21* noalias sret(%struct.point21) align 4 %0, [3 x i32] %1) #0 {
  %3 = alloca %struct.point21, align 4
  %4 = bitcast %struct.point21* %3 to [3 x i32]*
  store [3 x i32] %1, [3 x i32]* %4, align 4
  %5 = bitcast %struct.point21* %0 to i8*
  %6 = bitcast %struct.point21* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 12, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="arm1176jzf-s" "target-features"="+armv6kz,+dsp,+fp64,+strict-align,+vfp2,+vfp2sp,-aes,-d32,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-neon,-sha2,-thumb-mode,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3, !4, !5, !6, !7, !8}
!llvm.ident = !{!9}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"min_enum_size", i32 4}
!2 = !{i32 8, !"branch-target-enforcement", i32 0}
!3 = !{i32 8, !"sign-return-address", i32 0}
!4 = !{i32 8, !"sign-return-address-all", i32 0}
!5 = !{i32 8, !"sign-return-address-with-bkey", i32 0}
!6 = !{i32 7, !"PIC Level", i32 2}
!7 = !{i32 7, !"PIE Level", i32 2}
!8 = !{i32 7, !"frame-pointer", i32 2}
!9 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
