; ModuleID = '../../wrap/struct_int8.c'
source_filename = "../../wrap/struct_int8.c"
target datalayout = "e-m:e-p:32:32-i64:64-n32-S128"
target triple = "riscv32-esp-unknown-elf"

%struct.point1 = type { i8 }
%struct.point2 = type { i8, i8 }
%struct.point3 = type { i8, i8, i8 }
%struct.point4 = type { i8, i8, i8, i8 }
%struct.point5 = type { i8, i8, i8, i8, i8 }
%struct.point6 = type { i8, i8, i8, i8, i8, i8 }
%struct.point7 = type { i8, i8, i8, i8, i8, i8, i8 }
%struct.point8 = type { i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point9 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point10 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point11 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point12 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point13 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point14 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point15 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point16 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point17 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point18 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point19 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }
%struct.point20 = type { i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8, i8 }

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo1(i32 %0) #0 {
  %2 = alloca %struct.point1, align 1
  %3 = alloca %struct.point1, align 1
  %4 = getelementptr inbounds %struct.point1, ptr %3, i32 0, i32 0
  %5 = trunc i32 %0 to i8
  store i8 %5, ptr %4, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 1, i1 false)
  %6 = getelementptr inbounds %struct.point1, ptr %2, i32 0, i32 0
  %7 = load i8, ptr %6, align 1
  %8 = zext i8 %7 to i32
  ret i32 %8
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i32(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo2(i32 %0) #0 {
  %2 = alloca %struct.point2, align 1
  %3 = alloca %struct.point2, align 1
  %4 = alloca i32, align 4
  %5 = trunc i32 %0 to i16
  store i16 %5, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 2, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %4, ptr align 1 %2, i32 2, i1 false)
  %6 = load i32, ptr %4, align 4
  ret i32 %6
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo3(i32 %0) #0 {
  %2 = alloca %struct.point3, align 1
  %3 = alloca %struct.point3, align 1
  %4 = alloca i32, align 4
  %5 = trunc i32 %0 to i24
  store i24 %5, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 3, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %4, ptr align 1 %2, i32 3, i1 false)
  %6 = load i32, ptr %4, align 4
  ret i32 %6
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo4(i32 %0) #0 {
  %2 = alloca %struct.point4, align 1
  %3 = alloca %struct.point4, align 1
  store i32 %0, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 4, i1 false)
  %4 = load i32, ptr %2, align 1
  ret i32 %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo5([2 x i32] %0) #0 {
  %2 = alloca %struct.point5, align 1
  %3 = alloca %struct.point5, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 5, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 5, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 5, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo6([2 x i32] %0) #0 {
  %2 = alloca %struct.point6, align 1
  %3 = alloca %struct.point6, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo7([2 x i32] %0) #0 {
  %2 = alloca %struct.point7, align 1
  %3 = alloca %struct.point7, align 1
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %3, ptr align 4 %4, i32 7, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 7, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 1 %2, i32 7, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo8([2 x i32] %0) #0 {
  %2 = alloca %struct.point8, align 1
  %3 = alloca %struct.point8, align 1
  store [2 x i32] %0, ptr %3, align 1
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %2, ptr align 1 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 1
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo9(ptr dead_on_unwind noalias writable sret(%struct.point9) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 9, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo10(ptr dead_on_unwind noalias writable sret(%struct.point10) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo11(ptr dead_on_unwind noalias writable sret(%struct.point11) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 11, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo12(ptr dead_on_unwind noalias writable sret(%struct.point12) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo13(ptr dead_on_unwind noalias writable sret(%struct.point13) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 13, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo14(ptr dead_on_unwind noalias writable sret(%struct.point14) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 14, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo15(ptr dead_on_unwind noalias writable sret(%struct.point15) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 15, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo16(ptr dead_on_unwind noalias writable sret(%struct.point16) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo17(ptr dead_on_unwind noalias writable sret(%struct.point17) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 17, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo18(ptr dead_on_unwind noalias writable sret(%struct.point18) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 18, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo19(ptr dead_on_unwind noalias writable sret(%struct.point19) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 19, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo20(ptr dead_on_unwind noalias writable sret(%struct.point20) align 1 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 1 %0, ptr align 1 %1, i32 20, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="generic-rv32" "target-features"="+32bit,+a,+c,+m,+relax,+zmmul,-b,-d,-e,-experimental-smmpm,-experimental-smnpm,-experimental-ssnpm,-experimental-sspm,-experimental-ssqosid,-experimental-supm,-experimental-zacas,-experimental-zalasr,-experimental-zicfilp,-experimental-zicfiss,-f,-h,-shcounterenw,-shgatpa,-shtvala,-shvsatpa,-shvstvala,-shvstvecd,-smaia,-smcdeleg,-smcsrind,-smepmp,-smstateen,-ssaia,-ssccfg,-ssccptr,-sscofpmf,-sscounterenw,-sscsrind,-ssstateen,-ssstrict,-sstc,-sstvala,-sstvecd,-ssu64xl,-svade,-svadu,-svbare,-svinval,-svnapot,-svpbmt,-v,-xcvalu,-xcvbi,-xcvbitmanip,-xcvelw,-xcvmac,-xcvmem,-xcvsimd,-xesppie,-xsfcease,-xsfvcp,-xsfvfnrclipxfqf,-xsfvfwmaccqqq,-xsfvqmaccdod,-xsfvqmaccqoq,-xsifivecdiscarddlone,-xsifivecflushdlone,-xtheadba,-xtheadbb,-xtheadbs,-xtheadcmo,-xtheadcondmov,-xtheadfmemidx,-xtheadmac,-xtheadmemidx,-xtheadmempair,-xtheadsync,-xtheadvdot,-xventanacondops,-xwchc,-za128rs,-za64rs,-zaamo,-zabha,-zalrsc,-zama16b,-zawrs,-zba,-zbb,-zbc,-zbkb,-zbkc,-zbkx,-zbs,-zca,-zcb,-zcd,-zce,-zcf,-zcmop,-zcmp,-zcmt,-zdinx,-zfa,-zfbfmin,-zfh,-zfhmin,-zfinx,-zhinx,-zhinxmin,-zic64b,-zicbom,-zicbop,-zicboz,-ziccamoa,-ziccif,-zicclsm,-ziccrse,-zicntr,-zicond,-zicsr,-zifencei,-zihintntl,-zihintpause,-zihpm,-zimop,-zk,-zkn,-zknd,-zkne,-zknh,-zkr,-zks,-zksed,-zksh,-zkt,-ztso,-zvbb,-zvbc,-zve32f,-zve32x,-zve64d,-zve64f,-zve64x,-zvfbfmin,-zvfbfwma,-zvfh,-zvfhmin,-zvkb,-zvkg,-zvkn,-zvknc,-zvkned,-zvkng,-zvknha,-zvknhb,-zvks,-zvksc,-zvksed,-zvksg,-zvksh,-zvkt,-zvl1024b,-zvl128b,-zvl16384b,-zvl2048b,-zvl256b,-zvl32768b,-zvl32b,-zvl4096b,-zvl512b,-zvl64b,-zvl65536b,-zvl8192b" }
attributes #1 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }

!llvm.module.flags = !{!0, !1, !2, !4, !5}
!llvm.ident = !{!6}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"target-abi", !"ilp32"}
!2 = !{i32 6, !"riscv-isa", !3}
!3 = !{!"rv32i2p1_m2p0_a2p1_c2p0_zmmul1p0"}
!4 = !{i32 7, !"frame-pointer", i32 2}
!5 = !{i32 8, !"SmallDataLimit", i32 8}
!6 = !{!"clang version 19.1.2 (https://github.com/espressif/llvm-project 510a078c1ad4aee4460818bcb38ff0ba3fbf6a83)"}
