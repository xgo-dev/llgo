package rtlib

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/llgo/internal/crosscompile/compile"
)

func joinFileList(root, files string) []string {
	items := strings.Fields(files)
	paths := make([]string, 0, len(items))
	prefix := root
	if prefix != "" {
		prefix = filepath.Clean(prefix)
		if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
			prefix += string(os.PathSeparator)
		}
	}
	for _, item := range items {
		paths = append(paths, prefix+filepath.FromSlash(item))
	}
	return paths
}

func platformSpecifiedFiles(builtinsDir, target string) []string {
	switch {
	case strings.Contains(target, "riscv32"):
		files := joinFileList(builtinsDir, `
riscv/mulsi3.S
riscv/fp_mode.c
riscv/save.S
riscv/restore.S
		`)
		// Only add atomic.c for non-ESP targets (ESP doesn't support A extension)
		if target != "riscv32-esp-elf" {
			files = append(files, filepath.Join(builtinsDir, "atomic.c"))
		}
		return files
	case strings.Contains(target, "riscv64"):
		return joinFileList(builtinsDir, `
addtf3.c
comparetf2.c
divtc3.c
divtf3.c
extenddftf2.c
extendhftf2.c
extendsftf2.c
fixtfdi.c
fixtfsi.c
fixtfti.c
fixunstfdi.c
fixunstfsi.c
fixunstfti.c
floatditf.c
floatsitf.c
floattitf.c
floatunditf.c
floatunsitf.c
floatuntitf.c
multc3.c
multf3.c
powitf2.c
subtf3.c
trunctfdf2.c
trunctfhf2.c
trunctfsf2.c
atomic.c
		`)
	case strings.Contains(target, "arm"):
		return joinFileList(builtinsDir, `
arm/aeabi_cdcmp.S
arm/aeabi_cdcmpeq_check_nan.c
arm/aeabi_cfcmp.S
arm/aeabi_cfcmpeq_check_nan.c
arm/aeabi_dcmp.S
arm/aeabi_div0.c
arm/aeabi_drsub.c
arm/aeabi_fcmp.S
arm/aeabi_frsub.c
arm/aeabi_idivmod.S
arm/aeabi_ldivmod.S
arm/aeabi_memcmp.S
arm/aeabi_memcpy.S
arm/aeabi_memmove.S
arm/aeabi_memset.S
arm/aeabi_uidivmod.S
arm/aeabi_uldivmod.S
divmodsi4.c
udivmodsi4.c
		`)
	case strings.Contains(target, "avr"):
		return joinFileList(builtinsDir, `
avr/divmodhi4.S
avr/divmodqi4.S
avr/mulhi3.S
avr/mulqi3.S
avr/udivmodhi4.S
avr/udivmodqi4.S
		`)

	case target == "xtensa":
		return joinFileList(builtinsDir, `
xtensa/ieee754_sqrtf.S
atomic.c
		`)
	}
	return nil
}

func withPlatformSpecifiedFiles(baseDir, target string, files []string) []string {
	builtinsDir := filepath.Join(baseDir, "lib", "builtins")
	return append(files, platformSpecifiedFiles(builtinsDir, target)...)
}

func GetCompilerRTConfig() compile.LibConfig {
	return compile.LibConfig{
		Name:           "compiler-rt",
		Url:            "https://github.com/goplus/compiler-rt/archive/refs/tags/xtensa_release_19.1.2.tar.gz",
		Version:        "xtensa_release_19.1.2",
		ResourceSubDir: "compiler-rt-xtensa_release_19.1.2",
	}
}

func GetCompilerRTCompileConfig(baseDir, target string) compile.CompileConfig {
	return compile.CompileConfig{
		Groups: []compile.CompileGroup{
			{
				OutputFileName: "libclang_builtins-" + target + ".a",
				Files: withPlatformSpecifiedFiles(baseDir, target, joinFileList(baseDir, `
lib/builtins/absvdi2.c
lib/builtins/absvsi2.c
lib/builtins/absvti2.c
lib/builtins/adddf3.c
lib/builtins/addsf3.c
lib/builtins/addvdi3.c
lib/builtins/addvsi3.c
lib/builtins/addvti3.c
lib/builtins/apple_versioning.c
lib/builtins/ashldi3.c
lib/builtins/ashlti3.c
lib/builtins/ashrdi3.c
lib/builtins/ashrti3.c
lib/builtins/bswapdi2.c
lib/builtins/bswapsi2.c
lib/builtins/clzdi2.c
lib/builtins/clzsi2.c
lib/builtins/clzti2.c
lib/builtins/cmpdi2.c
lib/builtins/cmpti2.c
lib/builtins/comparedf2.c
lib/builtins/comparesf2.c
lib/builtins/ctzdi2.c
lib/builtins/ctzsi2.c
lib/builtins/ctzti2.c
lib/builtins/divdc3.c
lib/builtins/divdf3.c
lib/builtins/divdi3.c
lib/builtins/divmoddi4.c
lib/builtins/divmodsi4.c
lib/builtins/divmodti4.c
lib/builtins/divsc3.c
lib/builtins/divsf3.c
lib/builtins/divsi3.c
lib/builtins/divti3.c
lib/builtins/extendsfdf2.c
lib/builtins/extendhfsf2.c
lib/builtins/ffsdi2.c
lib/builtins/ffssi2.c
lib/builtins/ffsti2.c
lib/builtins/fixdfdi.c
lib/builtins/fixdfsi.c
lib/builtins/fixdfti.c
lib/builtins/fixsfdi.c
lib/builtins/fixsfsi.c
lib/builtins/fixsfti.c
lib/builtins/fixunsdfdi.c
lib/builtins/fixunsdfsi.c
lib/builtins/fixunsdfti.c
lib/builtins/fixunssfdi.c
lib/builtins/fixunssfsi.c
lib/builtins/fixunssfti.c
lib/builtins/floatdidf.c
lib/builtins/floatdisf.c
lib/builtins/floatsidf.c
lib/builtins/floatsisf.c
lib/builtins/floattidf.c
lib/builtins/floattisf.c
lib/builtins/floatundidf.c
lib/builtins/floatundisf.c
lib/builtins/floatunsidf.c
lib/builtins/floatunsisf.c
lib/builtins/floatuntidf.c
lib/builtins/floatuntisf.c
lib/builtins/fp_mode.c
lib/builtins/int_util.c
lib/builtins/lshrdi3.c
lib/builtins/lshrti3.c
lib/builtins/moddi3.c
lib/builtins/modsi3.c
lib/builtins/modti3.c
lib/builtins/muldc3.c
lib/builtins/muldf3.c
lib/builtins/muldi3.c
lib/builtins/mulodi4.c
lib/builtins/mulosi4.c
lib/builtins/muloti4.c
lib/builtins/mulsc3.c
lib/builtins/mulsf3.c
lib/builtins/multi3.c
lib/builtins/mulvdi3.c
lib/builtins/mulvsi3.c
lib/builtins/mulvti3.c
lib/builtins/negdf2.c
lib/builtins/negdi2.c
lib/builtins/negsf2.c
lib/builtins/negti2.c
lib/builtins/negvdi2.c
lib/builtins/negvsi2.c
lib/builtins/negvti2.c
lib/builtins/os_version_check.c
lib/builtins/paritydi2.c
lib/builtins/paritysi2.c
lib/builtins/parityti2.c
lib/builtins/popcountdi2.c
lib/builtins/popcountsi2.c
lib/builtins/popcountti2.c
lib/builtins/powidf2.c
lib/builtins/powisf2.c
lib/builtins/subdf3.c
lib/builtins/subsf3.c
lib/builtins/subvdi3.c
lib/builtins/subvsi3.c
lib/builtins/subvti3.c
lib/builtins/trampoline_setup.c
lib/builtins/truncdfhf2.c
lib/builtins/truncdfsf2.c
lib/builtins/truncsfhf2.c
lib/builtins/ucmpdi2.c
lib/builtins/ucmpti2.c
lib/builtins/udivdi3.c
lib/builtins/udivmoddi4.c
lib/builtins/udivmodsi4.c
lib/builtins/udivmodti4.c
lib/builtins/udivsi3.c
lib/builtins/udivti3.c
lib/builtins/umoddi3.c
lib/builtins/umodsi3.c
lib/builtins/umodti3.c
lib/builtins/gcc_personality_v0.c
lib/builtins/clear_cache.c
lib/builtins/addtf3.c
lib/builtins/comparetf2.c
lib/builtins/divtc3.c
lib/builtins/divtf3.c
lib/builtins/extenddftf2.c
lib/builtins/extendhftf2.c
lib/builtins/extendsftf2.c
lib/builtins/fixtfdi.c
lib/builtins/fixtfsi.c
lib/builtins/fixtfti.c
lib/builtins/fixunstfdi.c
lib/builtins/fixunstfsi.c
lib/builtins/fixunstfti.c
lib/builtins/floatditf.c
lib/builtins/floatsitf.c
lib/builtins/floattitf.c
lib/builtins/floatunditf.c
lib/builtins/floatunsitf.c
lib/builtins/floatuntitf.c
lib/builtins/multc3.c
lib/builtins/multf3.c
lib/builtins/powitf2.c
lib/builtins/subtf3.c
lib/builtins/trunctfdf2.c
lib/builtins/trunctfhf2.c
lib/builtins/trunctfsf2.c
				`)),
				CFlags: []string{
					"-DNDEBUG",
					"-DVISIBILITY_HIDDEN",
				},
				CCFlags: []string{
					"-Oz",
					"-fno-ident",
					"-Wno-unused-parameter",
					"-fno-lto",
					"-Werror=array-bounds",
					"-Werror=uninitialized",
					"-Werror=shadow",
					"-Werror=empty-body",
					"-Werror=sizeof-pointer-memaccess",
					"-Werror=sizeof-array-argument",
					"-Werror=suspicious-memaccess",
					"-Werror=builtin-memcpy-chk-size",
					"-Werror=array-bounds-pointer-arithmetic",
					"-Werror=return-stack-address",
					"-Werror=sizeof-array-decay",
					"-Werror=format-insufficient-args",
					"-Wformat",
					"-std=c11",
					"-fno-builtin",
					"-fvisibility=hidden",
					"-fomit-frame-pointer",
				},
			},
		},
	}
}
