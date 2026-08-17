; RUN: opt -load-pass-plugin=%plugin -passes=llgo-lto-pre-globaldce -S %s | FileCheck %s

target datalayout = "e-m:e-p270:32:32-p271:32:32-p272:64:64-i64:64-f80:128-n8:16:32:64-S128"

%reflect.Value = type { ptr }

@query = private unnamed_addr constant [5 x i8] c"Query"
@mutation = private unnamed_addr constant [8 x i8] c"Mutation"
@subscription = private unnamed_addr constant [12 x i8] c"Subscription"

declare void @reflect.Value.MethodByName(ptr sret(%reflect.Value), ptr, i64)
declare { ptr, i1 } @llvm.type.checked.load(ptr, i32, metadata)
declare void @llvm.assume(i1)

define void @shared_sret(ptr %receiver) {
entry:
  %ret = alloca %reflect.Value
  call void @reflect.Value.MethodByName(ptr sret(%reflect.Value) %ret, ptr "llgo.reflect.methodbyname.name"="1" @query, i64 5) #0
  %query.fn = load ptr, ptr %ret
  %query.checked = call { ptr, i1 } @llvm.type.checked.load(ptr %query.fn, i32 0, metadata !"go.method.value.reflect")
  %query.ok = extractvalue { ptr, i1 } %query.checked, 1
  call void @llvm.assume(i1 %query.ok)

  call void @reflect.Value.MethodByName(ptr sret(%reflect.Value) %ret, ptr "llgo.reflect.methodbyname.name"="1" @mutation, i64 8) #0
  %mutation.fn = load ptr, ptr %ret
  %mutation.checked = call { ptr, i1 } @llvm.type.checked.load(ptr %mutation.fn, i32 0, metadata !"go.method.value.reflect")
  %mutation.ok = extractvalue { ptr, i1 } %mutation.checked, 1
  call void @llvm.assume(i1 %mutation.ok)

  call void @reflect.Value.MethodByName(ptr sret(%reflect.Value) %ret, ptr "llgo.reflect.methodbyname.name"="1" @subscription, i64 12) #0
  %subscription.fn = load ptr, ptr %ret
  %subscription.checked = call { ptr, i1 } @llvm.type.checked.load(ptr %subscription.fn, i32 0, metadata !"go.method.value.reflect")
  %subscription.ok = extractvalue { ptr, i1 } %subscription.checked, 1
  call void @llvm.assume(i1 %subscription.ok)
  ret void
}

define void @shared_sret_unknown(ptr %receiver, ptr %dynamic.name, i64 %dynamic.len) {
entry:
  %ret = alloca %reflect.Value
  call void @reflect.Value.MethodByName(ptr sret(%reflect.Value) %ret, ptr "llgo.reflect.methodbyname.name"="1" @query, i64 5) #0
  %query.fn = load ptr, ptr %ret
  %query.checked = call { ptr, i1 } @llvm.type.checked.load(ptr %query.fn, i32 0, metadata !"go.method.value.reflect")
  %query.ok = extractvalue { ptr, i1 } %query.checked, 1
  call void @llvm.assume(i1 %query.ok)

  call void @reflect.Value.MethodByName(ptr sret(%reflect.Value) %ret, ptr "llgo.reflect.methodbyname.name"="1" %dynamic.name, i64 %dynamic.len) #0
  %dynamic.fn = load ptr, ptr %ret
  %dynamic.checked = call { ptr, i1 } @llvm.type.checked.load(ptr %dynamic.fn, i32 0, metadata !"go.method.value.reflect")
  %dynamic.ok = extractvalue { ptr, i1 } %dynamic.checked, 1
  call void @llvm.assume(i1 %dynamic.ok)
  ret void
}

define void @shared_sret_duplicate(ptr %receiver) {
entry:
  %ret = alloca %reflect.Value
  call void @reflect.Value.MethodByName(ptr sret(%reflect.Value) %ret, ptr "llgo.reflect.methodbyname.name"="1" @query, i64 5) #0
  %first.fn = load ptr, ptr %ret
  %first.checked = call { ptr, i1 } @llvm.type.checked.load(ptr %first.fn, i32 0, metadata !"go.method.value.reflect")
  %first.ok = extractvalue { ptr, i1 } %first.checked, 1
  call void @llvm.assume(i1 %first.ok)

  call void @reflect.Value.MethodByName(ptr sret(%reflect.Value) %ret, ptr "llgo.reflect.methodbyname.name"="1" @query, i64 5) #0
  %second.fn = load ptr, ptr %ret
  %second.checked = call { ptr, i1 } @llvm.type.checked.load(ptr %second.fn, i32 0, metadata !"go.method.value.reflect")
  %second.ok = extractvalue { ptr, i1 } %second.checked, 1
  call void @llvm.assume(i1 %second.ok)
  ret void
}

; CHECK-LABEL: define void @shared_sret(
; CHECK: call void @reflect.Value.MethodByName({{.*}}@query
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Query")
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Mutation")
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Subscription")
; CHECK: call void @reflect.Value.MethodByName({{.*}}@mutation
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Query")
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Mutation")
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Subscription")
; CHECK: call void @reflect.Value.MethodByName({{.*}}@subscription
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Query")
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Mutation")
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Subscription")
; CHECK-NOT: !"go.method.value.reflect"

; CHECK-LABEL: define void @shared_sret_unknown(
; CHECK-NOT: !"go.method.value.reflect.Query"
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect")
; CHECK-NOT: !"go.method.value.reflect.Query"
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect")
; CHECK-NOT: !"go.method.value.reflect.Query"
; CHECK: ret void

; CHECK-LABEL: define void @shared_sret_duplicate(
; CHECK: call void @reflect.Value.MethodByName({{.*}}@query
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Query")
; CHECK-NOT: !"go.method.value.reflect.Query"
; CHECK: call void @reflect.Value.MethodByName({{.*}}@query
; CHECK: @llvm.type.checked.load({{.*}}!"go.method.value.reflect.Query")
; CHECK-NOT: !"go.method.value.reflect.Query"
; CHECK: ret void

attributes #0 = { "llgo.reflect.methodbyname"="value" }
