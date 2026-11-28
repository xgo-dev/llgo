; ModuleID = '../../wrap/struct_pointer.c'
source_filename = "../../wrap/struct_pointer.c"
target datalayout = "e-m:e-p:32:32-p10:8:8-p20:8:8-i64:64-f128:64-n32:64-S128-ni:1:10:20"
target triple = "wasm32-unknown-emscripten"

%struct.point1 = type { i8* }
%struct.point2 = type { i8*, i8* }
%struct.point3 = type { i8*, i8*, i8* }
%struct.point4 = type { i8*, i8*, i8*, i8* }
%struct.point5 = type { i8*, i8*, i8*, i8*, i8* }
%struct.point6 = type { i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point7 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point8 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point9 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point10 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point11 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point12 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point13 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point14 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point15 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point16 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point17 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point18 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point19 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }
%struct.point20 = type { i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8*, i8* }

; Function Attrs: noinline nounwind optnone
define hidden i8* @demo1(i8* %0) #0 {
  %2 = alloca %struct.point1, align 4
  %3 = alloca %struct.point1, align 4
  %4 = getelementptr inbounds %struct.point1, %struct.point1* %3, i32 0, i32 0
  store i8* %0, i8** %4, align 4
  %5 = bitcast %struct.point1* %2 to i8*
  %6 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %5, i8* align 4 %6, i32 4, i1 false)
  %7 = getelementptr inbounds %struct.point1, %struct.point1* %2, i32 0, i32 0
  %8 = load i8*, i8** %7, align 4
  ret i8* %8
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define hidden void @demo2(%struct.point2* noalias sret(%struct.point2) align 4 %0, %struct.point2* noundef byval(%struct.point2) align 4 %1) #0 {
  %3 = bitcast %struct.point2* %0 to i8*
  %4 = bitcast %struct.point2* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo3(%struct.point3* noalias sret(%struct.point3) align 4 %0, %struct.point3* noundef byval(%struct.point3) align 4 %1) #0 {
  %3 = bitcast %struct.point3* %0 to i8*
  %4 = bitcast %struct.point3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo4(%struct.point4* noalias sret(%struct.point4) align 4 %0, %struct.point4* noundef byval(%struct.point4) align 4 %1) #0 {
  %3 = bitcast %struct.point4* %0 to i8*
  %4 = bitcast %struct.point4* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo5(%struct.point5* noalias sret(%struct.point5) align 4 %0, %struct.point5* noundef byval(%struct.point5) align 4 %1) #0 {
  %3 = bitcast %struct.point5* %0 to i8*
  %4 = bitcast %struct.point5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo6(%struct.point6* noalias sret(%struct.point6) align 4 %0, %struct.point6* noundef byval(%struct.point6) align 4 %1) #0 {
  %3 = bitcast %struct.point6* %0 to i8*
  %4 = bitcast %struct.point6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo7(%struct.point7* noalias sret(%struct.point7) align 4 %0, %struct.point7* noundef byval(%struct.point7) align 4 %1) #0 {
  %3 = bitcast %struct.point7* %0 to i8*
  %4 = bitcast %struct.point7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo8(%struct.point8* noalias sret(%struct.point8) align 4 %0, %struct.point8* noundef byval(%struct.point8) align 4 %1) #0 {
  %3 = bitcast %struct.point8* %0 to i8*
  %4 = bitcast %struct.point8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo9(%struct.point9* noalias sret(%struct.point9) align 4 %0, %struct.point9* noundef byval(%struct.point9) align 4 %1) #0 {
  %3 = bitcast %struct.point9* %0 to i8*
  %4 = bitcast %struct.point9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo10(%struct.point10* noalias sret(%struct.point10) align 4 %0, %struct.point10* noundef byval(%struct.point10) align 4 %1) #0 {
  %3 = bitcast %struct.point10* %0 to i8*
  %4 = bitcast %struct.point10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 40, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo11(%struct.point11* noalias sret(%struct.point11) align 4 %0, %struct.point11* noundef byval(%struct.point11) align 4 %1) #0 {
  %3 = bitcast %struct.point11* %0 to i8*
  %4 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 44, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo12(%struct.point12* noalias sret(%struct.point12) align 4 %0, %struct.point12* noundef byval(%struct.point12) align 4 %1) #0 {
  %3 = bitcast %struct.point12* %0 to i8*
  %4 = bitcast %struct.point12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 48, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo13(%struct.point13* noalias sret(%struct.point13) align 4 %0, %struct.point13* noundef byval(%struct.point13) align 4 %1) #0 {
  %3 = bitcast %struct.point13* %0 to i8*
  %4 = bitcast %struct.point13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 52, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo14(%struct.point14* noalias sret(%struct.point14) align 4 %0, %struct.point14* noundef byval(%struct.point14) align 4 %1) #0 {
  %3 = bitcast %struct.point14* %0 to i8*
  %4 = bitcast %struct.point14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 56, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo15(%struct.point15* noalias sret(%struct.point15) align 4 %0, %struct.point15* noundef byval(%struct.point15) align 4 %1) #0 {
  %3 = bitcast %struct.point15* %0 to i8*
  %4 = bitcast %struct.point15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 60, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo16(%struct.point16* noalias sret(%struct.point16) align 4 %0, %struct.point16* noundef byval(%struct.point16) align 4 %1) #0 {
  %3 = bitcast %struct.point16* %0 to i8*
  %4 = bitcast %struct.point16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 64, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo17(%struct.point17* noalias sret(%struct.point17) align 4 %0, %struct.point17* noundef byval(%struct.point17) align 4 %1) #0 {
  %3 = bitcast %struct.point17* %0 to i8*
  %4 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 68, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo18(%struct.point18* noalias sret(%struct.point18) align 4 %0, %struct.point18* noundef byval(%struct.point18) align 4 %1) #0 {
  %3 = bitcast %struct.point18* %0 to i8*
  %4 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 72, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo19(%struct.point19* noalias sret(%struct.point19) align 4 %0, %struct.point19* noundef byval(%struct.point19) align 4 %1) #0 {
  %3 = bitcast %struct.point19* %0 to i8*
  %4 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 76, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo20(%struct.point20* noalias sret(%struct.point20) align 4 %0, %struct.point20* noundef byval(%struct.point20) align 4 %1) #0 {
  %3 = bitcast %struct.point20* %0 to i8*
  %4 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 80, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="none" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0}
!llvm.ident = !{!1}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
