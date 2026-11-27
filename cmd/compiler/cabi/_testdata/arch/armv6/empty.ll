; ModuleID = '../../wrap/empty.c'
source_filename = "../../wrap/empty.c"
target datalayout = "e-m:e-p:32:32-Fi8-i64:64-v128:64:128-a:0:32-n32-S64"
target triple = "armv6kz-unknown-linux-gnueabihf"

%struct.empty = type {}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo0() #0 {
  %1 = alloca %struct.empty, align 1
  %2 = alloca %struct.empty, align 1
  %3 = bitcast %struct.empty* %1 to i8*
  %4 = bitcast %struct.empty* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %3, i8* align 1 %4, i32 0, i1 false)
  ret void
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1(i32 noundef %0) #0 {
  %2 = alloca %struct.empty, align 1
  %3 = alloca %struct.empty, align 1
  %4 = alloca i32, align 4
  store i32 %0, i32* %4, align 4
  %5 = bitcast %struct.empty* %2 to i8*
  %6 = bitcast %struct.empty* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 1 %6, i32 0, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo2(i32 noundef %0) #0 {
  %2 = alloca %struct.empty, align 1
  %3 = alloca i32, align 4
  store i32 %0, i32* %3, align 4
  %4 = load i32, i32* %3, align 4
  ret i32 %4
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo3(i32 noundef %0, i32 noundef %1) #0 {
  %3 = alloca %struct.empty, align 1
  %4 = alloca i32, align 4
  %5 = alloca i32, align 4
  store i32 %0, i32* %4, align 4
  store i32 %1, i32* %5, align 4
  %6 = load i32, i32* %4, align 4
  %7 = load i32, i32* %5, align 4
  %8 = add nsw i32 %6, %7
  ret i32 %8
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
