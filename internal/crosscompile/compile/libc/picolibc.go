package libc

import (
	"path/filepath"

	"github.com/goplus/llgo/internal/crosscompile/compile"
)

// GetPicolibcConfig returns the configuration for downloading and building picolibc
func GetPicolibcConfig() compile.LibConfig {
	return compile.LibConfig{
		Name:           "picolibc",
		Version:        "v0.1.0",
		Url:            "https://github.com/goplus/picolibc/archive/refs/heads/main.zip",
		ResourceSubDir: "picolibc-main",
	}
}

// GetPicolibcCompileConfig returns configuration for picolibc
func GetPicolibcCompileConfig(baseDir, target string) compile.CompileConfig {
	return compile.CompileConfig{
		ExportCFlags: []string{
			"-I" + baseDir,
			"-isystem" + filepath.Join(baseDir, "newlib", "libc", "include"),
		},
		Groups: []compile.CompileGroup{
			{
				OutputFileName: "libc-" + target + ".a",
				Files: joinFileList(baseDir, `
newlib/libc/string/bcmp.c
newlib/libc/string/bcopy.c
newlib/libc/string/bzero.c
newlib/libc/string/explicit_bzero.c
newlib/libc/string/ffsl.c
newlib/libc/string/ffsll.c
newlib/libc/string/fls.c
newlib/libc/string/flsl.c
newlib/libc/string/flsll.c
newlib/libc/string/gnu_basename.c
newlib/libc/string/index.c
newlib/libc/string/memccpy.c
newlib/libc/string/memchr.c
newlib/libc/string/memcmp.c
newlib/libc/string/memcpy.c
newlib/libc/string/memmem.c
newlib/libc/string/memmove.c
newlib/libc/string/mempcpy.c
newlib/libc/string/memrchr.c
newlib/libc/string/memset.c
newlib/libc/string/rawmemchr.c
newlib/libc/string/rindex.c
newlib/libc/string/stpcpy.c
newlib/libc/string/stpncpy.c
newlib/libc/string/strcasecmp.c
newlib/libc/string/strcasecmp_l.c
newlib/libc/string/strcasestr.c
newlib/libc/string/strcat.c
newlib/libc/string/strchr.c
newlib/libc/string/strchrnul.c
newlib/libc/string/strcmp.c
newlib/libc/string/strcoll.c
newlib/libc/string/strcoll_l.c
newlib/libc/string/strcpy.c
newlib/libc/string/strcspn.c
newlib/libc/string/strerror_r.c
newlib/libc/string/strlcat.c
newlib/libc/string/strlcpy.c
newlib/libc/string/strlen.c
newlib/libc/string/strlwr.c
newlib/libc/string/strncasecmp.c
newlib/libc/string/strncasecmp_l.c
newlib/libc/string/strncat.c
newlib/libc/string/strncmp.c
newlib/libc/string/strncpy.c
newlib/libc/string/strndup.c
newlib/libc/string/strnlen.c
newlib/libc/string/strnstr.c
newlib/libc/string/strpbrk.c
newlib/libc/string/strrchr.c
newlib/libc/string/strsep.c
newlib/libc/string/strsignal.c
newlib/libc/string/strspn.c
newlib/libc/string/strstr.c
newlib/libc/string/strtok.c
newlib/libc/string/strtok_r.c
newlib/libc/string/strupr.c
newlib/libc/string/strverscmp.c
newlib/libc/string/strxfrm.c
newlib/libc/string/strxfrm_l.c
newlib/libc/string/swab.c
newlib/libc/string/timingsafe_bcmp.c
newlib/libc/string/timingsafe_memcmp.c
newlib/libc/string/strerror.c
newlib/libc/string/wcpcpy.c
newlib/libc/string/wcpncpy.c
newlib/libc/string/wcscasecmp.c
newlib/libc/string/wcscasecmp_l.c
newlib/libc/string/wcscat.c
newlib/libc/string/wcschr.c
newlib/libc/string/wcscmp.c
newlib/libc/string/wcscoll.c
newlib/libc/string/wcscoll_l.c
newlib/libc/string/wcscpy.c
newlib/libc/string/wcscspn.c
newlib/libc/string/wcsdup.c
newlib/libc/string/wcslcat.c
newlib/libc/string/wcslcpy.c
newlib/libc/string/wcslen.c
newlib/libc/string/wcsncasecmp.c
newlib/libc/string/wcsncasecmp_l.c
newlib/libc/string/wcsncat.c
newlib/libc/string/wcsncmp.c
newlib/libc/string/wcsncpy.c
newlib/libc/string/wcsnlen.c
newlib/libc/string/wcspbrk.c
newlib/libc/string/wcsrchr.c
newlib/libc/string/wcsspn.c
newlib/libc/string/wcsstr.c
newlib/libc/string/wcstok.c
newlib/libc/string/wcswidth.c
newlib/libc/string/wcsxfrm.c
newlib/libc/string/wcsxfrm_l.c
newlib/libc/string/wcwidth.c
newlib/libc/string/wmemchr.c
newlib/libc/string/wmemcmp.c
newlib/libc/string/wmemcpy.c
newlib/libc/string/wmemmove.c
newlib/libc/string/wmempcpy.c
newlib/libc/string/wmemset.c
newlib/libc/string/xpg_strerror_r.c

newlib/libc/stdlib/nano-calloc.c
newlib/libc/stdlib/nano-malloc.c
newlib/libc/stdlib/nano-pvalloc.c
newlib/libc/stdlib/nano-realloc.c
newlib/libc/stdlib/nano-valloc.c
newlib/libc/stdlib/rand.c
newlib/libc/stdlib/srand.c
newlib/libc/stdlib/nano-free.c

newlib/libc/tinystdio/printf.c
newlib/libc/tinystdio/putchar.c
newlib/libc/tinystdio/puts.c
				`),
				CFlags: []string{
					"-D_COMPILING_NEWLIB",
					"-D_HAVE_ALIAS_ATTRIBUTE",
					"-DTINY_STDIO",
					"-DPOSIX_IO",
					"-DFORMAT_DEFAULT_INTEGER",
					"-D_IEEE_LIBM",
					"-D__OBSOLETE_MATH_FLOAT=1",
					"-D__OBSOLETE_MATH_DOUBLE=0",
					"-D_WANT_IO_C99_FORMATS",
					"-nostdlib",
					"-I" + baseDir,
					"-isystem" + filepath.Join(baseDir, "newlib", "libc", "include"),
					"-I" + filepath.Join(baseDir, "newlib", "libm", "common"),
					"-I" + filepath.Join(baseDir, "newlib", "libc", "locale"),

					"-I" + filepath.Join(baseDir, "newlib", "libc", "tinystdio"),
				},
				LDFlags: _libcLDFlags,
				CCFlags: _libcCCFlags,
			},
		},
	}
}
