; ModuleID = '../../wrap/composite.c'
source_filename = "../../wrap/composite.c"
target datalayout = "e-m:e-p:32:32-i64:64-n32-S128"
target triple = "riscv32-esp-unknown-elf"

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
define dso_local i32 @demo_array10(i32 %0) #0 {
  %2 = alloca %struct.array10, align 4
  %3 = alloca %struct.array10, align 4
  %4 = getelementptr inbounds %struct.array10, ptr %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basearray1, ptr %4, i32 0, i32 0
  store i32 %0, ptr %5, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 4, i1 false)
  %6 = getelementptr inbounds %struct.array10, ptr %2, i32 0, i32 0
  %7 = getelementptr inbounds %struct.basearray1, ptr %6, i32 0, i32 0
  %8 = load i32, ptr %7, align 4
  ret i32 %8
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i32(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo_array11([2 x i32] %0) #0 {
  %2 = alloca %struct.array11, align 4
  %3 = alloca %struct.array11, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local i32 @demo_point10(i32 %0) #0 {
  %2 = alloca %struct.point10, align 4
  %3 = alloca %struct.point10, align 4
  %4 = getelementptr inbounds %struct.point10, ptr %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basepoint1, ptr %4, i32 0, i32 0
  store i32 %0, ptr %5, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 4, i1 false)
  %6 = getelementptr inbounds %struct.point10, ptr %2, i32 0, i32 0
  %7 = getelementptr inbounds %struct.basepoint1, ptr %6, i32 0, i32 0
  %8 = load i32, ptr %7, align 4
  ret i32 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo_point11([2 x i32] %0) #0 {
  %2 = alloca %struct.point11, align 4
  %3 = alloca %struct.point11, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo_array20([2 x i32] %0) #0 {
  %2 = alloca %struct.array20, align 4
  %3 = alloca %struct.array20, align 4
  %4 = getelementptr inbounds %struct.array20, ptr %3, i32 0, i32 0
  %5 = getelementptr inbounds %struct.basearray2, ptr %4, i32 0, i32 0
  store [2 x i32] %0, ptr %5, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %6 = getelementptr inbounds %struct.array20, ptr %2, i32 0, i32 0
  %7 = getelementptr inbounds %struct.basearray2, ptr %6, i32 0, i32 0
  %8 = load [2 x i32], ptr %7, align 4
  ret [2 x i32] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_array21(ptr dead_on_unwind noalias writable sret(%struct.array21) align 4 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %1, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo_point20([2 x i32] %0) #0 {
  %2 = alloca %struct.point20, align 4
  %3 = alloca %struct.point20, align 4
  %4 = getelementptr inbounds %struct.point20, ptr %3, i32 0, i32 0
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %5 = getelementptr inbounds %struct.point20, ptr %2, i32 0, i32 0
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo_point21(ptr dead_on_unwind noalias writable sret(%struct.point21) align 4 %0, ptr noundef %1) #0 {
  %3 = alloca ptr, align 4
  %4 = alloca ptr, align 4
  store ptr %0, ptr %3, align 4
  store ptr %1, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %1, i32 12, i1 false)
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
