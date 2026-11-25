; ModuleID = '../../wrap/empty.c'
source_filename = "../../wrap/empty.c"
target datalayout = "e-m:e-p:32:32-p270:32:32-p271:32:32-p272:64:64-f64:32:64-f80:32-n8:16:32-S128"
target triple = "i386-unknown-linux-gnu"

%struct.empty = type {}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo0(%struct.empty* noalias sret(%struct.empty) align 1 %0) #0 {
  %2 = alloca i8*, align 4
  %3 = alloca %struct.empty, align 1
  %4 = bitcast %struct.empty* %0 to i8*
  store i8* %4, i8** %2, align 4
  %5 = bitcast %struct.empty* %0 to i8*
  %6 = bitcast %struct.empty* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %5, i8* align 1 %6, i32 0, i1 false)
  ret void
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone uwtable
define dso_local void @demo1(%struct.empty* noalias sret(%struct.empty) align 1 %0, i32 noundef %1) #0 {
  %3 = alloca i8*, align 4
  %4 = alloca %struct.empty, align 1
  %5 = alloca i32, align 4
  %6 = bitcast %struct.empty* %0 to i8*
  store i8* %6, i8** %3, align 4
  store i32 %1, i32* %5, align 4
  %7 = bitcast %struct.empty* %0 to i8*
  %8 = bitcast %struct.empty* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 1 %7, i8* align 1 %8, i32 0, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone uwtable
define dso_local i32 @demo2(i32 noundef %0) #0 {
  %2 = alloca %struct.empty, align 1
  %3 = alloca i32, align 4
  store i32 %0, i32* %3, align 4
  %4 = load i32, i32* %3, align 4
  ret i32 %4
}

; Function Attrs: noinline nounwind optnone uwtable
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

attributes #0 = { noinline nounwind optnone uwtable "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="pentium4" "target-features"="+cx8,+fxsr,+mmx,+sse,+sse2,+x87" "tune-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3, !4, !5}
!llvm.ident = !{!6}

!0 = !{i32 1, !"NumRegisterParameters", i32 0}
!1 = !{i32 1, !"wchar_size", i32 4}
!2 = !{i32 7, !"PIC Level", i32 2}
!3 = !{i32 7, !"PIE Level", i32 2}
!4 = !{i32 7, !"uwtable", i32 2}
!5 = !{i32 7, !"frame-pointer", i32 2}
!6 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
