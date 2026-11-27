; ModuleID = '../../wrap/array_float64.c'
source_filename = "../../wrap/array_float64.c"
target datalayout = "e-m:e-p:32:32-Fi8-i64:64-v128:64:128-a:0:32-n32-S64"
target triple = "armv6kz-unknown-linux-gnueabihf"

%struct.array1 = type { [1 x double] }
%struct.array2 = type { [2 x double] }
%struct.array3 = type { [3 x double] }
%struct.array4 = type { [4 x double] }
%struct.array5 = type { [5 x double] }
%struct.array6 = type { [6 x double] }
%struct.array7 = type { [7 x double] }
%struct.array8 = type { [8 x double] }
%struct.array9 = type { [9 x double] }
%struct.array10 = type { [10 x double] }
%struct.array11 = type { [11 x double] }
%struct.array12 = type { [12 x double] }
%struct.array13 = type { [13 x double] }
%struct.array14 = type { [14 x double] }
%struct.array15 = type { [15 x double] }
%struct.array16 = type { [16 x double] }
%struct.array17 = type { [17 x double] }
%struct.array18 = type { [18 x double] }
%struct.array19 = type { [19 x double] }
%struct.array20 = type { [20 x double] }

; Function Attrs: noinline nounwind optnone
define dso_local %struct.array1 @demo1(%struct.array1 %0) #0 {
  %2 = alloca %struct.array1, align 8
  %3 = alloca %struct.array1, align 8
  store %struct.array1 %0, %struct.array1* %3, align 8
  %4 = bitcast %struct.array1* %2 to i8*
  %5 = bitcast %struct.array1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %4, i8* align 8 %5, i32 8, i1 false)
  %6 = load %struct.array1, %struct.array1* %2, align 8
  ret %struct.array1 %6
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local %struct.array2 @demo2(%struct.array2 %0) #0 {
  %2 = alloca %struct.array2, align 8
  %3 = alloca %struct.array2, align 8
  store %struct.array2 %0, %struct.array2* %3, align 8
  %4 = bitcast %struct.array2* %2 to i8*
  %5 = bitcast %struct.array2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %4, i8* align 8 %5, i32 16, i1 false)
  %6 = load %struct.array2, %struct.array2* %2, align 8
  ret %struct.array2 %6
}

; Function Attrs: noinline nounwind optnone
define dso_local %struct.array3 @demo3(%struct.array3 %0) #0 {
  %2 = alloca %struct.array3, align 8
  %3 = alloca %struct.array3, align 8
  store %struct.array3 %0, %struct.array3* %3, align 8
  %4 = bitcast %struct.array3* %2 to i8*
  %5 = bitcast %struct.array3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %4, i8* align 8 %5, i32 24, i1 false)
  %6 = load %struct.array3, %struct.array3* %2, align 8
  ret %struct.array3 %6
}

; Function Attrs: noinline nounwind optnone
define dso_local %struct.array4 @demo4(%struct.array4 %0) #0 {
  %2 = alloca %struct.array4, align 8
  %3 = alloca %struct.array4, align 8
  store %struct.array4 %0, %struct.array4* %3, align 8
  %4 = bitcast %struct.array4* %2 to i8*
  %5 = bitcast %struct.array4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %4, i8* align 8 %5, i32 32, i1 false)
  %6 = load %struct.array4, %struct.array4* %2, align 8
  ret %struct.array4 %6
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo5(%struct.array5* noalias sret(%struct.array5) align 8 %0, [5 x i64] %1) #0 {
  %3 = alloca %struct.array5, align 8
  %4 = getelementptr inbounds %struct.array5, %struct.array5* %3, i32 0, i32 0
  %5 = bitcast [5 x double]* %4 to [5 x i64]*
  store [5 x i64] %1, [5 x i64]* %5, align 8
  %6 = bitcast %struct.array5* %0 to i8*
  %7 = bitcast %struct.array5* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %6, i8* align 8 %7, i32 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo6(%struct.array6* noalias sret(%struct.array6) align 8 %0, [6 x i64] %1) #0 {
  %3 = alloca %struct.array6, align 8
  %4 = getelementptr inbounds %struct.array6, %struct.array6* %3, i32 0, i32 0
  %5 = bitcast [6 x double]* %4 to [6 x i64]*
  store [6 x i64] %1, [6 x i64]* %5, align 8
  %6 = bitcast %struct.array6* %0 to i8*
  %7 = bitcast %struct.array6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %6, i8* align 8 %7, i32 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(%struct.array7* noalias sret(%struct.array7) align 8 %0, [7 x i64] %1) #0 {
  %3 = alloca %struct.array7, align 8
  %4 = getelementptr inbounds %struct.array7, %struct.array7* %3, i32 0, i32 0
  %5 = bitcast [7 x double]* %4 to [7 x i64]*
  store [7 x i64] %1, [7 x i64]* %5, align 8
  %6 = bitcast %struct.array7* %0 to i8*
  %7 = bitcast %struct.array7* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %6, i8* align 8 %7, i32 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo8(%struct.array8* noalias sret(%struct.array8) align 8 %0, [8 x i64] %1) #0 {
  %3 = alloca %struct.array8, align 8
  %4 = getelementptr inbounds %struct.array8, %struct.array8* %3, i32 0, i32 0
  %5 = bitcast [8 x double]* %4 to [8 x i64]*
  store [8 x i64] %1, [8 x i64]* %5, align 8
  %6 = bitcast %struct.array8* %0 to i8*
  %7 = bitcast %struct.array8* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %6, i8* align 8 %7, i32 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(%struct.array9* noalias sret(%struct.array9) align 8 %0, %struct.array9* noundef byval(%struct.array9) align 8 %1) #0 {
  %3 = bitcast %struct.array9* %0 to i8*
  %4 = bitcast %struct.array9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(%struct.array10* noalias sret(%struct.array10) align 8 %0, %struct.array10* noundef byval(%struct.array10) align 8 %1) #0 {
  %3 = bitcast %struct.array10* %0 to i8*
  %4 = bitcast %struct.array10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 80, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(%struct.array11* noalias sret(%struct.array11) align 8 %0, %struct.array11* noundef byval(%struct.array11) align 8 %1) #0 {
  %3 = bitcast %struct.array11* %0 to i8*
  %4 = bitcast %struct.array11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 88, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(%struct.array12* noalias sret(%struct.array12) align 8 %0, %struct.array12* noundef byval(%struct.array12) align 8 %1) #0 {
  %3 = bitcast %struct.array12* %0 to i8*
  %4 = bitcast %struct.array12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 96, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(%struct.array13* noalias sret(%struct.array13) align 8 %0, %struct.array13* noundef byval(%struct.array13) align 8 %1) #0 {
  %3 = bitcast %struct.array13* %0 to i8*
  %4 = bitcast %struct.array13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 104, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(%struct.array14* noalias sret(%struct.array14) align 8 %0, %struct.array14* noundef byval(%struct.array14) align 8 %1) #0 {
  %3 = bitcast %struct.array14* %0 to i8*
  %4 = bitcast %struct.array14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 112, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(%struct.array15* noalias sret(%struct.array15) align 8 %0, %struct.array15* noundef byval(%struct.array15) align 8 %1) #0 {
  %3 = bitcast %struct.array15* %0 to i8*
  %4 = bitcast %struct.array15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 120, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(%struct.array16* noalias sret(%struct.array16) align 8 %0, %struct.array16* noundef byval(%struct.array16) align 8 %1) #0 {
  %3 = bitcast %struct.array16* %0 to i8*
  %4 = bitcast %struct.array16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 128, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(%struct.array17* noalias sret(%struct.array17) align 8 %0, %struct.array17* noundef byval(%struct.array17) align 8 %1) #0 {
  %3 = bitcast %struct.array17* %0 to i8*
  %4 = bitcast %struct.array17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 136, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(%struct.array18* noalias sret(%struct.array18) align 8 %0, %struct.array18* noundef byval(%struct.array18) align 8 %1) #0 {
  %3 = bitcast %struct.array18* %0 to i8*
  %4 = bitcast %struct.array18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 144, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(%struct.array19* noalias sret(%struct.array19) align 8 %0, %struct.array19* noundef byval(%struct.array19) align 8 %1) #0 {
  %3 = bitcast %struct.array19* %0 to i8*
  %4 = bitcast %struct.array19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 152, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(%struct.array20* noalias sret(%struct.array20) align 8 %0, %struct.array20* noundef byval(%struct.array20) align 8 %1) #0 {
  %3 = bitcast %struct.array20* %0 to i8*
  %4 = bitcast %struct.array20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 160, i1 false)
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
