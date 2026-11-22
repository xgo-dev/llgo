//go:build !llgo

package ssa

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWasmBoundaryPaddingUses(t *testing.T) {
	const home = `
  %home = alloca %Record
  store %Record %value, ptr %home
`
	for _, tc := range []struct {
		name, body string
		clear      bool
	}{
		{"length_and_capacity", `
  %length = extractvalue %Record %value, 0, 1, 1
  %capacity = extractvalue %Record %value, 1, 2
  %other = extractvalue %Record %value, 2
`, false},
		// The data slot is not itself a header. Losing the header context
		// while walking a multi-index extract would incorrectly skip its clear.
		{"data_slot", `
  %data = extractvalue %Record %value, 0, 1, 0
  call void @slot({ ptr, i32 } %data)
`, true},
		{"padding_word", `
  %padding = extractvalue %Record %value, 1, 0, 1
  call void @word(i32 %padding)
`, true},
		{"chained_data_slot", `
  %array = extractvalue %Record %value, 0
  %text = extractvalue [2 x %String] %array, 1
  %data = extractvalue %String %text, 0
  %pointer = extractvalue { ptr, i32 } %data, 0
  call void @escape(ptr %pointer)
`, false},
		{"overwritten_padding", `
  %header = extractvalue %Record %value, 1
  %data = extractvalue %Slice %header, 0
  %cleanData = insertvalue { ptr, i32 } %data, i32 0, 1
  %clean = insertvalue %Slice %header, { ptr, i32 } %cleanData, 0
  call void @header(%Slice %clean)
`, false},
		{"overwritten_padding_direct", `
  %header = extractvalue %Record %value, 1
  %clean = insertvalue %Slice %header, i32 0, 0, 1
  call void @header(%Slice %clean)
`, false},
		{"pointer_write_preserves_padding", `
  %header = extractvalue %Record %value, 1
  %updated = insertvalue %Slice %header, ptr null, 0, 0
  call void @header(%Slice %updated)
`, true},
		{"copied_data_slot", `
  %data = extractvalue %Record %value, 1, 0
  %copy = insertvalue %Slice zeroinitializer, { ptr, i32 } %data, 0
  call void @header(%Slice %copy)
`, true},
		{"whole_value", `
  call void @consume(%Record %value)
`, true},
		{"private_length_and_capacity", home + `
  %lengthp = getelementptr %Record, ptr %home, i32 0, i32 0, i32 1, i32 1
  %length = load i64, ptr %lengthp
  %capacityp = getelementptr %Record, ptr %home, i32 0, i32 1, i32 2
  %capacity = load i64, ptr %capacityp
`, false},
		{"private_reload", home + `
  %copy = load %Record, ptr %home
  %length = extractvalue %Record %copy, 0, 1, 1
`, false},
		{"private_lifetime", `
  %home = alloca %Record
  call void @llvm.lifetime.start(ptr %home)
  store %Record %value, ptr %home
  %copy = load %Record, ptr %home
  %length = extractvalue %Record %copy, 0, 1, 1
  call void @llvm.lifetime.end(ptr %home)
`, false},
		// Builder.Index can spill an array and address an element directly,
		// without a leading zero index into the array's LLVM type.
		{"private_array_element", `
  %array = extractvalue %Record %value, 0
  %home = alloca [2 x %String]
  store [2 x %String] %array, ptr %home
  %element = getelementptr %String, ptr %home, i32 1
  %lengthp = getelementptr %String, ptr %element, i32 0, i32 1
  %length = load i64, ptr %lengthp
`, false},
		{"private_data_slot", home + `
  %datap = getelementptr %Record, ptr %home, i32 0, i32 1, i32 0
  %data = load { ptr, i32 }, ptr %datap
  call void @slot({ ptr, i32 } %data)
`, true},
		{"private_native_pointer", home + `
  %datap = getelementptr %Record, ptr %home, i32 0, i32 1, i32 0
  %nativep = getelementptr {ptr, i32}, ptr %datap, i32 0, i32 0
  %native = load ptr, ptr %nativep
  call void @escape(ptr %native)
`, false},
		{"private_padding_word", home + `
  %wordp = getelementptr %Record, ptr %home, i32 0, i32 1, i32 0, i32 1
  %padding = load i32, ptr %wordp
  call void @word(i32 %padding)
`, true},
		{"raw_load", home + `
  %bits = load i64, ptr %home
`, true},
		{"type_punned_gep", home + `
  %bytep = getelementptr i8, ptr %home, i32 1
  %byte = load i8, ptr %bytep
`, true},
		{"zero_offset_type_pun", home + `
  %bytep = getelementptr i8, ptr %home, i32 0
  %byte = load i8, ptr %bytep
`, true},
		{"raw_padding_offset", home + `
  %paddingp = getelementptr i8, ptr %home, i32 4
  %padding = load i32, ptr %paddingp
`, true},
		{"private_memset", home + `
  call void @llvm.memset.p0.i32(ptr %home, i8 0, i32 64, i1 false)
`, false},
		{"volatile_memset", home + `
  call void @llvm.memset.p0.i32(ptr %home, i8 0, i32 64, i1 true)
`, true},
		{"volatile_load", home + `
  %lengthp = getelementptr %Record, ptr %home, i32 0, i32 0, i32 1, i32 1
  %length = load volatile i64, ptr %lengthp
`, true},
		{"atomic_load", home + `
  %lengthp = getelementptr %Record, ptr %home, i32 0, i32 0, i32 1, i32 1
  %length = load atomic i64, ptr %lengthp monotonic, align 8
`, true},
		{"volatile_store", `
  %home = alloca %Record
  store volatile %Record %value, ptr %home
`, true},
		{"additional_store", home + `
  store %Record zeroinitializer, ptr %home
`, true},
		{"field_mutation", home + `
  %lengthp = getelementptr %Record, ptr %home, i32 0, i32 0, i32 1, i32 1
  store i64 7, ptr %lengthp
`, true},
		{"escaped_home", home + `
  call void @escape(ptr %home)
`, true},
		// A scalar field's address can be used to reach its containing
		// allocation; the absence of headers in that field does not prove safety.
		{"escaped_scalar_field", home + `
  %otherp = getelementptr %Record, ptr %home, i32 0, i32 2
  call void @escape(ptr %otherp)
`, true},
		{"escaped_header_length", home + `
  %lengthp = getelementptr %Record, ptr %home, i32 0, i32 0, i32 1, i32 1
  call void @escape(ptr %lengthp)
`, true},
		{"stored_home_address", home + `
  %address = ptrtoint ptr %home to i32
  %wide = zext i32 %address to i64
  store i64 %wide, ptr @escapedAddress
`, true},
		{"indirect_escape", home + `
  call void %callback(ptr %home)
`, true},
		{"dynamic_index", home + `
  %element = getelementptr %Record, ptr %home, i32 0, i32 0, i64 %index
  %text = load %String, ptr %element
  %length = extractvalue %String %text, 1
`, true},
		{"out_of_bounds_index", home + `
  %element = getelementptr %Record, ptr %home, i32 0, i32 0, i32 2
  %text = load %String, ptr %element
  %length = extractvalue %String %text, 1
`, true},
		// Keep nonzero base indices conservative, even when the address returns
		// to the original allocation before a valid, length-only read.
		{"nonzero_base_index", home + `
  %next = getelementptr %Record, ptr %home, i32 1
  %back = getelementptr %Record, ptr %next, i32 -1
  %copy = load %Record, ptr %back
  %length = extractvalue %Record %copy, 0, 1, 1
`, true},
		{"different_allocation_type", `
  %home = alloca [64 x i8]
  store %Record %value, ptr %home
`, true},
		{"multiple_allocations", `
  %home = alloca %Record, i32 2
  store %Record %value, ptr %home
`, true},
		// A valid use graph can cycle through memory without an SSA cycle.
		// The proof must terminate conservatively instead of following forever.
		{"cyclic_copy", home + `
  %copy = load %Record, ptr %home
  store %Record %copy, ptr %home
`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := newJ32Program(t)
			// These are the physical Memory32 header layouts. Parsing the
			// fixtures also exercises LLVM's multi-index extract instructions.
			ir := `
%String = type { { ptr, i32 }, i64 }
%Slice = type { { ptr, i32 }, i64, i64 }
%Record = type { [2 x %String], %Slice, i64 }
@escapedAddress = global i64 0
declare void @header(%Slice)
declare void @word(i32)
declare void @slot({ ptr, i32 })
declare void @consume(%Record)
declare void @escape(ptr)
declare void @llvm.memset.p0.i32(ptr, i8, i32, i1 immarg)
declare void @llvm.lifetime.start(ptr)
declare void @llvm.lifetime.end(ptr)
define void @observe(%Record %value, i64 %index, ptr %callback) {
entry:
` + tc.body + "\n  ret void\n}\n"
			path := filepath.Join(t.TempDir(), "boundary.ll")
			if err := os.WriteFile(path, []byte(ir), 0o644); err != nil {
				t.Fatal(err)
			}
			buf, err := llvm.NewMemoryBufferFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			mod, err := prog.ctx.ParseIR(buf)
			if err != nil {
				t.Fatal(err)
			}
			defer mod.Dispose()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			prog.rtStringTy = mod.GetTypeByName("String")
			prog.rtSliceTy = mod.GetTypeByName("Slice")
			value := mod.NamedFunction("observe").Param(0)
			if got := prog.wasmPaddingObserved(value); got != tc.clear {
				t.Fatalf("boundary needs padding cleared = %v, want %v:\n%s", got, tc.clear, ir)
			}
		})
	}
}
