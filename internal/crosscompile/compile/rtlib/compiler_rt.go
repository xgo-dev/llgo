package rtlib

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile/compile"
)

func platformSpecifiedFiles(builtinsDir, target string) []string {
	switch {
	case strings.Contains(target, "riscv32"):
		files := []string{
			filepath.Join(builtinsDir, "riscv", "mulsi3.S"),
			filepath.Join(builtinsDir, "riscv", "fp_mode.c"),
			filepath.Join(builtinsDir, "riscv", "save.S"),
			filepath.Join(builtinsDir, "riscv", "restore.S"),
		}
		// Only add atomic.c for non-ESP targets (ESP doesn't support A extension)
		if target != "riscv32-esp-elf" {
			files = append(files, filepath.Join(builtinsDir, "atomic.c"))
		}
		return files
	case strings.Contains(target, "riscv64"):
		return []string{
			filepath.Join(builtinsDir, "addtf3.c"),
			filepath.Join(builtinsDir, "comparetf2.c"),
			filepath.Join(builtinsDir, "divtc3.c"),
			filepath.Join(builtinsDir, "divtf3.c"),
			filepath.Join(builtinsDir, "extenddftf2.c"),
			filepath.Join(builtinsDir, "extendhftf2.c"),
			filepath.Join(builtinsDir, "extendsftf2.c"),
			filepath.Join(builtinsDir, "fixtfdi.c"),
			filepath.Join(builtinsDir, "fixtfsi.c"),
			filepath.Join(builtinsDir, "fixtfti.c"),
			filepath.Join(builtinsDir, "fixunstfdi.c"),
			filepath.Join(builtinsDir, "fixunstfsi.c"),
			filepath.Join(builtinsDir, "fixunstfti.c"),
			filepath.Join(builtinsDir, "floatditf.c"),
			filepath.Join(builtinsDir, "floatsitf.c"),
			filepath.Join(builtinsDir, "floattitf.c"),
			filepath.Join(builtinsDir, "floatunditf.c"),
			filepath.Join(builtinsDir, "floatunsitf.c"),
			filepath.Join(builtinsDir, "floatuntitf.c"),
			filepath.Join(builtinsDir, "multc3.c"),
			filepath.Join(builtinsDir, "multf3.c"),
			filepath.Join(builtinsDir, "powitf2.c"),
			filepath.Join(builtinsDir, "subtf3.c"),
			filepath.Join(builtinsDir, "trunctfdf2.c"),
			filepath.Join(builtinsDir, "trunctfhf2.c"),
			filepath.Join(builtinsDir, "trunctfsf2.c"),
			filepath.Join(builtinsDir, "atomic.c"),
		}
	case strings.Contains(target, "arm"):
		return []string{
			filepath.Join(builtinsDir, "arm", "aeabi_cdcmp.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_cdcmpeq_check_nan.c"),
			filepath.Join(builtinsDir, "arm", "aeabi_cfcmp.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_cfcmpeq_check_nan.c"),
			filepath.Join(builtinsDir, "arm", "aeabi_dcmp.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_div0.c"),
			filepath.Join(builtinsDir, "arm", "aeabi_drsub.c"),
			filepath.Join(builtinsDir, "arm", "aeabi_fcmp.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_frsub.c"),
			filepath.Join(builtinsDir, "arm", "aeabi_idivmod.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_ldivmod.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_memcmp.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_memcpy.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_memmove.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_memset.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_uidivmod.S"),
			filepath.Join(builtinsDir, "arm", "aeabi_uldivmod.S"),

			// These two are not technically EABI builtins but are used by them and only
			// seem to be used on ARM. LLVM seems to use __divsi3 and __modsi3 on most
			// other architectures.
			// Most importantly, they have a different calling convention on AVR so
			// should not be used on AVR.
			filepath.Join(builtinsDir, "divmodsi4.c"),
			filepath.Join(builtinsDir, "udivmodsi4.c"),
		}
	case strings.Contains(target, "avr"):
		return []string{
			filepath.Join(builtinsDir, "avr", "divmodhi4.S"),
			filepath.Join(builtinsDir, "avr", "divmodqi4.S"),
			filepath.Join(builtinsDir, "avr", "mulhi3.S"),
			filepath.Join(builtinsDir, "avr", "mulqi3.S"),
			filepath.Join(builtinsDir, "avr", "udivmodhi4.S"),
			filepath.Join(builtinsDir, "avr", "udivmodqi4.S"),
		}

	case target == "xtensa":
		return []string{
			filepath.Join(builtinsDir, "xtensa", "ieee754_sqrtf.S"),
			filepath.Join(builtinsDir, "atomic.c"),
		}
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
				OutputFileName: fmt.Sprintf("libclang_builtins-%s.a", target),
				Files: withPlatformSpecifiedFiles(baseDir, target, []string{
					filepath.Join(baseDir, "lib", "builtins", "absvdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "absvsi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "absvti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "adddf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "addsf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "addvdi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "addvsi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "addvti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "apple_versioning.c"),
					filepath.Join(baseDir, "lib", "builtins", "ashldi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "ashlti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "ashrdi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "ashrti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "bswapdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "bswapsi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "clzdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "clzsi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "clzti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "cmpdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "cmpti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "comparedf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "comparesf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "ctzdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "ctzsi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "ctzti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "divdc3.c"),
					filepath.Join(baseDir, "lib", "builtins", "divdf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "divdi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "divmoddi4.c"),
					filepath.Join(baseDir, "lib", "builtins", "divmodsi4.c"),
					filepath.Join(baseDir, "lib", "builtins", "divmodti4.c"),
					filepath.Join(baseDir, "lib", "builtins", "divsc3.c"),
					filepath.Join(baseDir, "lib", "builtins", "divsf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "divsi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "divti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "extendsfdf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "extendhfsf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "ffsdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "ffssi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "ffsti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixdfdi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixdfsi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixdfti.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixsfdi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixsfsi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixsfti.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunsdfdi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunsdfsi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunsdfti.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunssfdi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunssfsi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunssfti.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatdidf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatdisf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatsidf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatsisf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floattidf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floattisf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatundidf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatundisf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatunsidf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatunsisf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatuntidf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatuntisf.c"),
					filepath.Join(baseDir, "lib", "builtins", "fp_mode.c"),
					filepath.Join(baseDir, "lib", "builtins", "int_util.c"),
					filepath.Join(baseDir, "lib", "builtins", "lshrdi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "lshrti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "moddi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "modsi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "modti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "muldc3.c"),
					filepath.Join(baseDir, "lib", "builtins", "muldf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "muldi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "mulodi4.c"),
					filepath.Join(baseDir, "lib", "builtins", "mulosi4.c"),
					filepath.Join(baseDir, "lib", "builtins", "muloti4.c"),
					filepath.Join(baseDir, "lib", "builtins", "mulsc3.c"),
					filepath.Join(baseDir, "lib", "builtins", "mulsf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "multi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "mulvdi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "mulvsi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "mulvti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "negdf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "negdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "negsf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "negti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "negvdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "negvsi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "negvti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "os_version_check.c"),
					filepath.Join(baseDir, "lib", "builtins", "paritydi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "paritysi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "parityti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "popcountdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "popcountsi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "popcountti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "powidf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "powisf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "subdf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "subsf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "subvdi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "subvsi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "subvti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "trampoline_setup.c"),
					filepath.Join(baseDir, "lib", "builtins", "truncdfhf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "truncdfsf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "truncsfhf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "ucmpdi2.c"),
					filepath.Join(baseDir, "lib", "builtins", "ucmpti2.c"),
					filepath.Join(baseDir, "lib", "builtins", "udivdi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "udivmoddi4.c"),
					filepath.Join(baseDir, "lib", "builtins", "udivmodsi4.c"),
					filepath.Join(baseDir, "lib", "builtins", "udivmodti4.c"),
					filepath.Join(baseDir, "lib", "builtins", "udivsi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "udivti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "umoddi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "umodsi3.c"),
					filepath.Join(baseDir, "lib", "builtins", "umodti3.c"),
					filepath.Join(baseDir, "lib", "builtins", "gcc_personality_v0.c"),
					filepath.Join(baseDir, "lib", "builtins", "clear_cache.c"),
					filepath.Join(baseDir, "lib", "builtins", "addtf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "comparetf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "divtc3.c"),
					filepath.Join(baseDir, "lib", "builtins", "divtf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "extenddftf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "extendhftf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "extendsftf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixtfdi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixtfsi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixtfti.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunstfdi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunstfsi.c"),
					filepath.Join(baseDir, "lib", "builtins", "fixunstfti.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatditf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatsitf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floattitf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatunditf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatunsitf.c"),
					filepath.Join(baseDir, "lib", "builtins", "floatuntitf.c"),
					filepath.Join(baseDir, "lib", "builtins", "multc3.c"),
					filepath.Join(baseDir, "lib", "builtins", "multf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "powitf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "subtf3.c"),
					filepath.Join(baseDir, "lib", "builtins", "trunctfdf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "trunctfhf2.c"),
					filepath.Join(baseDir, "lib", "builtins", "trunctfsf2.c"),
				}),
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
