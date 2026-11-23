package crosscompile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// The RISC-V newlib setjmp.S saves and restores the F registers in the
// hard-float ABI case with GNU-as displacement arithmetic
// ("FREG_S fs0, 14*SZREG+ 0*SZFREG(a0)"). The LLVM integrated assembler does
// not fold such displacement expressions, so hard-float targets fail to
// assemble the file; soft-float targets preprocess the whole block out and
// never hit the problem, which is why the arithmetic survived unnoticed.
// Rewrite the block to literal RV32 byte offsets (the GNU-as expression
// evaluates to 56 + 4k with SZREG = SZFREG = 4). llgo compiles this file for
// riscv32 only (riscv64 profiles do not use newlib-esp32). Idempotent: the
// rewrite runs only while the arithmetic form is still present.
var newlibRVSetjmpArith = regexp.MustCompile(
	`(FREG_[SL] fs(\d+), ?)14\*SZREG\+\s?(\d+)\*SZFREG\(a0\)`)

func patchNewlibRVSetjmp(sourceDir string) error {
	path := filepath.Join(sourceDir, "newlib", "libc", "machine", "riscv", "setjmp.S")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Test environments use fake source dirs; nothing to patch.
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if !newlibRVSetjmpArith.Match(data) {
		return nil // already patched (or fixed upstream)
	}

	count := 0
	patched := newlibRVSetjmpArith.ReplaceAllStringFunc(string(data), func(line string) string {
		m := newlibRVSetjmpArith.FindStringSubmatch(line)
		off := 56 + 4*atoi(m[3])
		comment := ""
		if count == 0 {
			comment = " /* literal GNU-as 14*SZREG+k*SZFREG displacement: LLVM's\n" +
				" * integrated assembler does not fold it */"
		}
		count++
		return m[1] + strconv.Itoa(off) + "(a0)" + comment
	})
	if count == 0 {
		return fmt.Errorf("patch %s: arithmetic displacements not recognized", path)
	}

	tmp := path + ".patch-tmp"
	if err := os.WriteFile(tmp, []byte(patched), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	return os.Rename(tmp, path)
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
