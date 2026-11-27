; ModuleID = '../../wrap/array_pointer.c'
source_filename = "../../wrap/array_pointer.c"
target datalayout = "e-m:e-p:32:32-p10:8:8-p20:8:8-i64:64-f128:64-n32:64-S128-ni:1:10:20"
target triple = "wasm32-unknown-emscripten"

%struct.array1 = type { [1 x i8*] }
%struct.array2 = type { [2 x i8*] }
%struct.array3 = type { [3 x i8*] }
%struct.array4 = type { [4 x i8*] }
%struct.array5 = type { [5 x i8*] }
%struct.array6 = type { [6 x i8*] }
%struct.array7 = type { [7 x i8*] }
%struct.array8 = type { [8 x i8*] }
%struct.array9 = type { [9 x i8*] }
%struct.array10 = type { [10 x i8*] }
%struct.array11 = type { [11 x i8*] }
%struct.array12 = type { [12 x i8*] }
%struct.array13 = type { [13 x i8*] }
%struct.array14 = type { [14 x i8*] }
%struct.array15 = type { [15 x i8*] }
%struct.array16 = type { [16 x i8*] }
%struct.array17 = type { [17 x i8*] }
%struct.array18 = type { [18 x i8*] }
%struct.array19 = type { [19 x i8*] }
%struct.array20 = type { [20 x i8*] }

; Function Attrs: noinline nounwind optnone
define hidden i8* @demo1(i8* %0) #0 {
  %2 = alloca %struct.array1, align 4
  %3 = alloca %struct.array1, align 4
  %4 = getelementptr inbounds %struct.array1, %struct.array1* %3, i32 0, i32 0
  %5 = bitcast [1 x i8*]* %4 to i8**
  store i8* %0, i8** %5, align 4
  %6 = bitcast %struct.array1* %2 to i8*
  %7 = bitcast %struct.array1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %6, i8* align 4 %7, i32 4, i1 false)
  %8 = getelementptr inbounds %struct.array1, %struct.array1* %2, i32 0, i32 0
  %9 = bitcast [1 x i8*]* %8 to i8**
  %10 = load i8*, i8** %9, align 4
  ret i8* %10
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define hidden void @demo2(%struct.array2* noalias sret(%struct.array2) align 4 %0, %struct.array2* noundef byval(%struct.array2) align 4 %1) #0 {
  %3 = bitcast %struct.array2* %0 to i8*
  %4 = bitcast %struct.array2* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo3(%struct.array3* noalias sret(%struct.array3) align 4 %0, %struct.array3* noundef byval(%struct.array3) align 4 %1) #0 {
  %3 = bitcast %struct.array3* %0 to i8*
  %4 = bitcast %struct.array3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo4(%struct.array4* noalias sret(%struct.array4) align 4 %0, %struct.array4* noundef byval(%struct.array4) align 4 %1) #0 {
  %3 = bitcast %struct.array4* %0 to i8*
  %4 = bitcast %struct.array4* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo5(%struct.array5* noalias sret(%struct.array5) align 4 %0, %struct.array5* noundef byval(%struct.array5) align 4 %1) #0 {
  %3 = bitcast %struct.array5* %0 to i8*
  %4 = bitcast %struct.array5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo6(%struct.array6* noalias sret(%struct.array6) align 4 %0, %struct.array6* noundef byval(%struct.array6) align 4 %1) #0 {
  %3 = bitcast %struct.array6* %0 to i8*
  %4 = bitcast %struct.array6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo7(%struct.array7* noalias sret(%struct.array7) align 4 %0, %struct.array7* noundef byval(%struct.array7) align 4 %1) #0 {
  %3 = bitcast %struct.array7* %0 to i8*
  %4 = bitcast %struct.array7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo8(%struct.array8* noalias sret(%struct.array8) align 4 %0, %struct.array8* noundef byval(%struct.array8) align 4 %1) #0 {
  %3 = bitcast %struct.array8* %0 to i8*
  %4 = bitcast %struct.array8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo9(%struct.array9* noalias sret(%struct.array9) align 4 %0, %struct.array9* noundef byval(%struct.array9) align 4 %1) #0 {
  %3 = bitcast %struct.array9* %0 to i8*
  %4 = bitcast %struct.array9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo10(%struct.array10* noalias sret(%struct.array10) align 4 %0, %struct.array10* noundef byval(%struct.array10) align 4 %1) #0 {
  %3 = bitcast %struct.array10* %0 to i8*
  %4 = bitcast %struct.array10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo11(%struct.array11* noalias sret(%struct.array11) align 4 %0, %struct.array11* noundef byval(%struct.array11) align 4 %1) #0 {
  %3 = bitcast %struct.array11* %0 to i8*
  %4 = bitcast %struct.array11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 44, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo12(%struct.array12* noalias sret(%struct.array12) align 4 %0, %struct.array12* noundef byval(%struct.array12) align 4 %1) #0 {
  %3 = bitcast %struct.array12* %0 to i8*
  %4 = bitcast %struct.array12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo13(%struct.array13* noalias sret(%struct.array13) align 4 %0, %struct.array13* noundef byval(%struct.array13) align 4 %1) #0 {
  %3 = bitcast %struct.array13* %0 to i8*
  %4 = bitcast %struct.array13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 52, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo14(%struct.array14* noalias sret(%struct.array14) align 4 %0, %struct.array14* noundef byval(%struct.array14) align 4 %1) #0 {
  %3 = bitcast %struct.array14* %0 to i8*
  %4 = bitcast %struct.array14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo15(%struct.array15* noalias sret(%struct.array15) align 4 %0, %struct.array15* noundef byval(%struct.array15) align 4 %1) #0 {
  %3 = bitcast %struct.array15* %0 to i8*
  %4 = bitcast %struct.array15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 60, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo16(%struct.array16* noalias sret(%struct.array16) align 4 %0, %struct.array16* noundef byval(%struct.array16) align 4 %1) #0 {
  %3 = bitcast %struct.array16* %0 to i8*
  %4 = bitcast %struct.array16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo17(%struct.array17* noalias sret(%struct.array17) align 4 %0, %struct.array17* noundef byval(%struct.array17) align 4 %1) #0 {
  %3 = bitcast %struct.array17* %0 to i8*
  %4 = bitcast %struct.array17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 68, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo18(%struct.array18* noalias sret(%struct.array18) align 4 %0, %struct.array18* noundef byval(%struct.array18) align 4 %1) #0 {
  %3 = bitcast %struct.array18* %0 to i8*
  %4 = bitcast %struct.array18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo19(%struct.array19* noalias sret(%struct.array19) align 4 %0, %struct.array19* noundef byval(%struct.array19) align 4 %1) #0 {
  %3 = bitcast %struct.array19* %0 to i8*
  %4 = bitcast %struct.array19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 76, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo20(%struct.array20* noalias sret(%struct.array20) align 4 %0, %struct.array20* noundef byval(%struct.array20) align 4 %1) #0 {
  %3 = bitcast %struct.array20* %0 to i8*
  %4 = bitcast %struct.array20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 80, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="none" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0}
!llvm.ident = !{!1}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
