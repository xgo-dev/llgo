// LITTEST
package main

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"unsafe"
)

//CHECK: @"*{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8" = weak_odr constant { %"{{.*}}/runtime/abi.PtrType", %"{{.*}}/runtime/abi.UncommonType", [3 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -712860747, i8 11, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 25 }, ptr null }, ptr @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8" }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 3, i16 2, i32 24 }, [3 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo1", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo1" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo2", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo2" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 49 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.demo3", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.demo3" }] }, align 8
//CHECK: @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8" = weak_odr constant { %"{{.*}}/runtime/abi.StructType", %"{{.*}}/runtime/abi.UncommonType", [3 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.StructType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 235980794, i8 9, i8 8, i8 8, i8 25, { ptr, ptr } { ptr @"{{.*}}/runtime/internal/runtime.structequal", ptr @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8" }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 25 }, ptr @"*{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8$fields", i64 2, i64 2 } }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 3, i16 2, i32 24 }, [3 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo1", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo1" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo2", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo2" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 49 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.demo3", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *{{.*}}/_testgo/abimethod.T}.demo3" }] }, align 8
//CHECK: @_llgo_int = weak_odr constant %"{{.*}}/runtime/abi.Type" { i64 8, i64 0, i32 -25294021, i8 12, i8 8, i8 8, i8 2, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 3 }, ptr @"*_llgo_int" }, align 8
//CHECK: @"*_llgo_int" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -939606833, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 3 }, ptr null }, ptr @_llgo_int }, align 8
//CHECK: @"*_llgo_{{.*}}/_testgo/abimethod.T" = weak_odr constant { %"{{.*}}/runtime/abi.PtrType", %"{{.*}}/runtime/abi.UncommonType", [3 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -908752194, i8 11, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 6 }, ptr null }, ptr @"_llgo_{{.*}}/_testgo/abimethod.T" }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 3, i16 2, i32 24 }, [3 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.(*T).Demo1", ptr @"{{.*}}/_testgo/abimethod.(*T).Demo1" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.(*T).Demo2", ptr @"{{.*}}/_testgo/abimethod.(*T).Demo2" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 49 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.(*T).demo3", ptr @"{{.*}}/_testgo/abimethod.(*T).demo3" }] }, align 8
//CHECK: @"_llgo_{{.*}}/_testgo/abimethod.T" = weak_odr constant { %"{{.*}}/runtime/abi.StructType", %"{{.*}}/runtime/abi.UncommonType", [1 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.StructType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 0, i32 -666093743, i8 13, i8 8, i8 8, i8 25, { ptr, ptr } { ptr @"{{.*}}/runtime/internal/runtime.structequal", ptr @"_llgo_{{.*}}/_testgo/abimethod.T" }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 6 }, ptr @"*_llgo_{{.*}}/_testgo/abimethod.T" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"{{.*}}/_testgo/abimethod.struct$eovYmOhZg4X0zMSsuscSshndnbbAGvB2E3cyG8E7Y4U$fields", i64 1, i64 1 } }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 1, i16 1, i32 24 }, [1 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.(*T).Demo1", ptr @"{{.*}}/_testgo/abimethod.T.Demo1" }] }, align 8
//CHECK: @"{{.*}}/_testgo/abimethod.struct$eovYmOhZg4X0zMSsuscSshndnbbAGvB2E3cyG8E7Y4U$fields" = weak_odr constant [1 x %"{{.*}}/runtime/abi.StructField"] [%"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 1 }, ptr @_llgo_int, i64 0, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }], align 8
//CHECK: @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 2131144854, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 10 }, ptr @"*_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1805835775, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 10 }, ptr null }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }, align 8
//CHECK: @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA$out" = weak_odr constant [1 x ptr] [ptr @_llgo_int], align 8
//CHECK: @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8$fields" = weak_odr constant [2 x %"{{.*}}/runtime/abi.StructField"] [%"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 1 }, ptr @_llgo_int, i64 0, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }, %"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 1 }, ptr @"*_llgo_{{.*}}/_testgo/abimethod.T", i64 8, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 true }], align 8
//CHECK: @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc" = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 -1807485229, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @12, i64 25 }, ptr @"*_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc$imethods", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 929086049, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @12, i64 25 }, ptr null }, ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc" }, align 8
//CHECK: @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc$imethods" = weak_odr constant [1 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }], align 8
//CHECK: @_llgo_string = weak_odr constant %"{{.*}}/runtime/abi.Type" { i64 16, i64 8, i32 1749264893, i8 4, i8 8, i8 8, i8 24, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.strequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 6 }, ptr @"*_llgo_string" }, align 8
//CHECK: @"*_llgo_string" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1323879264, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 6 }, ptr null }, ptr @_llgo_string }, align 8
//CHECK: @"{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088" = weak_odr constant { %"{{.*}}/runtime/abi.StructType", %"{{.*}}/runtime/abi.UncommonType", [1 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.StructType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 0, i32 179876865, i8 9, i8 8, i8 8, i8 25, { ptr, ptr } { ptr @"{{.*}}/runtime/internal/runtime.structequal", ptr @"{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088" }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 24 }, ptr @"*{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088$fields", i64 2, i64 2 } }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 1, i16 1, i32 24 }, [1 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.Demo1", ptr @"{{.*}}/_testgo/abimethod.struct{m int; {{.*}}/_testgo/abimethod.T}.Demo1" }] }, align 8
//CHECK: @"*{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088" = weak_odr constant { %"{{.*}}/runtime/abi.PtrType", %"{{.*}}/runtime/abi.UncommonType", [3 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -343027978, i8 11, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @16, i64 24 }, ptr null }, ptr @"{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088" }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 3, i16 2, i32 24 }, [3 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.Demo1", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.Demo1" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.Demo2", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.Demo2" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 49 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.demo3", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.demo3" }] }, align 8
//CHECK: @"{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088$fields" = weak_odr constant [2 x %"{{.*}}/runtime/abi.StructField"] [%"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 1 }, ptr @_llgo_int, i64 0, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }, %"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 1 }, ptr @"_llgo_{{.*}}/_testgo/abimethod.T", i64 8, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 true }], align 8
//CHECK: @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4" = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 1090904853, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 25 }, ptr @"*_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4$imethods", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1063382362, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 25 }, ptr null }, ptr @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4" }, align 8
//CHECK: @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4$imethods" = weak_odr constant [1 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }], align 8
//CHECK: @"_llgo_iface$58AxoxqQ6sGUOM73FOqFrXsMlgxkU4HGd-S1Wl-ssYw" = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 541709743, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 38 }, ptr @"*_llgo_iface$58AxoxqQ6sGUOM73FOqFrXsMlgxkU4HGd-S1Wl-ssYw" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_iface$58AxoxqQ6sGUOM73FOqFrXsMlgxkU4HGd-S1Wl-ssYw$imethods", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_iface$58AxoxqQ6sGUOM73FOqFrXsMlgxkU4HGd-S1Wl-ssYw" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 945986433, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 38 }, ptr null }, ptr @"_llgo_iface$58AxoxqQ6sGUOM73FOqFrXsMlgxkU4HGd-S1Wl-ssYw" }, align 8
//CHECK: @"_llgo_iface$58AxoxqQ6sGUOM73FOqFrXsMlgxkU4HGd-S1Wl-ssYw$imethods" = weak_odr constant [2 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }, %"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }], align 8
//CHECK: @"{{.*}}/_testgo/abimethod.iface$kT5SIXt45Cspjl04Bof3DZVSOIltlDo-njpk6KqtZvA" = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 -601152795, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 56 }, ptr @"*{{.*}}/_testgo/abimethod.iface$kT5SIXt45Cspjl04Bof3DZVSOIltlDo-njpk6KqtZvA" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"{{.*}}/_testgo/abimethod.iface$kT5SIXt45Cspjl04Bof3DZVSOIltlDo-njpk6KqtZvA$imethods", i64 3, i64 3 } }, align 8
//CHECK: @"*{{.*}}/_testgo/abimethod.iface$kT5SIXt45Cspjl04Bof3DZVSOIltlDo-njpk6KqtZvA" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 162233315, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 56 }, ptr null }, ptr @"{{.*}}/_testgo/abimethod.iface$kT5SIXt45Cspjl04Bof3DZVSOIltlDo-njpk6KqtZvA" }, align 8
//CHECK: @"{{.*}}/_testgo/abimethod.iface$kT5SIXt45Cspjl04Bof3DZVSOIltlDo-njpk6KqtZvA$imethods" = weak_odr constant [3 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }, %"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 5 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }, %"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 49 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA" }], align 8
//CHECK: @"*{{.*}}/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY" = weak_odr constant { %"{{.*}}/runtime/abi.PtrType", %"{{.*}}/runtime/abi.UncommonType", [28 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1554050967, i8 11, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @27, i64 31 }, ptr null }, ptr @"{{.*}}/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY" }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 28, i16 24, i32 24 }, [28 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @37, i64 9 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Available", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Available" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @38, i64 15 }, ptr @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.AvailableBuffer", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.AvailableBuffer" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @40, i64 5 }, ptr @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Bytes", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Bytes" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @41, i64 3 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Cap", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Cap" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @42, i64 4 }, ptr @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Grow", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Grow" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @44, i64 3 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Len", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Len" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @45, i64 4 }, ptr @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Next", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Next" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @47, i64 4 }, ptr @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Peek", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Peek" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @52, i64 4 }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Read", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Read" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @54, i64 8 }, ptr @"_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadByte", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @56, i64 9 }, ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadBytes", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadBytes" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @58, i64 8 }, ptr @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadFrom", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadFrom" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @62, i64 8 }, ptr @"_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadRune", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @65, i64 10 }, ptr @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadString", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadString" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @67, i64 5 }, ptr @"_llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Reset", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Reset" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @69, i64 6 }, ptr @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.String", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.String" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @70, i64 8 }, ptr @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Truncate", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Truncate" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @71, i64 10 }, ptr @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadByte", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @73, i64 10 }, ptr @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadRune", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @74, i64 5 }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Write", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Write" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @75, i64 9 }, ptr @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteByte", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @77, i64 9 }, ptr @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteRune", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @79, i64 11 }, ptr @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteString", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteString" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @81, i64 7 }, ptr @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteTo", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteTo" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @85, i64 11 }, ptr @"_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.empty", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.empty" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @89, i64 10 }, ptr @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.grow", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.grow" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @92, i64 15 }, ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.readSlice", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.readSlice" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @94, i64 22 }, ptr @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.tryGrowByReslice", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.tryGrowByReslice" }] }, align 8
//CHECK: @"{{.*}}/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY" = weak_odr constant { %"{{.*}}/runtime/abi.StructType", %"{{.*}}/runtime/abi.UncommonType", [28 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.StructType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 -1137763463, i8 9, i8 8, i8 8, i8 25, { ptr, ptr } { ptr @"{{.*}}/runtime/internal/runtime.structequal", ptr @"{{.*}}/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY" }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @27, i64 31 }, ptr @"*{{.*}}/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"{{.*}}/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY$fields", i64 2, i64 2 } }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 28, i16 24, i32 24 }, [28 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @37, i64 9 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Available", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Available" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @38, i64 15 }, ptr @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.AvailableBuffer", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.AvailableBuffer" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @40, i64 5 }, ptr @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Bytes", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Bytes" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @41, i64 3 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Cap", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Cap" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @42, i64 4 }, ptr @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Grow", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Grow" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @44, i64 3 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Len", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Len" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @45, i64 4 }, ptr @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Next", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Next" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @47, i64 4 }, ptr @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Peek", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Peek" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @52, i64 4 }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Read", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Read" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @54, i64 8 }, ptr @"_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadByte", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @56, i64 9 }, ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadBytes", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadBytes" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @58, i64 8 }, ptr @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadFrom", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadFrom" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @62, i64 8 }, ptr @"_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadRune", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @65, i64 10 }, ptr @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadString", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadString" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @67, i64 5 }, ptr @"_llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Reset", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Reset" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @69, i64 6 }, ptr @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.String", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.String" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @70, i64 8 }, ptr @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Truncate", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Truncate" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @71, i64 10 }, ptr @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadByte", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @73, i64 10 }, ptr @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadRune", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @74, i64 5 }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Write", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Write" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @75, i64 9 }, ptr @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteByte", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @77, i64 9 }, ptr @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteRune", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @79, i64 11 }, ptr @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteString", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteString" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @81, i64 7 }, ptr @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteTo", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteTo" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @85, i64 11 }, ptr @"_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.empty", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.empty" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @89, i64 10 }, ptr @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.grow", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.grow" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @92, i64 15 }, ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.readSlice", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.readSlice" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @94, i64 22 }, ptr @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M", ptr @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.tryGrowByReslice", ptr @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.tryGrowByReslice" }] }, align 8
//CHECK: @"*_llgo_bytes.Buffer" = weak_odr constant { %"{{.*}}/runtime/abi.PtrType", %"{{.*}}/runtime/abi.UncommonType", [28 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 258663788, i8 11, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @29, i64 12 }, ptr null }, ptr @_llgo_bytes.Buffer }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @30, i64 5 }, i16 28, i16 24, i32 24 }, [28 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @37, i64 9 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"bytes.(*Buffer).Available", ptr @"bytes.(*Buffer).Available" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @38, i64 15 }, ptr @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY", ptr @"bytes.(*Buffer).AvailableBuffer", ptr @"bytes.(*Buffer).AvailableBuffer" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @40, i64 5 }, ptr @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY", ptr @"bytes.(*Buffer).Bytes", ptr @"bytes.(*Buffer).Bytes" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @41, i64 3 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"bytes.(*Buffer).Cap", ptr @"bytes.(*Buffer).Cap" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @42, i64 4 }, ptr @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA", ptr @"bytes.(*Buffer).Grow", ptr @"bytes.(*Buffer).Grow" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @44, i64 3 }, ptr @"_llgo_func$ETeB8WwW04JEq0ztcm-XPTJtuYvtpkjIsAc0-2NT9zA", ptr @"bytes.(*Buffer).Len", ptr @"bytes.(*Buffer).Len" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @45, i64 4 }, ptr @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY", ptr @"bytes.(*Buffer).Next", ptr @"bytes.(*Buffer).Next" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @47, i64 4 }, ptr @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ", ptr @"bytes.(*Buffer).Peek", ptr @"bytes.(*Buffer).Peek" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @52, i64 4 }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk", ptr @"bytes.(*Buffer).Read", ptr @"bytes.(*Buffer).Read" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @54, i64 8 }, ptr @"_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ", ptr @"bytes.(*Buffer).ReadByte", ptr @"bytes.(*Buffer).ReadByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @56, i64 9 }, ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0", ptr @"bytes.(*Buffer).ReadBytes", ptr @"bytes.(*Buffer).ReadBytes" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @58, i64 8 }, ptr @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8", ptr @"bytes.(*Buffer).ReadFrom", ptr @"bytes.(*Buffer).ReadFrom" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @62, i64 8 }, ptr @"_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y", ptr @"bytes.(*Buffer).ReadRune", ptr @"bytes.(*Buffer).ReadRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @65, i64 10 }, ptr @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o", ptr @"bytes.(*Buffer).ReadString", ptr @"bytes.(*Buffer).ReadString" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @67, i64 5 }, ptr @"_llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac", ptr @"bytes.(*Buffer).Reset", ptr @"bytes.(*Buffer).Reset" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @69, i64 6 }, ptr @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to", ptr @"bytes.(*Buffer).String", ptr @"bytes.(*Buffer).String" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @70, i64 8 }, ptr @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA", ptr @"bytes.(*Buffer).Truncate", ptr @"bytes.(*Buffer).Truncate" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @71, i64 10 }, ptr @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w", ptr @"bytes.(*Buffer).UnreadByte", ptr @"bytes.(*Buffer).UnreadByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @73, i64 10 }, ptr @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w", ptr @"bytes.(*Buffer).UnreadRune", ptr @"bytes.(*Buffer).UnreadRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @74, i64 5 }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk", ptr @"bytes.(*Buffer).Write", ptr @"bytes.(*Buffer).Write" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @75, i64 9 }, ptr @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg", ptr @"bytes.(*Buffer).WriteByte", ptr @"bytes.(*Buffer).WriteByte" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @77, i64 9 }, ptr @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw", ptr @"bytes.(*Buffer).WriteRune", ptr @"bytes.(*Buffer).WriteRune" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @79, i64 11 }, ptr @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw", ptr @"bytes.(*Buffer).WriteString", ptr @"bytes.(*Buffer).WriteString" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @81, i64 7 }, ptr @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M", ptr @"bytes.(*Buffer).WriteTo", ptr @"bytes.(*Buffer).WriteTo" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @85, i64 11 }, ptr @"_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk", ptr @"bytes.(*Buffer).empty", ptr @"bytes.(*Buffer).empty" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @89, i64 10 }, ptr @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU", ptr @"bytes.(*Buffer).grow", ptr @"bytes.(*Buffer).grow" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @92, i64 15 }, ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0", ptr @"bytes.(*Buffer).readSlice", ptr @"bytes.(*Buffer).readSlice" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @94, i64 22 }, ptr @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M", ptr @"bytes.(*Buffer).tryGrowByReslice", ptr @"bytes.(*Buffer).tryGrowByReslice" }] }, align 8
//CHECK: @_llgo_bytes.Buffer = weak_odr constant { %"{{.*}}/runtime/abi.StructType", %"{{.*}}/runtime/abi.UncommonType", [0 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.StructType" { %"{{.*}}/runtime/abi.Type" { i64 40, i64 0, i32 661676552, i8 5, i8 8, i8 8, i8 25, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @29, i64 12 }, ptr @"*_llgo_bytes.Buffer" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @30, i64 5 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"bytes.struct$8M6lRFZ7Fk2XCr2laNI9Y7uQtk2A8VDBrezMuq2Fkuo$fields", i64 3, i64 3 } }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @30, i64 5 }, i16 0, i16 0, i32 24 }, [0 x %"{{.*}}/runtime/abi.Method"] zeroinitializer }, align 8
//CHECK: @"[]_llgo_uint8" = weak_odr constant %"{{.*}}/runtime/abi.SliceType" { %"{{.*}}/runtime/abi.Type" { i64 24, i64 8, i32 370346748, i8 0, i8 8, i8 8, i8 23, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @32, i64 7 }, ptr @"*[]_llgo_uint8" }, ptr @_llgo_uint8 }, align 8
//CHECK: @"*[]_llgo_uint8" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -2143776929, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @32, i64 7 }, ptr null }, ptr @"[]_llgo_uint8" }, align 8
//CHECK: @_llgo_uint8 = weak_odr constant %"{{.*}}/runtime/abi.Type" { i64 1, i64 0, i32 269156761, i8 12, i8 1, i8 1, i8 8, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @33, i64 5 }, ptr @"*_llgo_uint8" }, align 8
//CHECK: @"*_llgo_uint8" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1277858201, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @33, i64 5 }, ptr null }, ptr @_llgo_uint8 }, align 8
//CHECK: @_llgo_bytes.readOp = weak_odr constant { %"{{.*}}/runtime/abi.Type", %"{{.*}}/runtime/abi.UncommonType", [0 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.Type" { i64 1, i64 0, i32 1507423333, i8 13, i8 1, i8 1, i8 3, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @36, i64 12 }, ptr @"*_llgo_bytes.readOp" }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @30, i64 5 }, i16 0, i16 0, i32 24 }, [0 x %"{{.*}}/runtime/abi.Method"] zeroinitializer }, align 8
//CHECK: @"*_llgo_bytes.readOp" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1082688598, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @36, i64 12 }, ptr null }, ptr @_llgo_bytes.readOp }, align 8
//CHECK: @"bytes.struct$8M6lRFZ7Fk2XCr2laNI9Y7uQtk2A8VDBrezMuq2Fkuo$fields" = weak_odr constant [3 x %"{{.*}}/runtime/abi.StructField"] [%"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @31, i64 3 }, ptr @"[]_llgo_uint8", i64 0, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }, %"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @34, i64 3 }, ptr @_llgo_int, i64 24, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }, %"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @35, i64 8 }, ptr @_llgo_bytes.readOp, i64 32, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }], align 8
//CHECK: @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -153447421, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @39, i64 14 }, ptr @"*_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1574747178, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @39, i64 14 }, ptr null }, ptr @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY" }, align 8
//CHECK: @"_llgo_func$Z_-7GWzB37LCYRTQLsSYmEihg_hqBK8o_GbT88pqnPY$out" = weak_odr constant [1 x ptr] [ptr @"[]_llgo_uint8"], align 8
//CHECK: @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -637187458, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @43, i64 9 }, ptr @"*_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer }, align 8
//CHECK: @"*_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 735356155, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @43, i64 9 }, ptr null }, ptr @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA" }, align 8
//CHECK: @"_llgo_func$VZ-8VPNF1RaLICwxc1Ghn7BbgyFX3v762OCdx127EkA$in" = weak_odr constant [1 x ptr] [ptr @_llgo_int], align 8
//CHECK: @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -875118098, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @46, i64 17 }, ptr @"*_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1809495648, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @46, i64 17 }, ptr null }, ptr @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY" }, align 8
//CHECK: @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY$in" = weak_odr constant [1 x ptr] [ptr @_llgo_int], align 8
//CHECK: @"_llgo_func$d4kMA_oCkLwnd1j8nVlv1hwRarEVuCIrDCpnHhDz9UY$out" = weak_odr constant [1 x ptr] [ptr @"[]_llgo_uint8"], align 8
//CHECK: @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1159315658, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @48, i64 26 }, ptr @"*_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -907349979, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @48, i64 26 }, ptr null }, ptr @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ" }, align 8
//CHECK: @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ$in" = weak_odr constant [1 x ptr] [ptr @_llgo_int], align 8
//CHECK: @_llgo_error = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 -1462738452, i8 4, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @49, i64 5 }, ptr @"*_llgo_error" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_iface$Fh8eUJ-Gw4e6TYuajcFIOSCuqSPKAt5nS4ow7xeGXEU$imethods", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_error" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1621558991, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @49, i64 5 }, ptr null }, ptr @_llgo_error }, align 8
//CHECK: @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1419376263, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @51, i64 13 }, ptr @"*_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1900367307, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @51, i64 13 }, ptr null }, ptr @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to" }, align 8
//CHECK: @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to$out" = weak_odr constant [1 x ptr] [ptr @_llgo_string], align 8
//CHECK: @"_llgo_iface$Fh8eUJ-Gw4e6TYuajcFIOSCuqSPKAt5nS4ow7xeGXEU$imethods" = weak_odr constant [1 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @50, i64 5 }, ptr @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to" }], align 8
//CHECK: @"_llgo_func$c8Vvy7bpifdsF4PDKWnpU2KNuhLPRr7XBIDM-4p45sQ$out" = weak_odr constant [2 x ptr] [ptr @"[]_llgo_uint8", ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -58533757, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @53, i64 26 }, ptr @"*_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1244675479, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @53, i64 26 }, ptr null }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk" }, align 8
//CHECK: @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk$in" = weak_odr constant [1 x ptr] [ptr @"[]_llgo_uint8"], align 8
//CHECK: @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk$out" = weak_odr constant [2 x ptr] [ptr @_llgo_int, ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1499372428, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @55, i64 21 }, ptr @"*_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1164205677, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @55, i64 21 }, ptr null }, ptr @"_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ" }, align 8
//CHECK: @"_llgo_func$lukqSsfDYBoIp_R8GMojGkZnrYDqaq2iHn8RkCjW7iQ$out" = weak_odr constant [2 x ptr] [ptr @_llgo_uint8, ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1659702664, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @57, i64 28 }, ptr @"*_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 2010891442, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @57, i64 28 }, ptr null }, ptr @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0" }, align 8
//CHECK: @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0$in" = weak_odr constant [1 x ptr] [ptr @_llgo_uint8], align 8
//CHECK: @"_llgo_func$aJkaU3jhXr0Q2QraTe2_TTdupeMMW2MD66UwBxynRM0$out" = weak_odr constant [2 x ptr] [ptr @"[]_llgo_uint8", ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1635781323, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @59, i64 30 }, ptr @"*_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1930979199, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @59, i64 30 }, ptr null }, ptr @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8" }, align 8
//CHECK: @_llgo_io.Reader = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 -1885455791, i8 4, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @60, i64 9 }, ptr @"*_llgo_io.Reader" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_iface$uycIKA3bbxRhudEjW1hHKWKdLqHQsCVy8NdW1bkQmNw$imethods", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_io.Reader" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 760453616, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @60, i64 9 }, ptr null }, ptr @_llgo_io.Reader }, align 8
//CHECK: @"_llgo_iface$uycIKA3bbxRhudEjW1hHKWKdLqHQsCVy8NdW1bkQmNw$imethods" = weak_odr constant [1 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @52, i64 4 }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk" }], align 8
//CHECK: @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8$in" = weak_odr constant [1 x ptr] [ptr @_llgo_io.Reader], align 8
//CHECK: @_llgo_int64 = weak_odr constant %"{{.*}}/runtime/abi.Type" { i64 8, i64 0, i32 394795202, i8 12, i8 8, i8 8, i8 6, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @61, i64 5 }, ptr @"*_llgo_int64" }, align 8
//CHECK: @"*_llgo_int64" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1901231210, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @61, i64 5 }, ptr null }, ptr @_llgo_int64 }, align 8
//CHECK: @"_llgo_func$uVmBDI0DMcrui3Q9y-g_hbtVN8JckQ18V2wmO5_G7A8$out" = weak_odr constant [2 x ptr] [ptr @_llgo_int64, ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1043083527, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @63, i64 26 }, ptr @"*_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y$out", i64 3, i64 3 } }, align 8
//CHECK: @"*_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 746645372, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @63, i64 26 }, ptr null }, ptr @"_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y" }, align 8
//CHECK: @_llgo_int32 = weak_odr constant %"{{.*}}/runtime/abi.Type" { i64 4, i64 0, i32 1448558410, i8 12, i8 4, i8 4, i8 5, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal32", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @64, i64 5 }, ptr @"*_llgo_int32" }, align 8
//CHECK: @"*_llgo_int32" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -38689692, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @64, i64 5 }, ptr null }, ptr @_llgo_int32 }, align 8
//CHECK: @"_llgo_func$q-bw-_pPYBCXnr1TXIF8sOD4fVVzzIlpHqD-A13AB4Y$out" = weak_odr constant [3 x ptr] [ptr @_llgo_int32, ptr @_llgo_int, ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 2138446355, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @66, i64 27 }, ptr @"*_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1932918037, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @66, i64 27 }, ptr null }, ptr @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o" }, align 8
//CHECK: @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o$in" = weak_odr constant [1 x ptr] [ptr @_llgo_uint8], align 8
//CHECK: @"_llgo_func$TBlCn7YTQdraI1HMiBWmkrqIGG-8UgD1UVyJy62Z_0o$out" = weak_odr constant [2 x ptr] [ptr @_llgo_string, ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1790696805, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @68, i64 6 }, ptr @"*_llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer }, align 8
//CHECK: @"*_llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -130179135, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @68, i64 6 }, ptr null }, ptr @"_llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac" }, align 8
//CHECK: @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1183719404, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @72, i64 12 }, ptr @"*_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1571491799, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @72, i64 12 }, ptr null }, ptr @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w" }, align 8
//CHECK: @"_llgo_func$8rsrSd_r3UHd_2DiYTyaOKR7BYkei4zw5ysG35KF38w$out" = weak_odr constant [1 x ptr] [ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1226479232, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @76, i64 17 }, ptr @"*_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 513101056, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @76, i64 17 }, ptr null }, ptr @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg" }, align 8
//CHECK: @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg$in" = weak_odr constant [1 x ptr] [ptr @_llgo_uint8], align 8
//CHECK: @"_llgo_func$w4tN9iibS_UimF5vLUWoKP0uAk2tJZF26VqETo_8LVg$out" = weak_odr constant [1 x ptr] [ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1027172853, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @78, i64 24 }, ptr @"*_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -841225112, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @78, i64 24 }, ptr null }, ptr @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw" }, align 8
//CHECK: @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw$in" = weak_odr constant [1 x ptr] [ptr @_llgo_int32], align 8
//CHECK: @"_llgo_func$uf8yw1UkUdbDuCneSpNKIq_NThWIEVE7f1IYfJGz_bw$out" = weak_odr constant [2 x ptr] [ptr @_llgo_int, ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -183202291, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @80, i64 25 }, ptr @"*_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1229992101, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @80, i64 25 }, ptr null }, ptr @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw" }, align 8
//CHECK: @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw$in" = weak_odr constant [1 x ptr] [ptr @_llgo_string], align 8
//CHECK: @"_llgo_func$thH5FBpdXzJNnCpSfiLU5ItTntFU6LWp0RJhDm2XJjw$out" = weak_odr constant [2 x ptr] [ptr @_llgo_int, ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1753174026, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @82, i64 30 }, ptr @"*_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1055881531, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @82, i64 30 }, ptr null }, ptr @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M" }, align 8
//CHECK: @_llgo_io.Writer = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 1423852385, i8 4, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @83, i64 9 }, ptr @"*_llgo_io.Writer" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_iface$kr1iSWwMezh0B9LdQN0MhEZUNZvBlHPhlst95jAyxE0$imethods", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_io.Writer" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1477879550, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @83, i64 9 }, ptr null }, ptr @_llgo_io.Writer }, align 8
//CHECK: @"_llgo_iface$kr1iSWwMezh0B9LdQN0MhEZUNZvBlHPhlst95jAyxE0$imethods" = weak_odr constant [1 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @74, i64 5 }, ptr @"_llgo_func$G2hch9Iy9DrhKKsg70PbL54bK-XSl-1IUUORN17J2Dk" }], align 8
//CHECK: @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M$in" = weak_odr constant [1 x ptr] [ptr @_llgo_io.Writer], align 8
//CHECK: @"_llgo_func$vSv85k0UY6JWccAc3T-lvdCx9J-4GM-oZC9zGLrxW1M$out" = weak_odr constant [2 x ptr] [ptr @_llgo_int64, ptr @_llgo_error], align 8
//CHECK: @"_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -541022001, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @86, i64 11 }, ptr @"*_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -367308996, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @86, i64 11 }, ptr null }, ptr @"_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk" }, align 8
//CHECK: @_llgo_bool = weak_odr constant %"{{.*}}/runtime/abi.Type" { i64 1, i64 0, i32 554183389, i8 12, i8 1, i8 1, i8 1, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @87, i64 4 }, ptr @"*_llgo_bool" }, align 8
//CHECK: @"*_llgo_bool" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1896950390, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @87, i64 4 }, ptr null }, ptr @_llgo_bool }, align 8
//CHECK: @"_llgo_func$YHeRw3AOvQtzv982-ZO3Yn8vh3Fx89RM3VvI8E4iKVk$out" = weak_odr constant [1 x ptr] [ptr @_llgo_bool], align 8
//CHECK: @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1134531106, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @90, i64 13 }, ptr @"*_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1763581361, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @90, i64 13 }, ptr null }, ptr @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU" }, align 8
//CHECK: @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU$in" = weak_odr constant [1 x ptr] [ptr @_llgo_int], align 8
//CHECK: @"_llgo_func$ekGNsrYBSzltfAjxbl6T8H6Yq8j16wzqS3nDj2xxGMU$out" = weak_odr constant [1 x ptr] [ptr @_llgo_int], align 8
//CHECK: @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 559523889, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @95, i64 21 }, ptr @"*_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M$out", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1842714480, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @95, i64 21 }, ptr null }, ptr @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M" }, align 8
//CHECK: @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M$in" = weak_odr constant [1 x ptr] [ptr @_llgo_int], align 8
//CHECK: @"_llgo_func$qVJ5SH6qhXP_h0AM41vpBGzQEMp-fQIfvwQEJy5NI8M$out" = weak_odr constant [2 x ptr] [ptr @_llgo_int, ptr @_llgo_bool], align 8
//CHECK: @"{{.*}}/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY$fields" = weak_odr constant [2 x %"{{.*}}/runtime/abi.StructField"] [%"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 1 }, ptr @_llgo_int, i64 0, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }, %"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @28, i64 6 }, ptr @"*_llgo_bytes.Buffer", i64 8, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 true }], align 8
//CHECK: @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U" = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 -195205541, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @96, i64 29 }, ptr @"*_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U$imethods", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 876051709, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @96, i64 29 }, ptr null }, ptr @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U" }, align 8
//CHECK: @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U$imethods" = weak_odr constant [1 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @69, i64 6 }, ptr @"_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to" }], align 8
//CHECK: @"*_llgo_{{.*}}/_testgo/abimethod.Pointer[any]" = weak_odr constant { %"{{.*}}/runtime/abi.PtrType", %"{{.*}}/runtime/abi.UncommonType", [2 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1798654480, i8 11, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @98, i64 26 }, ptr null }, ptr @"_llgo_{{.*}}/_testgo/abimethod.Pointer[any]" }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 2, i16 2, i32 24 }, [2 x %"{{.*}}/runtime/abi.Method"] [%"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @105, i64 4 }, ptr @"_llgo_func$oqZ09zjnrQRdlivNw60EomwRoboDQbCk5_Y4MGDpQMQ", ptr @"{{.*}}/_testgo/abimethod.(*Pointer[any]).Load", ptr @"{{.*}}/_testgo/abimethod.(*Pointer[any]).Load" }, %"{{.*}}/runtime/abi.Method" { %"{{.*}}/runtime/internal/runtime.String" { ptr @107, i64 5 }, ptr @"_llgo_func$Y8Pl6IHgSDuynhMRTXPlqFo9zl71SSTuMe0Wi5m8eWw", ptr @"{{.*}}/_testgo/abimethod.(*Pointer[any]).Store", ptr @"{{.*}}/_testgo/abimethod.(*Pointer[any]).Store" }] }, align 8
//CHECK: @"_llgo_{{.*}}/_testgo/abimethod.Pointer[any]" = weak_odr constant { %"{{.*}}/runtime/abi.StructType", %"{{.*}}/runtime/abi.UncommonType", [0 x %"{{.*}}/runtime/abi.Method"] } { %"{{.*}}/runtime/abi.StructType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -858197093, i8 5, i8 8, i8 8, i8 25, { ptr, ptr } { ptr @"{{.*}}/runtime/internal/runtime.structequal", ptr @"_llgo_{{.*}}/_testgo/abimethod.Pointer[any]" }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @98, i64 26 }, ptr @"*_llgo_{{.*}}/_testgo/abimethod.Pointer[any]" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"{{.*}}/_testgo/abimethod.struct$WF_Ikp6H-8IyobSlL849gp6AslXPTyT8oKnkzqHD2NA$fields", i64 2, i64 2 } }, %"{{.*}}/runtime/abi.UncommonType" { %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, i16 0, i16 0, i32 24 }, [0 x %"{{.*}}/runtime/abi.Method"] zeroinitializer }, align 8
//CHECK: @"[0]*_llgo_any" = weak_odr constant %"{{.*}}/runtime/abi.ArrayType" { %"{{.*}}/runtime/abi.Type" { i64 0, i64 0, i32 -1235244625, i8 8, i8 8, i8 8, i8 17, { ptr, ptr } { ptr @"{{.*}}/runtime/internal/runtime.arrayequal", ptr @"[0]*_llgo_any" }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @100, i64 16 }, ptr @"*[0]*_llgo_any" }, ptr @"*_llgo_any", ptr @"[]*_llgo_any", i64 0 }, align 8
//CHECK: @"*[0]*_llgo_any" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1487017406, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @100, i64 16 }, ptr null }, ptr @"[0]*_llgo_any" }, align 8
//CHECK: @"*_llgo_any" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1741196194, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @101, i64 12 }, ptr null }, ptr @_llgo_any }, align 8
//CHECK: @_llgo_any = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 1376530322, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @101, i64 12 }, ptr @"*_llgo_any" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer }, align 8
//CHECK: @"[]*_llgo_any" = weak_odr constant %"{{.*}}/runtime/abi.SliceType" { %"{{.*}}/runtime/abi.Type" { i64 24, i64 8, i32 1393026359, i8 0, i8 8, i8 8, i8 23, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @102, i64 15 }, ptr @"*[]*_llgo_any" }, ptr @"*_llgo_any" }, align 8
//CHECK: @"*[]*_llgo_any" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1459791968, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @102, i64 15 }, ptr null }, ptr @"[]*_llgo_any" }, align 8
//CHECK: @_llgo_Pointer = weak_odr constant %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 507576105, i8 12, i8 8, i8 8, i8 58, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @104, i64 14 }, ptr @"*_llgo_Pointer" }, align 8
//CHECK: @"*_llgo_Pointer" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1134390089, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @104, i64 14 }, ptr null }, ptr @_llgo_Pointer }, align 8
//CHECK: @"{{.*}}/_testgo/abimethod.struct$WF_Ikp6H-8IyobSlL849gp6AslXPTyT8oKnkzqHD2NA$fields" = weak_odr constant [2 x %"{{.*}}/runtime/abi.StructField"] [%"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @99, i64 1 }, ptr @"[0]*_llgo_any", i64 0, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }, %"{{.*}}/runtime/abi.StructField" { %"{{.*}}/runtime/internal/runtime.String" { ptr @103, i64 1 }, ptr @_llgo_Pointer, i64 0, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, i1 false }], align 8
//CHECK: @"_llgo_func$oqZ09zjnrQRdlivNw60EomwRoboDQbCk5_Y4MGDpQMQ" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 1908794164, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @106, i64 20 }, ptr @"*_llgo_func$oqZ09zjnrQRdlivNw60EomwRoboDQbCk5_Y4MGDpQMQ" }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$oqZ09zjnrQRdlivNw60EomwRoboDQbCk5_Y4MGDpQMQ$out", i64 1, i64 1 } }, align 8
//CHECK: @"*_llgo_func$oqZ09zjnrQRdlivNw60EomwRoboDQbCk5_Y4MGDpQMQ" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 709309578, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @106, i64 20 }, ptr null }, ptr @"_llgo_func$oqZ09zjnrQRdlivNw60EomwRoboDQbCk5_Y4MGDpQMQ" }, align 8
//CHECK: @"_llgo_func$oqZ09zjnrQRdlivNw60EomwRoboDQbCk5_Y4MGDpQMQ$out" = weak_odr constant [1 x ptr] [ptr @"*_llgo_any"], align 8
//CHECK: @"_llgo_func$Y8Pl6IHgSDuynhMRTXPlqFo9zl71SSTuMe0Wi5m8eWw" = weak_odr constant %"{{.*}}/runtime/abi.FuncType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1005812159, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @108, i64 19 }, ptr @"*_llgo_func$Y8Pl6IHgSDuynhMRTXPlqFo9zl71SSTuMe0Wi5m8eWw" }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_func$Y8Pl6IHgSDuynhMRTXPlqFo9zl71SSTuMe0Wi5m8eWw$in", i64 1, i64 1 }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer }, align 8
//CHECK: @"*_llgo_func$Y8Pl6IHgSDuynhMRTXPlqFo9zl71SSTuMe0Wi5m8eWw" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -1778721426, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @108, i64 19 }, ptr null }, ptr @"_llgo_func$Y8Pl6IHgSDuynhMRTXPlqFo9zl71SSTuMe0Wi5m8eWw" }, align 8
//CHECK: @"_llgo_func$Y8Pl6IHgSDuynhMRTXPlqFo9zl71SSTuMe0Wi5m8eWw$in" = weak_odr constant [1 x ptr] [ptr @"*_llgo_any"], align 8
//CHECK: @"_llgo_iface$uinGjIxPTfzB5e5h5gH-0VIvLl5rQdJ_yx2UsrxQqds" = weak_odr constant %"{{.*}}/runtime/abi.InterfaceType" { %"{{.*}}/runtime/abi.Type" { i64 16, i64 16, i32 414101712, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @109, i64 56 }, ptr @"*_llgo_iface$uinGjIxPTfzB5e5h5gH-0VIvLl5rQdJ_yx2UsrxQqds" }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 43 }, %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"_llgo_iface$uinGjIxPTfzB5e5h5gH-0VIvLl5rQdJ_yx2UsrxQqds$imethods", i64 2, i64 2 } }, align 8
//CHECK: @"*_llgo_iface$uinGjIxPTfzB5e5h5gH-0VIvLl5rQdJ_yx2UsrxQqds" = weak_odr constant %"{{.*}}/runtime/abi.PtrType" { %"{{.*}}/runtime/abi.Type" { i64 8, i64 8, i32 -756921395, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"{{.*}}/runtime/internal/runtime.String" { ptr @109, i64 56 }, ptr null }, ptr @"_llgo_iface$uinGjIxPTfzB5e5h5gH-0VIvLl5rQdJ_yx2UsrxQqds" }, align 8
//CHECK: @"_llgo_iface$uinGjIxPTfzB5e5h5gH-0VIvLl5rQdJ_yx2UsrxQqds$imethods" = weak_odr constant [2 x %"{{.*}}/runtime/abi.Imethod"] [%"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @105, i64 4 }, ptr @"_llgo_func$oqZ09zjnrQRdlivNw60EomwRoboDQbCk5_Y4MGDpQMQ" }, %"{{.*}}/runtime/abi.Imethod" { %"{{.*}}/runtime/internal/runtime.String" { ptr @107, i64 5 }, ptr @"_llgo_func$Y8Pl6IHgSDuynhMRTXPlqFo9zl71SSTuMe0Wi5m8eWw" }], align 8

type T struct {
	n int
}

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.T.Demo1"(%"{{.*}}/_testgo/abimethod.T" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/_testgo/abimethod.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}/_testgo/abimethod.T" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.(*T).Demo1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load %"{{.*}}/_testgo/abimethod.T", ptr %0, align 8
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/_testgo/abimethod.T.Demo1"(%"{{.*}}/_testgo/abimethod.T" %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func (t T) Demo1() int {
	return t.n
}

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.(*T).Demo2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:  %1 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:  %2 = load i64, ptr %1, align 8
// CHECK-NEXT:  ret i64 %2
// CHECK-NEXT: }
func (t *T) Demo2() int {
	return t.n
}

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.(*T).demo3"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:  %1 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:  %2 = load i64, ptr %1, align 8
// CHECK-NEXT:  ret i64 %2
// CHECK-NEXT: }
func (t *T) demo3() int {
	return t.n
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testGeneric"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testNamed1"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testNamed2"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testNamed3"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymous1"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymous2"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymous3"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymous4"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymous5"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymous6"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymous7"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymous8"()
// CHECK-NEXT:   call void @"{{.*}}/_testgo/abimethod.testAnonymousBuffer"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	testGeneric()
	testNamed1()
	testNamed2()
	testNamed3()
	testAnonymous1()
	testAnonymous2()
	testAnonymous3()
	testAnonymous4()
	testAnonymous5()
	testAnonymous6()
	testAnonymous7()
	testAnonymous8()
	testAnonymousBuffer()
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymous1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"*{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %5, 0
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %6, ptr %0, 1
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %7)
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %7, 0
// CHECK-NEXT:   %10 = getelementptr ptr, ptr %9, i64 3
// CHECK-NEXT:   %11 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } undef, ptr %11, 0
// CHECK-NEXT:   %13 = insertvalue { ptr, ptr } %12, ptr %8, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %13, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %13, 0
// CHECK-NEXT:   %16 = call i64 %15(ptr %14)
// CHECK-NEXT:   %17 = icmp ne i64 %16, 100
// CHECK-NEXT:   br i1 %17, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 20 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymous1() {
	var s I = &struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous1 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymous2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = load { i64, ptr }, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, ptr } %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 100
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @15, i64 20 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymous2() {
	var s I = struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous2 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymous3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, %"{{.*}}/_testgo/abimethod.T" }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = load { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, %"{{.*}}/_testgo/abimethod.T" } %4, ptr %5, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088")
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %6, 0
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %7, ptr %5, 1
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %8)
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %8, 0
// CHECK-NEXT:   %11 = getelementptr ptr, ptr %10, i64 3
// CHECK-NEXT:   %12 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %13 = insertvalue { ptr, ptr } undef, ptr %12, 0
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } %13, ptr %9, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %14, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %14, 0
// CHECK-NEXT:   %17 = call i64 %16(ptr %15)
// CHECK-NEXT:   %18 = icmp ne i64 %17, 100
// CHECK-NEXT:   br i1 %18, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @17, i64 20 }, ptr %19, align 8
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %19, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %20)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymous3() {
	var s I = struct {
		m int
		T
	}{10, T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous3 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymous4"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"*{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %0, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call i64 %14(ptr %13)
// CHECK-NEXT:   %16 = icmp ne i64 %15, 100
// CHECK-NEXT:   br i1 %16, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 20 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymous4() {
	var s I = &struct {
		m int
		T
	}{10, T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous4 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymous5"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4", ptr @"*{{.*}}/_testgo/abimethod.struct$F3FioEGWwXQRUdV6xoxVUEDjRNgBQIpL0XIyBECp088")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %0, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call i64 %14(ptr %13)
// CHECK-NEXT:   %16 = icmp ne i64 %15, 100
// CHECK-NEXT:   br i1 %16, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 20 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymous5() {
	var s I2 = &struct {
		m int
		T
	}{10, T{100}}
	if s.Demo2() != 100 {
		panic("testAnonymous5 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymous6"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = load { i64, ptr }, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, ptr } %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4", ptr @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 100
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 20 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymous6() {
	var s I2 = struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo2() != 100 {
		panic("testAnonymous6 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymous7"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = load { i64, ptr }, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, ptr } %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$58AxoxqQ6sGUOM73FOqFrXsMlgxkU4HGd-S1Wl-ssYw", ptr @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 100
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 20 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %24 = getelementptr ptr, ptr %23, i64 4
// CHECK-NEXT:   %25 = load ptr, ptr %24, align 8
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } undef, ptr %25, 0
// CHECK-NEXT:   %27 = insertvalue { ptr, ptr } %26, ptr %22, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %27, 1
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %27, 0
// CHECK-NEXT:   %30 = call i64 %29(ptr %28)
// CHECK-NEXT:   %31 = icmp ne i64 %30, 100
// CHECK-NEXT:   br i1 %31, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 20 }, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %33)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymous7() {
	var s interface {
		Demo1() int
		Demo2() int
	} = struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous7 error")
	}
	if s.Demo2() != 100 {
		panic("testAnonymous7 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymous8"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %4, align 8
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %5 = load { i64, ptr }, ptr %0, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { i64, ptr } %5, ptr %6, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"{{.*}}/_testgo/abimethod.iface$kT5SIXt45Cspjl04Bof3DZVSOIltlDo-njpk6KqtZvA", ptr @"{{.*}}/_testgo/abimethod.struct$mRfo5gQx8vKF1DvrL24XRoyvI_ttVDcwc1JYMRxWfb8")
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %7, 0
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %8, ptr %6, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %12 = getelementptr ptr, ptr %11, i64 3
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = insertvalue { ptr, ptr } undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue { ptr, ptr } %14, ptr %10, 1
// CHECK-NEXT:   %16 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %18 = call i64 %17(ptr %16)
// CHECK-NEXT:   %19 = icmp ne i64 %18, 100
// CHECK-NEXT:   br i1 %19, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @25, i64 20 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %24 = getelementptr ptr, ptr %23, i64 4
// CHECK-NEXT:   %25 = load ptr, ptr %24, align 8
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } undef, ptr %25, 0
// CHECK-NEXT:   %27 = insertvalue { ptr, ptr } %26, ptr %22, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %27, 1
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %27, 0
// CHECK-NEXT:   %30 = call i64 %29(ptr %28)
// CHECK-NEXT:   %31 = icmp ne i64 %30, 100
// CHECK-NEXT:   br i1 %31, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @25, i64 20 }, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %33)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %9)
// CHECK-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %9, 0
// CHECK-NEXT:   %36 = getelementptr ptr, ptr %35, i64 5
// CHECK-NEXT:   %37 = load ptr, ptr %36, align 8
// CHECK-NEXT:   %38 = insertvalue { ptr, ptr } undef, ptr %37, 0
// CHECK-NEXT:   %39 = insertvalue { ptr, ptr } %38, ptr %34, 1
// CHECK-NEXT:   %40 = extractvalue { ptr, ptr } %39, 1
// CHECK-NEXT:   %41 = extractvalue { ptr, ptr } %39, 0
// CHECK-NEXT:   %42 = call i64 %41(ptr %40)
// CHECK-NEXT:   %43 = icmp ne i64 %42, 100
// CHECK-NEXT:   br i1 %43, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @25, i64 20 }, ptr %44, align 8
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %44, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %45)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymous8() {
	var s interface {
		Demo1() int
		Demo2() int
		demo3() int
	} = struct {
		m int
		*T
	}{10, &T{100}}
	if s.Demo1() != 100 {
		panic("testAnonymous8 error")
	}
	if s.Demo2() != 100 {
		panic("testAnonymous8 error")
	}
	if s.demo3() != 100 {
		panic("testAnonymous8 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testAnonymousBuffer"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @bytes.NewBufferString(%"{{.*}}/runtime/internal/runtime.String" { ptr @26, i64 5 })
// CHECK-NEXT:   store i64 10, ptr %1, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$O6rEVxIuA5O1E0KWpQBCgGx26X5gYhJ_nnJnHVL8_7U", ptr @"*{{.*}}/_testgo/abimethod.struct$RGW016k7zllXgGPm1CvD5-IBe-9lphOOTCFtYyDGLjY")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %0, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.String" %14(ptr %13)
// CHECK-NEXT:   %16 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %15, %"{{.*}}/runtime/internal/runtime.String" { ptr @26, i64 5 })
// CHECK-NEXT:   %17 = xor i1 %16, true
// CHECK-NEXT:   br i1 %17, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @97, i64 25 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testAnonymousBuffer() {
	var s fmt.Stringer = &struct {
		m int
		*bytes.Buffer
	}{10, bytes.NewBufferString("hello")}
	if s.String() != "hello" {
		panic("testAnonymousBuffer error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testGeneric"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uinGjIxPTfzB5e5h5gH-0VIvLl5rQdJ_yx2UsrxQqds", ptr @"*_llgo_{{.*}}/_testgo/abimethod.Pointer[any]")
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %1, 0
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %2, ptr %0, 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/_testgo/abimethod.testGeneric$1"()
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 4
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   call void %12(ptr %11, ptr %4)
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %3)
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %3, 0
// CHECK-NEXT:   %15 = getelementptr ptr, ptr %14, i64 3
// CHECK-NEXT:   %16 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %17 = insertvalue { ptr, ptr } undef, ptr %16, 0
// CHECK-NEXT:   %18 = insertvalue { ptr, ptr } %17, ptr %13, 1
// CHECK-NEXT:   %19 = extractvalue { ptr, ptr } %18, 1
// CHECK-NEXT:   %20 = extractvalue { ptr, ptr } %18, 0
// CHECK-NEXT:   %21 = call ptr %20(ptr %19)
// CHECK-NEXT:   %22 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %21, align 8
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %22, 0
// CHECK-NEXT:   %24 = icmp eq ptr %23, @_llgo_int
// CHECK-NEXT:   br i1 %24, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @111, i64 17 }, ptr %25, align 8
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %25, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %26)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %22, 1
// CHECK-NEXT:   %28 = load i64, ptr %27, align 8
// CHECK-NEXT:   %29 = icmp ne i64 %28, 100
// CHECK-NEXT:   br i1 %29, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @110, i64 40 }, ptr %30, align 8
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %30, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %31)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/_testgo/abimethod.testGeneric$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %2, ptr %0, align 8
// CHECK-NEXT:   ret ptr %0
// CHECK-NEXT: }
func testGeneric() {
	var p IP = &Pointer[any]{}
	p.Store(func() *any {
		var a any = 100
		return &a
	}())
	if (*p.Load()).(int) != 100 {
		panic("testGeneric error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testNamed1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"*_llgo_{{.*}}/_testgo/abimethod.T")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 3
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call i64 %12(ptr %11)
// CHECK-NEXT:   %14 = icmp ne i64 %13, 100
// CHECK-NEXT:   br i1 %14, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @112, i64 16 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testNamed1() {
	var a I = &T{100}
	if a.Demo1() != 100 {
		panic("testNamed1 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testNamed2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %"{{.*}}/_testgo/abimethod.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = load %"{{.*}}/_testgo/abimethod.T", ptr %0, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %"{{.*}}/_testgo/abimethod.T" %2, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$WkyTd7mXEW0USaC6FIo7OG9IdUUyjAJl_h3PFrMEtHc", ptr @"_llgo_{{.*}}/_testgo/abimethod.T")
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5, ptr %3, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 3
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call i64 %14(ptr %13)
// CHECK-NEXT:   %16 = icmp ne i64 %15, 100
// CHECK-NEXT:   br i1 %16, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @113, i64 16 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testNamed2() {
	var a I = T{100}
	if a.Demo1() != 100 {
		panic("testNamed2 error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.testNamed3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$GIQLduxo5T_xLwYbboAKy8LzikHgsGzb7WxrkOH3Lr4", ptr @"*_llgo_{{.*}}/_testgo/abimethod.T")
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %2, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %3, ptr %0, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %4)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %4, 0
// CHECK-NEXT:   %7 = getelementptr ptr, ptr %6, i64 3
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = insertvalue { ptr, ptr } undef, ptr %8, 0
// CHECK-NEXT:   %10 = insertvalue { ptr, ptr } %9, ptr %5, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %10, 1
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %10, 0
// CHECK-NEXT:   %13 = call i64 %12(ptr %11)
// CHECK-NEXT:   %14 = icmp ne i64 %13, 100
// CHECK-NEXT:   br i1 %14, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @114, i64 16 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func testNamed3() {
	var a I2 = &T{100}
	if a.Demo2() != 100 {
		panic("testNamed4 error")
	}
}

type Pointer[T any] struct {
	// Mention *T in a field to disallow conversion between Pointer types.
	// See go.dev/issue/56603 for more details.
	// Use *T, not T, to avoid spurious recursive type definition errors.
	_ [0]*T
	v unsafe.Pointer
}

// Load atomically loads and returns the value stored in x.
func (x *Pointer[T]) Load() *T { return (*T)(atomic.LoadPointer(&x.v)) }

// Store atomically stores val into x.
func (x *Pointer[T]) Store(val *T) { atomic.StorePointer(&x.v, unsafe.Pointer(val)) }

type IP interface {
	Store(*any)
	Load() *any
}

type I interface {
	Demo1() int
}

type I2 interface {
	Demo2() int
}

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = load %"{{.*}}/_testgo/abimethod.T", ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/_testgo/abimethod.T.Demo1"(%"{{.*}}/_testgo/abimethod.T" %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/_testgo/abimethod.(*T).Demo2"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; *{{.*}}/_testgo/abimethod.T}.demo3"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/_testgo/abimethod.(*T).demo3"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo1"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = load %"{{.*}}/_testgo/abimethod.T", ptr %3, align 8
// CHECK-NEXT:   %5 = call i64 @"{{.*}}/_testgo/abimethod.T.Demo1"(%"{{.*}}/_testgo/abimethod.T" %4)
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.struct{m int; *{{.*}}/_testgo/abimethod.T}.Demo2"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/_testgo/abimethod.(*T).Demo2"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.struct{m int; *{{.*}}/_testgo/abimethod.T}.demo3"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/_testgo/abimethod.(*T).demo3"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.struct{m int; {{.*}}/_testgo/abimethod.T}.Demo1"({ i64, %"{{.*}}/_testgo/abimethod.T" } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, %"{{.*}}/_testgo/abimethod.T" }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, %"{{.*}}/_testgo/abimethod.T" } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load %"{{.*}}/_testgo/abimethod.T", ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/_testgo/abimethod.T.Demo1"(%"{{.*}}/_testgo/abimethod.T" %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.Demo1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load %"{{.*}}/_testgo/abimethod.T", ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/_testgo/abimethod.T.Demo1"(%"{{.*}}/_testgo/abimethod.T" %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.Demo2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/_testgo/abimethod.(*T).Demo2"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; {{.*}}/_testgo/abimethod.T}.demo3"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, %"{{.*}}/_testgo/abimethod.T" }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/_testgo/abimethod.(*T).demo3"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Available"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"bytes.(*Buffer).Available"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.AvailableBuffer"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).AvailableBuffer"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Bytes"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Bytes"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Cap"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"bytes.(*Buffer).Cap"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Grow"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Grow"(ptr %3, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Len"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i64 @"bytes.(*Buffer).Len"(ptr %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Next"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Next"(ptr %3, i64 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Read"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Read"(ptr %3, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadByte"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadByte"(ptr %2)
// CHECK-NEXT:   %4 = extractvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %3, 0
// CHECK-NEXT:   %5 = extractvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %3, 1
// CHECK-NEXT:   %6 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } undef, i8 %4, 0
// CHECK-NEXT:   %7 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %6, %"{{.*}}/runtime/internal/runtime.iface" %5, 1
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %7
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadBytes"(ptr %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadBytes"(ptr %3, i8 %1)
// CHECK-NEXT:   %5 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %5, 0
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadFrom"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadFrom"(ptr %3, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadRune"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadRune"(ptr %2)
// CHECK-NEXT:   %4 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %3, 0
// CHECK-NEXT:   %5 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %3, 1
// CHECK-NEXT:   %6 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %3, 2
// CHECK-NEXT:   %7 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %4, 0
// CHECK-NEXT:   %8 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, i64 %5, 1
// CHECK-NEXT:   %9 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %6, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.ReadString"(ptr %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadString"(ptr %3, i8 %1)
// CHECK-NEXT:   %5 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.String" %5, 0
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Reset"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Reset"(ptr %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.String"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @"bytes.(*Buffer).String"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Truncate"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Truncate"(ptr %3, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadByte"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadByte"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.UnreadRune"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadRune"(ptr %2)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %3
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.Write"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Write"(ptr %3, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteByte"(ptr %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).WriteByte"(ptr %3, i8 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteRune"(ptr %0, i32 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteRune"(ptr %3, i32 %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteString"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteString"(ptr %3, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.WriteTo"(ptr %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteTo"(ptr %3, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %5 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.empty"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i1 @"bytes.(*Buffer).empty"(ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.grow"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"bytes.(*Buffer).grow"(ptr %3, i64 %1)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.readSlice"(ptr %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).readSlice"(ptr %3, i8 %1)
// CHECK-NEXT:   %5 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %5, 0
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, i1 } @"{{.*}}/_testgo/abimethod.*struct{m int; *bytes.Buffer}.tryGrowByReslice"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i64, i1 } @"bytes.(*Buffer).tryGrowByReslice"(ptr %3, i64 %1)
// CHECK-NEXT:   %5 = extractvalue { i64, i1 } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i64, i1 } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i64, i1 } undef, i64 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i64, i1 } %7, i1 %6, 1
// CHECK-NEXT:   ret { i64, i1 } %8
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Available"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"bytes.(*Buffer).Available"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.AvailableBuffer"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).AvailableBuffer"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Bytes"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Bytes"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Cap"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"bytes.(*Buffer).Cap"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Grow"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Grow"(ptr %4, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Len"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i64 @"bytes.(*Buffer).Len"(ptr %3)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Next"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.Slice" @"bytes.(*Buffer).Next"(ptr %4, i64 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %5
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Read"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Read"(ptr %4, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadByte"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i8, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadByte"(ptr %3)
// CHECK-NEXT:   %5 = extractvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } undef, i8 %5, 0
// CHECK-NEXT:   %8 = insertvalue { i8, %"{{.*}}/runtime/internal/runtime.iface" } %7, %"{{.*}}/runtime/internal/runtime.iface" %6, 1
// CHECK-NEXT:   ret { i8, %"{{.*}}/runtime/internal/runtime.iface" } %8
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadBytes"({ i64, ptr } %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadBytes"(ptr %4, i8 %1)
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %6, 0
// CHECK-NEXT:   %9 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadFrom"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadFrom"(ptr %4, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadRune"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadRune"(ptr %3)
// CHECK-NEXT:   %5 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 0
// CHECK-NEXT:   %6 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 1
// CHECK-NEXT:   %7 = extractvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %4, 2
// CHECK-NEXT:   %8 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %5, 0
// CHECK-NEXT:   %9 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, i64 %6, 1
// CHECK-NEXT:   %10 = insertvalue { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %9, %"{{.*}}/runtime/internal/runtime.iface" %7, 2
// CHECK-NEXT:   ret { i32, i64, %"{{.*}}/runtime/internal/runtime.iface" } %10
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.ReadString"({ i64, ptr } %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).ReadString"(ptr %4, i8 %1)
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.String" %6, 0
// CHECK-NEXT:   %9 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Reset"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Reset"(ptr %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.String"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @"bytes.(*Buffer).String"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Truncate"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   call void @"bytes.(*Buffer).Truncate"(ptr %4, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadByte"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadByte"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.UnreadRune"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).UnreadRune"(ptr %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %4
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.Write"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.Slice" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).Write"(ptr %4, %"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteByte"({ i64, ptr } %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.iface" @"bytes.(*Buffer).WriteByte"(ptr %4, i8 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %5
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteRune"({ i64, ptr } %0, i32 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteRune"(ptr %4, i32 %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteString"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteString"(ptr %4, %"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.WriteTo"({ i64, ptr } %0, %"{{.*}}/runtime/internal/runtime.iface" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).WriteTo"(ptr %4, %"{{.*}}/runtime/internal/runtime.iface" %1)
// CHECK-NEXT:   %6 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.empty"({ i64, ptr } %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds { i64, ptr }, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i1 @"bytes.(*Buffer).empty"(ptr %3)
// CHECK-NEXT:   ret i1 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.grow"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call i64 @"bytes.(*Buffer).grow"(ptr %4, i64 %1)
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }

// CHECK-LABEL: define { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.readSlice"({ i64, ptr } %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } @"bytes.(*Buffer).readSlice"(ptr %4, i8 %1)
// CHECK-NEXT:   %6 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 0
// CHECK-NEXT:   %7 = extractvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %5, 1
// CHECK-NEXT:   %8 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } undef, %"{{.*}}/runtime/internal/runtime.Slice" %6, 0
// CHECK-NEXT:   %9 = insertvalue { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %8, %"{{.*}}/runtime/internal/runtime.iface" %7, 1
// CHECK-NEXT:   ret { %"{{.*}}/runtime/internal/runtime.Slice", %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, i1 } @"{{.*}}/_testgo/abimethod.struct{m int; *bytes.Buffer}.tryGrowByReslice"({ i64, ptr } %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca { i64, ptr }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store { i64, ptr } %0, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds { i64, ptr }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call { i64, i1 } @"bytes.(*Buffer).tryGrowByReslice"(ptr %4, i64 %1)
// CHECK-NEXT:   %6 = extractvalue { i64, i1 } %5, 0
// CHECK-NEXT:   %7 = extractvalue { i64, i1 } %5, 1
// CHECK-NEXT:   %8 = insertvalue { i64, i1 } undef, i64 %6, 0
// CHECK-NEXT:   %9 = insertvalue { i64, i1 } %8, i1 %7, 1
// CHECK-NEXT:   ret { i64, i1 } %9
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"{{.*}}/_testgo/abimethod.(*Pointer[any]).Load"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.Pointer[any]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %2 = load atomic ptr, ptr %1 seq_cst, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"{{.*}}/_testgo/abimethod.(*Pointer[any]).Store"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/_testgo/abimethod.Pointer[any]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   store atomic ptr %1, ptr %2 seq_cst, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
