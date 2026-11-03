package cl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Inspect the actual readonly function descriptor instead of looking for its
// code name inside an unrelated containing type or global initializer.
func ownerFuncvalParts(t *testing.T, storage llvm.Value) (llvm.Value, llvm.Value) {
	t.Helper()
	pointer := func(value llvm.Value) llvm.Value {
		if value.IsNil() {
			t.Fatal("missing function pointer storage")
		}
		if value.Type().TypeKind() == llvm.StructTypeKind {
			fields := value.Type().StructElementTypes()
			if len(fields) != 2 ||
				fields[0].TypeKind() != llvm.PointerTypeKind ||
				fields[1].TypeKind() != llvm.IntegerTypeKind || fields[1].IntTypeWidth() != 32 {
				t.Fatalf("invalid Memory32 pointer storage: %s", value)
			}
			if value.IsNull() {
				return llvm.ConstNull(fields[0])
			}
			if value.OperandsCount() != 2 || !value.Operand(1).IsNull() {
				t.Fatalf("invalid Memory32 pointer initializer: %s", value)
			}
			value = value.Operand(0)
		}
		if value.Type().TypeKind() != llvm.PointerTypeKind {
			t.Fatalf("function storage is not a pointer: %s", value)
		}
		return value
	}
	descriptor := pointer(storage).IsAGlobalVariable()
	if descriptor.IsNil() || !descriptor.IsGlobalConstant() {
		t.Fatalf("function storage is not a readonly descriptor: %s", storage)
	}
	header := descriptor.Initializer()
	if header.IsNil() || header.Type().TypeKind() != llvm.StructTypeKind ||
		header.OperandsCount() < 2 || header.OperandsCount() > 3 {
		t.Fatalf("function descriptor is not code/environment: %s", header)
	}
	if header.OperandsCount() == 3 {
		// Hosted Memory32 uses a zero-length i64 tail to give the descriptor
		// its Go-word alignment. It is not a third logical header field.
		padding := header.Operand(2)
		if padding.Type().TypeKind() != llvm.ArrayTypeKind || padding.Type().ArrayLength() != 0 ||
			padding.Type().ElementType().TypeKind() != llvm.IntegerTypeKind ||
			padding.Type().ElementType().IntTypeWidth() != 64 || !padding.IsNull() {
			t.Fatalf("function descriptor has a nonalignment third field: %s", header)
		}
	}
	return pointer(header.Operand(0)), pointer(header.Operand(1))
}

func ownerIRModule(t *testing.T, ir string) llvm.Module {
	t.Helper()
	path := filepath.Join(t.TempDir(), "owner.ll")
	if err := os.WriteFile(path, []byte(ir), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := llvm.NewContext()
	t.Cleanup(ctx.Dispose)
	buffer, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := ctx.ParseIR(buffer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(module.Dispose)
	return module
}
