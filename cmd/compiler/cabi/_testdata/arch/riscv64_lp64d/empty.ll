; ModuleID = '../../wrap/empty.c'
source_filename = "../../wrap/empty.c"
target datalayout = "e-m:e-p:64:64-i64:64-i128:128-n64-S128"
target triple = "riscv64-unknown-unknown-elf"

%struct.empty = type {}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo0() #0 {
  %1 = alloca %struct.empty, align 1
  %2 = alloca %struct.empty, align 1
  %3 = bitcast %struct.empty* %1 to i8*
  %4 = bitcast %struct.empty* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %3, i8* align 1 %4, i64 0, i1 false)
  ret void
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1(i32 noundef signext %0) #0 {
  %2 = alloca %struct.empty, align 1
  %3 = alloca %struct.empty, align 1
  %4 = alloca i32, align 4
  store i32 %0, i32* %4, align 4
  %5 = bitcast %struct.empty* %2 to i8*
  %6 = bitcast %struct.empty* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 1 %5, i8* align 1 %6, i64 0, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local signext i32 @demo2(i32 noundef signext %0) #0 {
  %2 = alloca %struct.empty, align 1
  %3 = alloca i32, align 4
  store i32 %0, i32* %3, align 4
  %4 = load i32, i32* %3, align 4
  ret i32 %4
}

; Function Attrs: noinline nounwind optnone
define dso_local signext i32 @demo3(i32 noundef signext %0, i32 noundef signext %1) #0 {
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

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+64bit,+a,+c,+d,+f,+m,+relax,-save-restore" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3}
!llvm.ident = !{!4}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"target-abi", !"lp64d"}
!2 = !{i32 7, !"frame-pointer", i32 2}
!3 = !{i32 1, !"SmallDataLimit", i32 8}
!4 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
