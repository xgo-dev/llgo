package llvmfmv

import (
	"fmt"
	"github.com/xgo-dev/llvm"
)

const registerWidthAttribute = attributePrefix + "register-width"
const registerBaselineAttribute = attributePrefix + "register-baseline"

// Retain the per-function compilation floor through ThinLTO importing and
// full LTO merging. Do not infer a baseline from an unrelated input module.
func RecordRegisterBaseline(m llvm.Module, arch, baseline string) {
	if arch != "amd64" || (baseline != "v3" && baseline != "v4") {
		return
	}
	attr := m.Context().CreateStringAttribute(registerBaselineAttribute, baseline)
	for f := m.FirstFunction(); !f.IsNil(); f = llvm.NextFunction(f) {
		if !f.IsDeclaration() {
			f.AddFunctionAttr(attr)
		}
	}
}

// VerifyRegisterCalls checks surviving direct edges by walking the users of
// marked ABI entries. This is independent of symbol spelling and O-level.
func VerifyRegisterCalls(m llvm.Module) error {
	for target := m.FirstFunction(); !target.IsNil(); target = llvm.NextFunction(target) {
		tag := stringAttribute(target, registerWidthAttribute)
		if tag == "" {
			continue
		}
		width := 256
		switch tag {
		case "256":
		case "512":
			width = 512
		default:
			return fmt.Errorf("invalid SIMD register width %q on %s", tag, target.Name())
		}
		if registerWidth(target, stringAttribute(target, registerBaselineAttribute)) < width {
			return fmt.Errorf("SIMD register entry %s lacks %d-bit features", target.Name(), width)
		}
		queue := []llvm.Value{target}
		seen := map[llvm.Value]bool{target: true}
		for len(queue) > 0 {
			value := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			for u := value.FirstUse(); !u.IsNil(); u = u.NextUse() {
				user := u.User()
				if !user.IsAGlobalAlias().IsNil() || (!user.IsAConstantExpr().IsNil() && user.Opcode() == llvm.BitCast) {
					if !seen[user] {
						seen[user] = true
						queue = append(queue, user)
					}
					continue
				}
				if (user.IsACallInst().IsNil() && user.IsAInvokeInst().IsNil()) || user.CalledValue() != value {
					continue
				}
				caller := user.InstructionParent().Parent()
				if registerWidth(caller, stringAttribute(caller, registerBaselineAttribute)) < width {
					return fmt.Errorf("SIMD register call %s -> %s requires %d-bit features", caller.Name(), target.Name(), width)
				}
			}
		}
	}
	return nil
}
