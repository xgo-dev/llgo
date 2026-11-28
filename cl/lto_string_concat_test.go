package cl_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Keep the IR driver independent of the Go LLVM bindings so the pass can also
// be tested directly: LLGO_LTO_PLUGIN=... go test cl/lto_string_concat_test.go.
func TestLTOPluginStringConcat(t *testing.T) {
	plugin := os.Getenv("LLGO_LTO_PLUGIN")
	if plugin == "" {
		t.Skip("LLGO_LTO_PLUGIN is not set")
	}
	opt := "opt"
	if config := os.Getenv("LLVM_CONFIG"); config != "" {
		out, err := exec.Command(config, "--bindir").Output()
		if err != nil {
			t.Fatalf("find opt: %v", err)
		}
		opt = filepath.Join(strings.TrimSpace(string(out)), "opt")
	}
	check := func(name, ir string, want []string) {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(opt, "-load-pass-plugin="+plugin,
				"-passes=llgo-lto-pre-globaldce", "-verify-each", "-S", "-o", "-")
			cmd.Stdin = strings.NewReader(ir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run StringConcat refinement: %v\n%s\ninput:\n%s", err, out, ir)
			}
			result := string(out)
			generic := strings.Contains(result, `metadata !"go.method.value.reflect"`)
			if generic != (len(want) == 0) {
				t.Fatalf("generic check = %v, want %v\n%s", generic, len(want) == 0, result)
			}
			if len(want) == 0 && strings.Contains(result, `metadata !"go.method.value.reflect.`) {
				t.Fatalf("unknown name gained a refined check\n%s", result)
			}
			for _, name := range want {
				if !strings.Contains(result, `metadata !"go.method.value.reflect.`+name+`"`) {
					t.Fatalf("missing finite choice %s\n%s", name, result)
				}
			}
		})
	}
	cases := []struct {
		name string
		want []string
	}{
		{name: "complete_table", want: []string{"KeepA"}},
		{name: "zero_initialized", want: []string{"KeepA"}},
		{name: "dynamic_finite_part", want: []string{"KeepA", "KeepB"}},
		{name: "lifetime_markers", want: []string{"KeepA"}},
		{name: "length_after_element_init"},
		{name: "length_before_element_init"},
		{name: "overwritten_pointer"},
		{name: "overlapping_store"},
		{name: "zero_after_stores"},
		{name: "partial_zero"},
		{name: "volatile_zero"},
		{name: "wide_zero"},
		{name: "unknown_alias"},
		{name: "unknown_aggregate_alias"},
		{name: "table_as_result_buffer"},
		{name: "concat_operand_bundle"},
		{name: "memset_operand_bundle"},
		{name: "lifetime_operand_bundle"},
		{name: "lifetime_bundle_only"},
		{name: "non_dominating_store"},
		{name: "conflicting_store"},
		{name: "duplicate_store"},
		{name: "uninitialized_element"},
		{name: "out_of_bounds_count"},
		{name: "partial_table_count"},
		{name: "wide_count"},
		{name: "unaligned_table_start"},
		{name: "mutable_string_bytes"},
		{name: "external_string_initializer"},
		{name: "weak_string_initializer"},
		{name: "unrelated_concat_symbol"},
	}
	// This matrix uses matching native pointer/int widths. Wasm32 keeps 64-bit
	// Go lengths and lowers String results through sret; it is tested below.
	for _, bits := range []int{32, 64} {
		for _, scalarStores := range []bool{false, true} {
			for _, tc := range cases {
				check(fmt.Sprintf("%d/scalar-stores=%t/%s", bits, scalarStores, tc.name),
					concatPluginIR(bits, scalarStores, tc.name), tc.want)
			}
		}
	}
	// The historical reader has an eight-byte field offset and always uses
	// scalar stores. Running it under both table representations adds no coverage.
	check("legacy_local_slots", concatPluginIR(64, true, "legacy_local_slots"), []string{"KeepA"})
	check("legacy_local_slots_clobber", concatPluginIR(64, true, "legacy_local_slots_clobber"), nil)
	for _, helper := range []string{"StringConcat", "StringCatBuffer"} {
		check("wasm32/"+helper, concatPluginWasmABIIR(helper), nil)
	}
}

func concatPluginIR(bits int, scalarStores bool, scenario string) string {
	word := fmt.Sprintf("i%d", bits)
	tableSize := bits / 8 * 4
	zero := fmt.Sprintf("call void @llvm.memset.p0.i64(ptr %%table, i8 0, i64 %d, i1 false)", tableSize)
	bundle := ` ["unknown"(ptr %table)]`
	first := fmt.Sprintf("store %%string { ptr @prefix, %s 4 }, ptr %%table", word)
	part := fmt.Sprintf("store %%string { ptr @a, %s 1 }, ptr %%second", word)
	if scalarStores {
		first = fmt.Sprintf("store ptr @prefix, ptr %%table\n%%firstlen = getelementptr %%string, ptr %%table, i32 0, i32 1\nstore %s 4, ptr %%firstlen", word)
		part = fmt.Sprintf("store ptr @a, ptr %%second\n%%secondlen = getelementptr %%string, ptr %%second, i32 0, i32 1\nstore %s 1, ptr %%secondlen", word)
	}
	before, after, tail, callBundle := "", "", "", ""
	data, count, countWord, buffer := "%table", "2", word, "null"
	prefix, prefixKind := "Keep", "private constant"
	callee := "github.com/xgo-dev/llgo/runtime/internal/runtime.StringConcat"
	legacy := false
	switch scenario {
	case "complete_table":
	case "zero_initialized":
		before = zero
	case "partial_zero":
		before = strings.Replace(zero, fmt.Sprintf("i64 %d", tableSize), "i64 1", 1)
	case "volatile_zero":
		before = strings.Replace(zero, "i1 false", "i1 true", 1)
	case "wide_zero":
		before = "call void @llvm.memset.p0.i128(ptr %table, i8 0, i128 18446744073709551616, i1 false)"
	case "dynamic_finite_part":
		part = fmt.Sprintf("%%suffix = select i1 %%choice, %%string { ptr @a, %s 1 }, %%string { ptr @b, %s 1 }\n", word, word)
		if scalarStores {
			part += fmt.Sprintf("%%suffixp = extractvalue %%string %%suffix, 0\n%%suffixn = extractvalue %%string %%suffix, 1\nstore ptr %%suffixp, ptr %%second\n%%secondlen = getelementptr %%string, ptr %%second, i32 0, i32 1\nstore %s %%suffixn, ptr %%secondlen", word)
		} else {
			part += "store %string %suffix, ptr %second"
		}
	case "non_dominating_store":
		part = "br i1 %choice, label %write, label %join\nwrite:\n" + part + "\nbr label %join\njoin:"
	case "uninitialized_element":
		part = ""
	case "length_after_element_init", "length_before_element_init":
		after = fmt.Sprintf("%%len = getelementptr %%string, ptr %%table, i32 0, i32 1\nstore %s 2, ptr %%len", word)
		if scenario == "length_before_element_init" {
			before, after = after, ""
		}
	case "overwritten_pointer":
		after = "store ptr @other, ptr %table"
	case "overlapping_store":
		after = "store i8 0, ptr %table"
	case "zero_after_stores":
		after = zero
	case "unknown_alias":
		after = "call void @unknown(ptr %table)"
	case "unknown_aggregate_alias":
		after = fmt.Sprintf("%%alias0 = insertvalue %%slice poison, ptr %%table, 0\n%%alias1 = insertvalue %%slice %%alias0, %s 2, 1\n%%alias = insertvalue %%slice %%alias1, %s 2, 2\ncall void @unknownSlice(%%slice %%alias)", word, word)
	case "conflicting_store":
		after = fmt.Sprintf("store %%string { ptr @b, %s 1 }, ptr %%second", word)
	case "duplicate_store":
		after = fmt.Sprintf("store %%string { ptr @prefix, %s 4 }, ptr %%table", word)
		if scalarStores {
			after = "store ptr @prefix, ptr %table"
		}
	case "out_of_bounds_count":
		count = "3"
	case "partial_table_count":
		count = "1"
	case "wide_count":
		countWord, count = "i128", "18446744073709551616"
	case "table_as_result_buffer":
		buffer = "%table"
	case "unaligned_table_start":
		after = "%unaligned = getelementptr i8, ptr %table, i64 1"
		data, count = "%unaligned", "1"
	case "mutable_string_bytes":
		prefixKind = "private global"
		after = "store i8 68, ptr @prefix"
	case "external_string_initializer":
		prefixKind = "private externally_initialized constant"
	case "weak_string_initializer":
		prefixKind = "weak constant"
	case "unrelated_concat_symbol":
		callee = "example.org/runtime/internal/runtime.StringConcat"
	case "lifetime_markers":
		before = "call void @llvm.lifetime.start.p0(ptr %table)"
		tail = "call void @llvm.lifetime.end.p0(ptr %table)"
	case "concat_operand_bundle":
		callBundle = bundle
	case "memset_operand_bundle":
		before = zero + bundle
		after = "call void @unknownNoArgs()"
	case "lifetime_operand_bundle":
		before = "call void @llvm.lifetime.start.p0(ptr %table)" + bundle
		after = "call void @unknownNoArgs()"
	case "lifetime_bundle_only":
		before = "%dummy = alloca i8\ncall void @llvm.lifetime.start.p0(ptr %dummy)" + bundle
		after = "call void @unknownNoArgs()"
	case "legacy_local_slots", "legacy_local_slots_clobber":
		legacy, prefix = true, "KeepA"
		if scenario == "legacy_local_slots_clobber" {
			after = zero

		}
	default:
		panic("unknown StringConcat IR scenario: " + scenario)
	}
	body := fmt.Sprintf(`
  %%second = getelementptr [2 x %%string], ptr %%table, i32 0, i32 1
  %s
  %s
  %s
  %s
  %%name = call %%string @"%s"(ptr %s, %s %s, ptr %s)%s
  %%p = extractvalue %%string %%name, 0
  %%n = extractvalue %%string %%name, 1
  %s
`, before, first, part, after, callee, data, countWord, count, buffer, callBundle, tail)
	if legacy {
		body = before + `
  %firstlen = getelementptr i8, ptr %table, i64 8
  %second = getelementptr %string, ptr %table, i64 1
  %secondlen = getelementptr i8, ptr %second, i64 8
  store ptr @prefix, ptr %table
  store i64 5, ptr %firstlen
  store ptr @prefix, ptr %second
  store i64 5, ptr %secondlen
  ` + after + `
  %index = select i1 %choice, i64 0, i64 1
  %element = getelementptr %string, ptr %table, i64 %index
  %length = getelementptr i8, ptr %element, i64 8
  %p = load ptr, ptr %element
  %n = load i64, ptr %length
`
	}
	return fmt.Sprintf(`
target datalayout = "e-p:%d:%d-i64:64-n8:16:32:64-S128"
%%string = type { ptr, %s }
%%slice = type { ptr, %s, %s }
%%value = type { ptr, ptr, i64 }
@prefix = %s [%d x i8] c%q
@other = private constant [4 x i8] c"Drop"
@a = private constant [1 x i8] c"A"
@b = private constant [1 x i8] c"B"
declare %%string @"%s"(ptr, %s, ptr)
declare void @llvm.memset.p0.i64(ptr, i8, i64, i1)
declare void @llvm.memset.p0.i128(ptr, i8, i128, i1)
declare void @llvm.lifetime.start.p0(ptr)
declare void @llvm.lifetime.end.p0(ptr)
declare void @unknown(ptr)
declare void @unknownNoArgs()
declare void @unknownSlice(%%slice)
declare void @methodByName(ptr sret(%%value), ptr, %s)
declare { ptr, i1 } @llvm.type.checked.load(ptr, i32, metadata)
declare void @llvm.assume(i1)
declare void @consume(ptr)
define void @entry(i1 %%choice) {
  %%ret = alloca %%value
  %%table = alloca [2 x %%string]
  %s
  call void @methodByName(ptr sret(%%value) %%ret, ptr "llgo.reflect.methodbyname.name"="1" %%p, %s %%n) #0
  %%slot = getelementptr %%value, ptr %%ret, i32 0, i32 1
  %%method = load ptr, ptr %%slot
  %%checked = call { ptr, i1 } @llvm.type.checked.load(ptr %%method, i32 0, metadata !"go.method.value.reflect")
  %%ok = extractvalue { ptr, i1 } %%checked, 1
  call void @llvm.assume(i1 %%ok)
  call void @consume(ptr %%method)
  ret void
}
attributes #0 = { "llgo.reflect.methodbyname"="value" }
`, bits, bits, word, word, word, prefixKind, len(prefix), prefix, callee, countWord, word, body, word)
}

// These helper signatures and the padded String come from production Wasm32
// post-ABI IR: physical pointers are 32 bits, Go lengths are 64 bits, results
// use sret, and binary String arguments use byval. Normalize only the reflective
// consumer so this test isolates the helper boundary's conservative fallback.
func concatPluginWasmABIIR(helper string) string {
	var decl, call string
	switch helper {
	case "StringConcat":
		decl = "ptr sret(%string), ptr, i64, ptr"
		call = "ptr sret(%string) %nameout, ptr %table, i64 2, ptr null"
	case "StringCatBuffer":
		decl = "ptr sret(%string), ptr byval(%string), ptr byval(%string), ptr"
		call = "ptr sret(%string) %nameout, ptr byval(%string) %table, ptr byval(%string) %second, ptr null"
	default:
		panic("unknown Wasm string helper: " + helper)
	}
	return fmt.Sprintf(`
target datalayout = "e-m:e-p:32:32-i64:64-n32:64-S128"
target triple = "wasm32-unknown-emscripten"
%%string = type { { ptr, i32 }, i64 }
%%value = type { ptr, ptr, i64 }
@name = private constant [5 x i8] c"KeepA"
declare void @"github.com/xgo-dev/llgo/runtime/internal/runtime.%s"(%s)
declare void @methodByName(ptr sret(%%value), ptr, i64)
declare { ptr, i1 } @llvm.type.checked.load(ptr, i32, metadata)
declare void @llvm.assume(i1)
declare void @consume(ptr)
define void @entry() {
  %%ret = alloca %%value
  %%nameout = alloca %%string
  %%table = alloca [2 x %%string]
  %%second = getelementptr [2 x %%string], ptr %%table, i32 0, i32 1
  store %%string { { ptr, i32 } { ptr @name, i32 0 }, i64 5 }, ptr %%table
  store %%string zeroinitializer, ptr %%second
  call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.%s"(%s)
  %%name = load %%string, ptr %%nameout
  %%data = extractvalue %%string %%name, 0
  %%p = extractvalue { ptr, i32 } %%data, 0
  %%n = extractvalue %%string %%name, 1
  call void @methodByName(ptr sret(%%value) %%ret, ptr "llgo.reflect.methodbyname.name"="1" %%p, i64 %%n) #0
  %%slot = getelementptr %%value, ptr %%ret, i32 0, i32 1
  %%method = load ptr, ptr %%slot
  %%checked = call { ptr, i1 } @llvm.type.checked.load(ptr %%method, i32 0, metadata !"go.method.value.reflect")
  %%ok = extractvalue { ptr, i1 } %%checked, 1
  call void @llvm.assume(i1 %%ok)
  call void @consume(ptr %%method)
  ret void
}
attributes #0 = { "llgo.reflect.methodbyname"="value" }
`, helper, decl, helper, call)
}
