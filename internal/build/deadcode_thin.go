package build

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/dcepass"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/xtool/ar"
	"github.com/xgo-dev/llvm"
)

// Initially restrict the experiment to native Mach-O size builds. DWARF uses
// persistent archive paths in its debug map, and other platforms have different
// archive formats. Multi-root builds must never share program-specific rewrites.
func useDeadcodeThinArchives(ctx *context) bool {
	if ctx == nil || ctx.buildConf == nil || runtime.GOOS != "darwin" {
		return false
	}
	c := ctx.buildConf
	return c.deadcodeDropEnabled() && c.ltoMode() == lto.Thin &&
		c.BuildMode == BuildModeExe && c.Target == "" && !c.GenLL &&
		c.Goos == runtime.GOOS && c.Goarch == runtime.GOARCH &&
		c.DebugArtifactMode == DebugArtifactNone && len(ctx.initial) == 1 &&
		(ctx.mode == ModeBuild || ctx.mode == ModeTest)
}

// rewriteDeadcodeThinArchives never changes package archives or frontend
// modules. Cached and freshly emitted archives follow the identical path, and
// the returned directory belongs to this link (including a possible PCLN relink).
func rewriteDeadcodeThinArchives(ctx *context, inputs []string, live map[string][]int, verbose bool) ([]string, string, error) {
	dir, err := os.MkdirTemp("", "llgo-deadcode-thin-*")
	if err != nil {
		return nil, "", err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(dir)
		}
	}()
	out := append([]string(nil), inputs...)
	total := 0
	for i, input := range inputs {
		work := filepath.Join(dir, strconv.Itoa(i))
		if err := os.Mkdir(work, 0o755); err != nil {
			return nil, "", err
		}
		archive := filepath.Join(work, "package.a")
		count, err := rewriteDeadcodeThinArchive(ctx, input, archive, work, live, verbose)
		if err != nil {
			return nil, "", fmt.Errorf("ThinLTO dead-code rewrite of %s: %w", input, err)
		}
		if count != 0 {
			out[i] = archive
			total += count
		}
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[dce] ThinLTO package summaries: rewrote %d method slots\n", total)
	}
	success = true
	return out, dir, nil
}

func rewriteDeadcodeThinArchive(ctx *context, input, output, work string, live map[string][]int, verbose bool) (int, error) {
	f, err := os.Open(input)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	r, err := ar.NewReader(f)
	if err != nil {
		return 0, err
	}
	var members aPackage
	defer members.disposeArchiveBuffers()
	var ordered []llvm.ArchiveMember
	changed := 0
	for i := 0; ; i++ {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		name := strings.TrimRight(header.Name, "\x00")
		if strings.HasPrefix(name, "__.SYMDEF") {
			continue // The archive writer regenerates the symbol index.
		}
		if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
			return 0, fmt.Errorf("unexpected Mach-O archive member %q", name)
		}
		memberDir := filepath.Join(work, strconv.Itoa(i))
		if err := os.Mkdir(memberDir, 0o755); err != nil {
			return 0, err
		}
		path := filepath.Join(memberDir, name)
		data, err := io.ReadAll(r)
		if err != nil {
			return 0, err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return 0, err
		}
		members.ObjFiles = append(members.ObjFiles, path)
		if isLLVMBitcode(data) {
			buffer, count, err := rewriteDeadcodeThinMember(path, live)
			if err != nil {
				return 0, fmt.Errorf("%s: %w", name, err)
			}
			if count != 0 {
				changed += count
				members.ObjBuffers = append(members.ObjBuffers, packageArchiveBuffer{name: name, buffer: buffer})
				ordered = append(ordered, llvm.NewArchiveMemberFromMemoryBuffer(name, buffer))
				if useExternalPackageArchiver {
					if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
						return 0, err
					}
				}
				continue
			}
		}
		// Native C/assembly members and untouched bitcode retain their bytes.
		ordered = append(ordered, llvm.NewArchiveMemberFromFile(path))
	}
	if changed != 0 {
		// Keep member ordering: moving weak definitions across native members
		// can change which definition archive extraction selects.
		if useExternalPackageArchiver {
			err = ctx.createArchiveFile(output, members.ObjFiles, verbose)
		} else {
			err = llvm.WriteArchive(output, ctx.targetTriple(), ordered)
		}
		if err != nil {
			return 0, err
		}
	}
	return changed, nil
}

func isLLVMBitcode(data []byte) bool {
	return bytes.HasPrefix(data, []byte{'B', 'C', 0xc0, 0xde}) ||
		bytes.HasPrefix(data, []byte{0xde, 0xc0, 0x17, 0x0b})
}

func rewriteDeadcodeThinMember(path string, live map[string][]int) (llvm.MemoryBuffer, int, error) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod, err := ctx.ParseBitcodeFile(path)
	if err != nil {
		return llvm.MemoryBuffer{}, 0, err
	}
	defer mod.Dispose()
	count := dcepass.RewriteMethodSlots(mod, live)
	if count == 0 {
		return llvm.MemoryBuffer{}, 0, nil
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		return llvm.MemoryBuffer{}, 0, err
	}
	// Recompute the summary from the rewritten IR; retaining the old index
	// would preserve the very method edges this pass is intended to remove.
	buffer, err := emitThinLTOBitcode(mod, llvm.TargetMachine{})
	return buffer, count, err
}
