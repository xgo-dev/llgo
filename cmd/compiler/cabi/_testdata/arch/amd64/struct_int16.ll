; ModuleID = '../../wrap/struct_int16.c'
source_filename = "../../wrap/struct_int16.c"
target datalayout = "e-m:e-p270:32:32-p271:32:32-p272:64:64-i64:64-f80:128-n8:16:32:64-S128"
target triple = "amd64-unknown-linux-gnu"

%struct.point1 = type { i16 }
%struct.point2 = type { i16, i16 }
%struct.point3 = type { i16, i16, i16 }
%struct.point4 = type { i16, i16, i16, i16 }
%struct.point5 = type { i16, i16, i16, i16, i16 }
%struct.point6 = type { i16, i16, i16, i16, i16, i16 }
%struct.point7 = type { i16, i16, i16, i16, i16, i16, i16 }
%struct.point8 = type { i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point9 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point10 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point11 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point12 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point13 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point14 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point15 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point16 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point17 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point18 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point19 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }
%struct.point20 = type { i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16, i16 }

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i16 @demo1(i16 %0) #0 {
  %2 = alloca %struct.point1, align 2
  %3 = alloca %struct.point1, align 2
  %4 = getelementptr inbounds %struct.point1, %struct.point1* %3, i32 0, i32 0
  store i16 %0, i16* %4, align 2
  %5 = bitcast %struct.point1* %2 to i8*
  %6 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 2, i1 false)
  %7 = getelementptr inbounds %struct.point1, %struct.point1* %2, i32 0, i32 0
  %8 = load i16, i16* %7, align 2
  ret i16 %8
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo2(i32 %0) #0 {
  %2 = alloca %struct.point2, align 2
  %3 = alloca %struct.point2, align 2
  %4 = bitcast %struct.point2* %3 to i32*
  store i32 %0, i32* %4, align 2
  %5 = bitcast %struct.point2* %2 to i8*
  %6 = bitcast %struct.point2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 4, i1 false)
  %7 = bitcast %struct.point2* %2 to i32*
  %8 = load i32, i32* %7, align 2
  ret i32 %8
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i48 @demo3(i48 %0) #0 {
  %2 = alloca %struct.point3, align 2
  %3 = alloca %struct.point3, align 2
  %4 = alloca i48, align 8
  %5 = alloca i48, align 8
  store i48 %0, i48* %4, align 8
  %6 = bitcast %struct.point3* %3 to i8*
  %7 = bitcast i48* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point3* %2 to i8*
  %9 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 6, i1 false)
  %10 = bitcast i48* %5 to i8*
  %11 = bitcast %struct.point3* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 6, i1 false)
  %12 = load i48, i48* %5, align 8
  ret i48 %12
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i64 @demo4(i64 %0) #0 {
  %2 = alloca %struct.point4, align 2
  %3 = alloca %struct.point4, align 2
  %4 = bitcast %struct.point4* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point4* %2 to i8*
  %6 = bitcast %struct.point4* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point4* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i16 } @demo5(i64 %0, i16 %1) #0 {
  %3 = alloca %struct.point5, align 2
  %4 = alloca %struct.point5, align 2
  %5 = alloca { i64, i16 }, align 2
  %6 = alloca { i64, i16 }, align 8
  %7 = getelementptr inbounds { i64, i16 }, { i64, i16 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 2
  %8 = getelementptr inbounds { i64, i16 }, { i64, i16 }* %5, i32 0, i32 1
  store i16 %1, i16* %8, align 2
  %9 = bitcast %struct.point5* %4 to i8*
  %10 = bitcast { i64, i16 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %9, i8* align 2 %10, i64 10, i1 false)
  %11 = bitcast %struct.point5* %3 to i8*
  %12 = bitcast %struct.point5* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %11, i8* align 2 %12, i64 10, i1 false)
  %13 = bitcast { i64, i16 }* %6 to i8*
  %14 = bitcast %struct.point5* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %13, i8* align 2 %14, i64 10, i1 false)
  %15 = load { i64, i16 }, { i64, i16 }* %6, align 8
  ret { i64, i16 } %15
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i32 } @demo6(i64 %0, i32 %1) #0 {
  %3 = alloca %struct.point6, align 2
  %4 = alloca %struct.point6, align 2
  %5 = alloca { i64, i32 }, align 2
  %6 = alloca { i64, i32 }, align 8
  %7 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 2
  %8 = getelementptr inbounds { i64, i32 }, { i64, i32 }* %5, i32 0, i32 1
  store i32 %1, i32* %8, align 2
  %9 = bitcast %struct.point6* %4 to i8*
  %10 = bitcast { i64, i32 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %9, i8* align 2 %10, i64 12, i1 false)
  %11 = bitcast %struct.point6* %3 to i8*
  %12 = bitcast %struct.point6* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %11, i8* align 2 %12, i64 12, i1 false)
  %13 = bitcast { i64, i32 }* %6 to i8*
  %14 = bitcast %struct.point6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %13, i8* align 2 %14, i64 12, i1 false)
  %15 = load { i64, i32 }, { i64, i32 }* %6, align 8
  ret { i64, i32 } %15
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i48 } @demo7(i64 %0, i48 %1) #0 {
  %3 = alloca %struct.point7, align 2
  %4 = alloca %struct.point7, align 2
  %5 = alloca { i64, i48 }, align 2
  %6 = alloca { i64, i48 }, align 8
  %7 = getelementptr inbounds { i64, i48 }, { i64, i48 }* %5, i32 0, i32 0
  store i64 %0, i64* %7, align 2
  %8 = getelementptr inbounds { i64, i48 }, { i64, i48 }* %5, i32 0, i32 1
  store i48 %1, i48* %8, align 2
  %9 = bitcast %struct.point7* %4 to i8*
  %10 = bitcast { i64, i48 }* %5 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %9, i8* align 2 %10, i64 14, i1 false)
  %11 = bitcast %struct.point7* %3 to i8*
  %12 = bitcast %struct.point7* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %11, i8* align 2 %12, i64 14, i1 false)
  %13 = bitcast { i64, i48 }* %6 to i8*
  %14 = bitcast %struct.point7* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %13, i8* align 2 %14, i64 14, i1 false)
  %15 = load { i64, i48 }, { i64, i48 }* %6, align 8
  ret { i64, i48 } %15
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local { i64, i64 } @demo8(i64 %0, i64 %1) #0 {
  %3 = alloca %struct.point8, align 2
  %4 = alloca %struct.point8, align 2
  %5 = bitcast %struct.point8* %4 to { i64, i64 }*
  %6 = getelementptr inbounds { i64, i64 }, { i64, i64 }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 2
  %7 = getelementptr inbounds { i64, i64 }, { i64, i64 }* %5, i32 0, i32 1
  store i64 %1, i64* %7, align 2
  %8 = bitcast %struct.point8* %3 to i8*
  %9 = bitcast %struct.point8* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 16, i1 false)
  %10 = bitcast %struct.point8* %3 to { i64, i64 }*
  %11 = load { i64, i64 }, { i64, i64 }* %10, align 2
  ret { i64, i64 } %11
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo9(%struct.point9* noalias sret(%struct.point9) align 2 %0, %struct.point9* noundef byval(%struct.point9) align 8 %1) #0 {
  %3 = bitcast %struct.point9* %0 to i8*
  %4 = bitcast %struct.point9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo10(%struct.point10* noalias sret(%struct.point10) align 2 %0, %struct.point10* noundef byval(%struct.point10) align 8 %1) #0 {
  %3 = bitcast %struct.point10* %0 to i8*
  %4 = bitcast %struct.point10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo11(%struct.point11* noalias sret(%struct.point11) align 2 %0, %struct.point11* noundef byval(%struct.point11) align 8 %1) #0 {
  %3 = bitcast %struct.point11* %0 to i8*
  %4 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 22, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo12(%struct.point12* noalias sret(%struct.point12) align 2 %0, %struct.point12* noundef byval(%struct.point12) align 8 %1) #0 {
  %3 = bitcast %struct.point12* %0 to i8*
  %4 = bitcast %struct.point12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 24, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo13(%struct.point13* noalias sret(%struct.point13) align 2 %0, %struct.point13* noundef byval(%struct.point13) align 8 %1) #0 {
  %3 = bitcast %struct.point13* %0 to i8*
  %4 = bitcast %struct.point13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 26, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo14(%struct.point14* noalias sret(%struct.point14) align 2 %0, %struct.point14* noundef byval(%struct.point14) align 8 %1) #0 {
  %3 = bitcast %struct.point14* %0 to i8*
  %4 = bitcast %struct.point14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 28, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo15(%struct.point15* noalias sret(%struct.point15) align 2 %0, %struct.point15* noundef byval(%struct.point15) align 8 %1) #0 {
  %3 = bitcast %struct.point15* %0 to i8*
  %4 = bitcast %struct.point15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 30, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo16(%struct.point16* noalias sret(%struct.point16) align 2 %0, %struct.point16* noundef byval(%struct.point16) align 8 %1) #0 {
  %3 = bitcast %struct.point16* %0 to i8*
  %4 = bitcast %struct.point16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 32, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo17(%struct.point17* noalias sret(%struct.point17) align 2 %0, %struct.point17* noundef byval(%struct.point17) align 8 %1) #0 {
  %3 = bitcast %struct.point17* %0 to i8*
  %4 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 34, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo18(%struct.point18* noalias sret(%struct.point18) align 2 %0, %struct.point18* noundef byval(%struct.point18) align 8 %1) #0 {
  %3 = bitcast %struct.point18* %0 to i8*
  %4 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 36, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo19(%struct.point19* noalias sret(%struct.point19) align 2 %0, %struct.point19* noundef byval(%struct.point19) align 8 %1) #0 {
  %3 = bitcast %struct.point19* %0 to i8*
  %4 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 38, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo20(%struct.point20* noalias sret(%struct.point20) align 2 %0, %struct.point20* noundef byval(%struct.point20) align 8 %1) #0 {
  %3 = bitcast %struct.point20* %0 to i8*
  %4 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %3, i8* align 8 %4, i64 40, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone uwtable "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="x86-64" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3, !4}
!llvm.ident = !{!5}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 7, !"PIC Level", i32 2}
!2 = !{i32 7, !"PIE Level", i32 2}
!3 = !{i32 7, !"uwtable", i32 2}
!4 = !{i32 7, !"frame-pointer", i32 2}
!5 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
