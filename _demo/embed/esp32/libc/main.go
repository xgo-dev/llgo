package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
	"github.com/xgo-dev/llgo/_demo/embed/esp32/watchdog"
)

//
//go:linkname strlen C.strlen
func strlen(str *c.Char) c.SizeT

//go:linkname strcpy C.strcpy
func strcpy(dest, src *c.Char) *c.Char

//go:linkname strncpy C.strncpy
func strncpy(dest, src *c.Char, n c.SizeT) *c.Char

//go:linkname strcat C.strcat
func strcat(dest, src *c.Char) *c.Char

//go:linkname strncat C.strncat
func strncat(dest, src *c.Char, n c.SizeT) *c.Char

//go:linkname strcmp C.strcmp
func strcmp(s1, s2 *c.Char) c.Int

//go:linkname strncmp C.strncmp
func strncmp(s1, s2 *c.Char, n c.SizeT) c.Int

//go:linkname strchr C.strchr
func strchr(s *c.Char, c c.Int) *c.Char

//go:linkname strrchr C.strrchr
func strrchr(s *c.Char, c c.Int) *c.Char

//go:linkname strstr C.strstr
func strstr(haystack, needle *c.Char) *c.Char

//go:linkname strtok C.strtok
func strtok(str, delim *c.Char) *c.Char

//go:linkname strtok_r C.strtok_r
func strtok_r(str, delim *c.Char, saveptr **c.Char) *c.Char

//go:linkname strspn C.strspn
func strspn(s, accept *c.Char) c.SizeT

//go:linkname strcspn C.strcspn
func strcspn(s, reject *c.Char) c.SizeT

//go:linkname strdup C.strdup
func strdup(s *c.Char) *c.Char

//go:linkname strndup C.strndup
func strndup(s *c.Char, n c.SizeT) *c.Char

//go:linkname memcpy C.memcpy
func memcpy(dest, src unsafe.Pointer, n c.SizeT) unsafe.Pointer

//go:linkname memmove C.memmove
func memmove(dest, src unsafe.Pointer, n c.SizeT) unsafe.Pointer

//go:linkname memset C.memset
func memset(s unsafe.Pointer, c c.Int, n c.SizeT) unsafe.Pointer

//go:linkname memcmp C.memcmp
func memcmp(s1, s2 unsafe.Pointer, n c.SizeT) c.Int

//go:linkname memchr C.memchr
func memchr(s unsafe.Pointer, c c.Int, n c.SizeT) unsafe.Pointer

//go:linkname memrchr C.memrchr
func memrchr(s unsafe.Pointer, c c.Int, n c.SizeT) unsafe.Pointer

//go:linkname bcmp C.bcmp
func bcmp(s1, s2 unsafe.Pointer, n c.SizeT) c.Int

//go:linkname bcopy C.bcopy
func bcopy(src, dest unsafe.Pointer, n c.SizeT)

//go:linkname bzero C.bzero
func bzero(s unsafe.Pointer, n c.SizeT)

//go:linkname explicit_bzero C.explicit_bzero
func explicit_bzero(s unsafe.Pointer, n c.SizeT)

//go:linkname ffs C.ffs
func ffs(i c.Int) c.Int

//go:linkname ffsl C.ffsl
func ffsl(i c.Long) c.Int

//go:linkname ffsll C.ffsll
func ffsll(i c.LongLong) c.Int

//go:linkname fls C.fls
func fls(i c.Int) c.Int

//go:linkname flsl C.flsl
func flsl(i c.Long) c.Int

//go:linkname flsll C.flsll
func flsll(i c.LongLong) c.Int

//go:linkname index C.index
func index(s *c.Char, c c.Int) *c.Char

//go:linkname rindex C.rindex
func rindex(s *c.Char, c c.Int) *c.Char

//go:linkname strcasecmp C.strcasecmp
func strcasecmp(s1, s2 *c.Char) c.Int

//go:linkname strncasecmp C.strncasecmp
func strncasecmp(s1, s2 *c.Char, n c.SizeT) c.Int

//go:linkname strlcat C.strlcat
func strlcat(dst, src *c.Char, size c.SizeT) c.SizeT

//go:linkname strlcpy C.strlcpy
func strlcpy(dst, src *c.Char, size c.SizeT) c.SizeT

//go:linkname strsep C.strsep
func strsep(stringp **c.Char, delim *c.Char) *c.Char

//go:linkname strcoll C.strcoll
func strcoll(s1, s2 *c.Char) c.Int

//go:linkname strxfrm C.strxfrm
func strxfrm(dest, src *c.Char, n c.SizeT) c.SizeT

//go:linkname strerror C.strerror
func strerror(errnum c.Int) *c.Char

//go:linkname strerror_r C.strerror_r
func strerror_r(errnum c.Int, buf *c.Char, buflen c.SizeT) c.Int

//go:linkname strverscmp C.strverscmp
func strverscmp(s1, s2 *c.Char) c.Int

//
//go:linkname malloc C.malloc
func malloc(size c.SizeT) unsafe.Pointer

//go:linkname calloc C.calloc
func calloc(nmemb, size c.SizeT) unsafe.Pointer

//go:linkname realloc C.realloc
func realloc(ptr unsafe.Pointer, size c.SizeT) unsafe.Pointer

//go:linkname reallocarray C.reallocarray
func reallocarray(ptr unsafe.Pointer, nmemb, size c.SizeT) unsafe.Pointer

//go:linkname reallocf C.reallocf
func reallocf(ptr unsafe.Pointer, size c.SizeT) unsafe.Pointer

//go:linkname valloc C.valloc
func valloc(size c.SizeT) unsafe.Pointer

//go:linkname pvalloc C.pvalloc
func pvalloc(size c.SizeT) unsafe.Pointer

//go:linkname memalign C.memalign
func memalign(alignment, size c.SizeT) unsafe.Pointer

//go:linkname aligned_alloc C.aligned_alloc
func aligned_alloc(alignment, size c.SizeT) unsafe.Pointer

//go:linkname posix_memalign C.posix_memalign
func posix_memalign(memptr *unsafe.Pointer, alignment, size c.SizeT) c.Int

//go:linkname free C.free
func free(ptr unsafe.Pointer)

//go:linkname malloc_size C.malloc_size
func malloc_size(ptr unsafe.Pointer) c.SizeT

//go:linkname malloc_usable_size C.malloc_usable_size
func malloc_usable_size(ptr unsafe.Pointer) c.SizeT

//go:linkname malloc_stats C.malloc_stats
func malloc_stats()

//go:linkname malloc_info C.malloc_info
func malloc_info(options c.Int, stream c.FilePtr) c.Int

//go:linkname mallopt C.mallopt
func mallopt(param, value c.Int) c.Int

//
//go:linkname fopen C.fopen
func fopen(pathname, mode *c.Char) c.FilePtr

//go:linkname fdopen C.fdopen
func fdopen(fd c.Int, mode *c.Char) c.FilePtr

//go:linkname freopen C.freopen
func freopen(pathname, mode *c.Char, stream c.FilePtr) c.FilePtr

//go:linkname fclose C.fclose
func fclose(stream c.FilePtr) c.Int

//go:linkname fflush C.fflush
func fflush(stream c.FilePtr) c.Int

//go:linkname fseek C.fseek
func fseek(stream c.FilePtr, offset c.Long, whence c.Int) c.Int

//go:linkname ftell C.ftell
func ftell(stream c.FilePtr) c.Long

// //go:linkname fseeko C.fseeko
// func fseeko(stream c.FilePtr, offset C.off_t, whence c.Int) c.Int

// //go:linkname ftello C.ftello
// func ftello(stream c.FilePtr) C.off_t

//go:linkname rewind C.rewind
func rewind(stream c.FilePtr)

// //go:linkname fgetpos C.fgetpos
// func fgetpos(stream c.FilePtr, pos *C.fpos_t) c.Int

// //go:linkname fsetpos C.fsetpos
// func fsetpos(stream c.FilePtr, pos *C.fpos_t) c.Int

//go:linkname fread C.fread
func fread(ptr unsafe.Pointer, size, nmemb c.SizeT, stream c.FilePtr) c.SizeT

//go:linkname fwrite C.fwrite
func fwrite(ptr unsafe.Pointer, size, nmemb c.SizeT, stream c.FilePtr) c.SizeT

//go:linkname fgetc C.fgetc
func fgetc(stream c.FilePtr) c.Int

//go:linkname getc C.getc
func getc(stream c.FilePtr) c.Int

//go:linkname getchar C.getchar
func getchar() c.Int

//go:linkname ungetc C.ungetc
func ungetc(c c.Int, stream c.FilePtr) c.Int

//go:linkname fgets C.fgets
func fgets(s *c.Char, size c.Int, stream c.FilePtr) *c.Char

//go:linkname fputc C.fputc
func fputc(c c.Int, stream c.FilePtr) c.Int

//go:linkname putc C.putc
func putc(c c.Int, stream c.FilePtr) c.Int

//go:linkname putchar C.putchar
func putchar(c c.Int) c.Int

//go:linkname fputs C.fputs
func fputs(s *c.Char, stream c.FilePtr) c.Int

//go:linkname puts C.puts
func puts(s *c.Char) c.Int

//go:linkname printf C.printf
func printf(format *c.Char, __llgo_va_list ...any) c.Int

//go:linkname fprintf C.fprintf
func fprintf(stream c.FilePtr, format *c.Char, __llgo_va_list ...any) c.Int

//go:linkname sprintf C.sprintf
func sprintf(str, format *c.Char, __llgo_va_list ...any) c.Int

//go:linkname snprintf C.snprintf
func snprintf(str *c.Char, size c.SizeT, format *c.Char, __llgo_va_list ...any) c.Int

// //go:linkname vprintf C.vprintf
// func vprintf(format *c.Char, ap C.va_list) c.Int

// //go:linkname vfprintf C.vfprintf
// func vfprintf(stream c.FilePtr, format *c.Char, ap C.va_list) c.Int

// //go:linkname vsprintf C.vsprintf
// func vsprintf(str, format *c.Char, ap C.va_list) c.Int

// //go:linkname vsnprintf C.vsnprintf
// func vsnprintf(str *c.Char, size c.SizeT, format *c.Char, ap C.va_list) c.Int

//go:linkname scanf C.scanf
func scanf(format *c.Char, __llgo_va_list ...any) c.Int

//go:linkname fscanf C.fscanf
func fscanf(stream c.FilePtr, format *c.Char, __llgo_va_list ...any) c.Int

//go:linkname sscanf C.sscanf
func sscanf(str, format *c.Char, __llgo_va_list ...any) c.Int

//go:linkname remove C.remove
func remove(pathname *c.Char) c.Int

//go:linkname rename C.rename
func rename(oldpath, newpath *c.Char) c.Int

//go:linkname tmpfile C.tmpfile
func tmpfile() c.FilePtr

//go:linkname tmpnam C.tmpnam
func tmpnam(s *c.Char) *c.Char

//go:linkname tempnam C.tempnam
func tempnam(dir, pfx *c.Char) *c.Char

//go:linkname fileno C.fileno
func fileno(stream c.FilePtr) c.Int

//go:linkname feof C.feof
func feof(stream c.FilePtr) c.Int

//go:linkname ferror C.ferror
func ferror(stream c.FilePtr) c.Int

//go:linkname clearerr C.clearerr
func clearerr(stream c.FilePtr)

//go:linkname perror C.perror
func perror(s *c.Char)

//go:linkname setvbuf C.setvbuf
func setvbuf(stream c.FilePtr, buf *c.Char, mode c.Int, size c.SizeT) c.Int

//go:linkname setbuf C.setbuf
func setbuf(stream c.FilePtr, buf *c.Char)

//go:linkname setbuffer C.setbuffer
func setbuffer(stream c.FilePtr, buf *c.Char, size c.SizeT)

//go:linkname setlinebuf C.setlinebuf
func setlinebuf(stream c.FilePtr)

//go:linkname fcloseall C.fcloseall
func fcloseall() c.Int

//go:linkname fmemopen C.fmemopen
func fmemopen(buf unsafe.Pointer, size c.SizeT, mode *c.Char) c.FilePtr

//go:linkname open_memstream C.open_memstream
func open_memstream(ptr **c.Char, sizeloc *c.SizeT) c.FilePtr

//go:linkname fpurge C.fpurge
func fpurge(stream c.FilePtr) c.Int

//go:linkname __fpurge C.__fpurge
func __fpurge(stream c.FilePtr) c.Int

//
// //go:linkname acos C.acos
// func acos(x c.Double) c.Double

// //go:linkname asin C.asin
// func asin(x c.Double) c.Double

// //go:linkname atan C.atan
// func atan(x c.Double) c.Double

// //go:linkname atan2 C.atan2
// func atan2(y, x c.Double) c.Double

// //go:linkname cos C.cos
// func cos(x c.Double) c.Double

// //go:linkname sin C.sin
// func sin(x c.Double) c.Double

// //go:linkname tan C.tan
// func tan(x c.Double) c.Double

// //go:linkname cosh C.cosh
// func cosh(x c.Double) c.Double

// //go:linkname sinh C.sinh
// func sinh(x c.Double) c.Double

// //go:linkname tanh C.tanh
// func tanh(x c.Double) c.Double

// //go:linkname exp C.exp
// func exp(x c.Double) c.Double

// //go:linkname frexp C.frexp
// func frexp(value c.Double, exp *c.Int) c.Double

// //go:linkname ldexp C.ldexp
// func ldexp(x c.Double, exp c.Int) c.Double

// //go:linkname log C.log
// func log(x c.Double) c.Double

// //go:linkname log10 C.log10
// func log10(x c.Double) c.Double

// //go:linkname modf C.modf
// func modf(value c.Double, iptr *c.Double) c.Double

// //go:linkname pow C.pow
// func pow(x, y c.Double) c.Double

// //go:linkname sqrt C.sqrt
// func sqrt(x c.Double) c.Double

// //go:linkname ceil C.ceil
// func ceil(x c.Double) c.Double

// //go:linkname fabs C.fabs
// func fabs(x c.Double) c.Double

// //go:linkname floor C.floor
// func floor(x c.Double) c.Double

// //go:linkname fmod C.fmod
// func fmod(x, y c.Double) c.Double

// //go:linkname erf C.erf
// func erf(x c.Double) c.Double

// //go:linkname erfc C.erfc
// func erfc(x c.Double) c.Double

// //go:linkname gamma C.gamma
// func gamma(x c.Double) c.Double

// //go:linkname hypot C.hypot
// func hypot(x, y c.Double) c.Double

// //go:linkname isnan C.isnan
// func isnan(x c.Double) c.Int

// //go:linkname isinf C.isinf
// func isinf(x c.Double) c.Int

// //go:linkname j0 C.j0
// func j0(x c.Double) c.Double

// //go:linkname j1 C.j1
// func j1(x c.Double) c.Double

// //go:linkname jn C.jn
// func jn(n c.Int, x c.Double) c.Double

// //go:linkname lgamma C.lgamma
// func lgamma(x c.Double) c.Double

// //go:linkname y0 C.y0
// func y0(x c.Double) c.Double

// //go:linkname y1 C.y1
// func y1(x c.Double) c.Double

// //go:linkname yn C.yn
// func yn(n c.Int, x c.Double) c.Double

// //go:linkname acosh C.acosh
// func acosh(x c.Double) c.Double

// //go:linkname asinh C.asinh
// func asinh(x c.Double) c.Double

// //go:linkname atanh C.atanh
// func atanh(x c.Double) c.Double

// //go:linkname cbrt C.cbrt
// func cbrt(x c.Double) c.Double

// //go:linkname expm1 C.expm1
// func expm1(x c.Double) c.Double

// //go:linkname ilogb C.ilogb
// func ilogb(x c.Double) c.Int

// //go:linkname log1p C.log1p
// func log1p(x c.Double) c.Double

// //go:linkname logb C.logb
// func logb(x c.Double) c.Double

// //go:linkname nextafter C.nextafter
// func nextafter(x, y c.Double) c.Double

// //go:linkname remainder C.remainder
// func remainder(x, y c.Double) c.Double

// //go:linkname scalbn C.scalbn
// func scalbn(x c.Double, n c.Int) c.Double

// //go:linkname scalbln C.scalbln
// func scalbln(x c.Double, n c.Long) c.Double

// //go:linkname nearbyint C.nearbyint
// func nearbyint(x c.Double) c.Double

// //go:linkname rint C.rint
// func rint(x c.Double) c.Double

// //go:linkname lrint C.lrint
// func lrint(x c.Double) c.Long

// //go:linkname llrint C.llrint
// func llrint(x c.Double) c.LongLong

// //go:linkname round C.round
// func round(x c.Double) c.Double

// //go:linkname lround C.lround
// func lround(x c.Double) c.Long

// //go:linkname llround C.llround
// func llround(x c.Double) c.LongLong

// //go:linkname trunc C.trunc
// func trunc(x c.Double) c.Double

// //go:linkname fdim C.fdim
// func fdim(x, y c.Double) c.Double

// //go:linkname fmax C.fmax
// func fmax(x, y c.Double) c.Double

// //go:linkname fmin C.fmin
// func fmin(x, y c.Double) c.Double

// //go:linkname fma C.fma
// func fma(x, y, z c.Double) c.Double

// // 浮点版本
// //
// //go:linkname acosf C.acosf
// func acosf(x c.Float) c.Float

// //go:linkname asinf C.asinf
// func asinf(x c.Float) c.Float

// //go:linkname atanf C.atanf
// func atanf(x c.Float) c.Float

// //go:linkname atan2f C.atan2f
// func atan2f(y, x c.Float) c.Float

// //go:linkname cosf C.cosf
// func cosf(x c.Float) c.Float

// //go:linkname sinf C.sinf
// func sinf(x c.Float) c.Float

// //go:linkname tanf C.tanf
// func tanf(x c.Float) c.Float

// //go:linkname coshf C.coshf
// func coshf(x c.Float) c.Float

// //go:linkname sinhf C.sinhf
// func sinhf(x c.Float) c.Float

// //go:linkname tanhf C.tanhf
// func tanhf(x c.Float) c.Float

// //go:linkname expf C.expf
// func expf(x c.Float) c.Float

// //go:linkname frexpf C.frexpf
// func frexpf(value c.Float, exp *c.Int) c.Float

// //go:linkname ldexpf C.ldexpf
// func ldexpf(x c.Float, exp c.Int) c.Float

// //go:linkname logf C.log极
// func logf(x c.Float) c.Float

// //go:linkname log10f C.log10f
// func log10f(x c.Float) c.Float

// //go:linkname modff C.modff
// func modff(value c.Float, iptr *c.Float) c.Float

// //go:linkname powf C.powf
// func powf(x, y c.Float) c.Float

// //go:linkname sqrtf C.sqrtf
// func sqrtf(x c.Float) c.Float

// //go:linkname ceilf C.ceilf
// func ceilf(x c.Float) c.Float

// //go:linkname fabsf C.fabsf
// func fabsf(x c.Float) c.Float

// //go:linkname floorf C.floorf
// func floorf(x c.Float) c.Float

// //go:linkname fmodf C.fmodf
// func fmodf(x, y c.Float) c.Float

// //go:linkname erff C.erff
// func erff(x c.Float) c.Float

// //go:linkname erfcf C.erfcf
// func erfcf(x c.Float) c.Float

// //go:linkname gammaf C.gammaf
// func gammaf(x c.Float) c.Float

// //go:linkname hypotf C.hypotf
// func hypotf(x, y c.Float) c.Float

// //go:linkname isnanf C.isnanf
// func isnanf(x c.Float) c.Int

// //go:linkname isinff C.isinff
// func isinff(x c.Float) c.Int

// //go:linkname j0f C.j0f
// func j0f(x c.Float) c.Float

// //go:linkname j1f C.j1f
// func j1f(x c.Float) c.Float

// //go:linkname jnf C.jnf
// func jnf(n c.Int, x c.Float) c.Float

// //go:linkname lgammaf C.lgammaf
// func lgammaf(x c.Float) c.Float

// //go:linkname y0f C.y0f
// func y0f(x c.Float) c.Float

// //go:linkname y1f C.y1f
// func y1f(x c.Float) c.Float

// //go:linkname ynf C.ynf
// func ynf(n c.Int, x c.Float) c.Float

// //go:linkname acoshf C.acoshf
// func acoshf(x c.Float) c.Float

// //go:linkname asinhf C.asinhf
// func asinhf(x c.Float) c.Float

// //go:linkname atanhf C.atanhf
// func atanhf(x c.Float) c.Float

// //go:linkname cbrtf C.cbrtf
// func cbrtf(x c.Float) c.Float

// //go:linkname expm1f C.expm1f
// func expm1f(x c.Float) c.Float

// //go:linkname ilogbf C.ilogbf
// func ilogbf(x c.Float) c.Int

// //go:linkname log1pf C.log1pf
// func log1pf(x c.Float) c.Float

// //go:linkname logbf C.logbf
// func logbf(x c.Float) c.Float

// //go:linkname nextafterf C.nextafterf
// func nextafterf(x, y c.Float) c.Float

// //go:linkname remainderf C.remainderf
// func remainderf(x, y c.Float) c.Float

// //go:linkname scalbnf C.scalbnf
// func scalbnf(x c.Float, n c.Int) c.Float

// //go:linkname scalblnf C.scalblnf
// func scalblnf(x c.Float, n c.Long) c.Float

// //go:linkname nearbyintf C.nearbyintf
// func nearbyintf(x c.Float) c.Float

// //go:linkname rintf C.rintf
// func rintf(x c.Float) c.Float

// //go:linkname lrintf C.lrintf
// func lrintf(x c.Float) c.Long

// //go:linkname llrintf C.llrintf
// func llrintf(x c.Float) c.LongLong

// //go:linkname roundf C.roundf
// func roundf(x c.Float) c.Float

// //go:linkname lroundf C.lroundf
// func lroundf(x c.Float) c.Long

// //go:linkname llroundf C.llroundf
// func llroundf(x c.Float) c.LongLong

// //go:linkname truncf C.truncf
// func truncf(x c.Float) c.Float

// //go:linkname fdimf C.fdimf
// func fdimf(x, y c.Float) c.Float

// //go:linkname fmaxf C.fmaxf
// func fmaxf(x, y c.Float) c.Float

// //go:linkname fminf C.fminf
// func fminf(x, y c.Float) c.Float

// //go:linkname fmaf C.fmaf
// func fmaf(x, y, z c.Float) c.Float

//
// //go:linkname time C.time
// func time(t *c.ti) C.time_t

// //go:linkname localtime C.localtime
// func localtime(t *C.time_t) *C.tm

// //go:linkname gmtime C.gmtime
// func gmtime(t *C.time_t) *C.tm

// //go:linkname mktime C.mktime
// func mktime(t *C.tm) C.time_t

// //go:linkname strftime C.strftime
// func strftime(s *c.Char, max c.SizeT, format *c.Char, tm *C.tm) c.SizeT

// //go:linkname clock C.clock
// func clock() C.clock_t

// //go:linkname difftime C.difftime
// func difftime(time1, time0 C.time_t) c.Double

// //go:linkname asctime C.asctime
// func asctime(tm *C.tm) *c.Char

// //go:linkname ctime C.ctime
// func ctime(t *C.time_t) *c.Char

//
//go:linkname exit C.exit
func exit(status c.Int)

//go:linkname abort C.abort
func abort()

//go:linkname system C.system
func system(command *c.Char) c.Int

//go:linkname getenv C.getenv
func getenv(name *c.Char) *c.Char

//go:linkname setenv C.setenv
func setenv(name, value *c.Char, overwrite c.Int) c.Int

//go:linkname unsetenv C.unsetenv
func unsetenv(name *c.Char) c.Int

//go:linkname putenv C.putenv
func putenv(string *c.Char) c.Int

//go:linkname clearenv C.clearenv
func clearenv() c.Int

// //go:linkname getpid C.getpid
// func getpid() C.pid_t

// //go:linkname getppid C.getppid
// func getppid() C.pid_t

// //go:linkname fork C.fork
// func fork() C.pid_t

// //go:linkname vfork C.vfork
// func vfork() C.pid_t

//go:linkname execve C.execve
func execve(path *c.Char, argv **c.Char, envp **c.Char) c.Int

//go:linkname execv C.execv
func execv(path *c.Char, argv **c.Char) c.Int

//go:linkname execvp C.execvp
func execvp(file *c.Char, argv **c.Char) c.Int

//go:linkname execl C.execl
func execl(path *c.Char, arg0 *c.Char, __llgo_va_list ...any) c.Int

//go:linkname execlp C.execlp
func execlp(file *c.Char, arg0 *c.Char, __llgo_va_list ...any) c.Int

//go:linkname execle C.execle
func execle(path *c.Char, arg0 *c.Char, __llgo_va_list ...any) c.Int

// //go:linkname wait C.wait
// func wait(status *c.Int) C.piI_t

// //go:linkname waitpid C.waitpid
// func waitpid(pid C.pid_t, status *c.Int, options c.Int) C.piI_t

//
//go:linkname signal C.signal
func signal(sig c.Int, handler unsafe.Pointer) unsafe.Pointer

//go:linkname raise C.raise
func raise(sig c.Int) c.Int

// //go:linkname kill C.kill
// func kill(pid C.pid_t, sig c.Int) c.Int

// //go:linkname alarm C.alarm
// func alarm(seconds C.uint) C.uint

//go:linkname pause C.pause
func pause() c.Int

//
// //go:linkname opendir C.opendir
// func opendir(name *c.Char) *C.DIR

// //go:linkname readdir C.readdir
// func readdir(dirp *C.DIR) *C.dirent

// //go:linkname closedir C.closedir
// func closedir(dirp *C.DIR) c.Int

// //go:linkname rewinddir C.rewinddir
// func rewinddir(dirp *C.DIR)

// //go:linkname seekdir C.seekdir
// func seekdir(dirp *C.DIR, loc c.Long)

// //go:linkname telldir C.telldir
// func telldir(dirp *C.DIR) c.Long

//
//go:linkname open C.open
func open(path *c.Char, oflag c.Int, mode c.Int) c.Int

//go:linkname creat C.creat
func creat(path *c.Char, mode c.Int) c.Int

//go:linkname close C.close
func close(fd c.Int) c.Int

//go:linkname read C.read
func read(fd c.Int, buf unsafe.Pointer, count c.SizeT) c.SsizeT

//go:linkname write C.write
func write(fd c.Int, buf unsafe.Pointer, count c.SizeT) c.SsizeT

// //go:linkname lseek C.lseek
// func lseek(fd c.Int, offset C.off_t, whence c.Int) C.ofI_t

//go:linkname fsync C.fsync
func fsync(fd c.Int) c.Int

//go:linkname fdatasync C.fdatasync
func fdatasync(fd c.Int) c.Int

// //go:linkname ftruncate C.ftruncate
// func ftruncate(fd c.Int, length C.off_t) c.Int

//go:linkname dup C.dup
func dup(fd c.Int) c.Int

//go:linkname dup2 C.dup2
func dup2(oldfd, newfd c.Int) c.Int

//go:linkname pipe C.pipe
func pipe(fildes *[2]c.Int) c.Int

//go:linkname chdir C.chdir
func chdir(path *c.Char) c.Int

//go:linkname fchdir C.fchdir
func fchdir(fd c.Int) c.Int

//go:linkname getcwd C.getcwd
func getcwd(buf *c.Char, size c.SizeT) *c.Char

//go:linkname chmod C.chmod
func chmod(path *c.Char, mode c.Int) c.Int

//go:linkname fchmod C.fchmod
func fchmod(fd c.Int, mode c.Int) c.Int

// //go:linkname chown C.chown
// func chown(path *c.Char, owner C.uid_t, group C.gid_t) c.Int

// //go:linkname fchown C.fchown
// func fchown(fd c.Int, owner C.uid_t, group C.gid_t) c.Int

// //go:linkname lchown C.lchown
// func lchown(path *c.Char, owner C.uid_t, group C.gid_t) c.Int

//go:linkname link C.link
func link(oldpath, newpath *c.Char) c.Int

//go:linkname unlink C.unlink
func unlink(path *c.Char) c.Int

//go:linkname symlink C.symlink
func symlink(oldpath, newpath *c.Char) c.Int

//go:linkname readlink C.readlink
func readlink(path *c.Char, buf *c.Char, bufsiz c.SizeT) c.SsizeT

//go:linkname mkdir C.mkdir
func mkdir(path *c.Char, mode c.Int) c.Int

//go:linkname rmdir C.rmdir
func rmdir(path *c.Char) c.Int

// //go:linkname stat C.stat
// func stat(path *c.Char, buf *C.struct_stat) c.Int

// //go:linkname fstat C.fstat
// func fstat(fd c.Int, buf *C.struct_stat) c.Int

// //go:linkname lstat C.lstat
// func lstat(path *c.Char, buf *C.struct_stat) c.Int

//go:linkname access C.access
func access(path *c.Char, mode c.Int) c.Int

//go:linkname umask C.umask
func umask(mask c.Int) c.Int

func testStringFunctions() {
	println("Testing string functions...")

	// Test strlen
	s := c.AllocaCStr("Hello, World!")
	// FIXME: defer
	// defer c.Free(unsafe.Pointer(s))
	if strlen(s) != 13 {
		println("FAIL: strlen")
	} else {
		println("PASS: strlen")
	}

	// Test strcpy
	dest := make([]byte, 20)
	strcpy((*c.Char)(unsafe.Pointer(&dest[0])), s)
	if c.GoString((*c.Char)(unsafe.Pointer(&dest[0]))) != "Hello, World!" {
		println("FAIL: strcpy")
	} else {
		println("PASS: strcpy")
	}

	// Test strcmp
	s1 := c.AllocaCStr("abc")
	s2 := c.AllocaCStr("abc")
	s3 := c.AllocaCStr("abd")
	// defer c.Free(unsafe.Pointer(s1))
	// defer c.Free(unsafe.Pointer(s2))
	// defer c.Free(unsafe.Pointer(s3))
	if strcmp(s1, s2) != 0 {
		println("FAIL: strcmp (equal)")
	} else if strcmp(s1, s3) >= 0 {
		println("FAIL: strcmp (less than)")
	} else {
		println("PASS: strcmp")
	}
}

func testMemoryFunctions() {
	println("Testing memory functions...")

	// Test malloc/free
	ptr := malloc(100)
	if ptr == nil {
		println("FAIL: malloc")
	} else {
		println("PASS: malloc")
		free(ptr)
	}

	// Test memcpy
	src := []byte("Hello")
	dest := make([]byte, 10)
	memcpy(unsafe.Pointer(&dest[0]), unsafe.Pointer(&src[0]), 5)
	if string(dest[:5]) != "Hello" {
		println("FAIL: memcpy")
	} else {
		println("PASS: memcpy")
	}

	// Test memset
	memset(unsafe.Pointer(&dest[0]), 'A', 5)
	if string(dest[:5]) != "AAAAA" {
		println("FAIL: memset")
	} else {
		println("PASS: memset")
	}
}

// func testMathFunctions() {
// 	println("Testing math functions...")

// 	// Test abs
// 	if c.ab(-5) != 5 || abs(5) != 5 {
// 		println("FAIL: abs")
// 	} else {
// 		println("PASS: abs")
// 	}

// 	// Test atoi
// 	s := c.AllocaCStr("12345")
// 	defer c.Free(unsafe.Pointer(s))
// 	if c.Atoi(s) != 12345 {
// 		println("FAIL: atoi")
// 	} else {
// 		println("PASS: atoi")
// 	}

// }

func testIOFunctions() {
	println("Testing I/O functions...")

	// Test fopen/fclose
	filename := c.AllocaCStr("test.txt")
	// defer c.Free(unsafe.Pointer(filename))
	mode := c.AllocaCStr("w")
	// defer c.Free(unsafe.Pointer(mode))
	file := fopen(filename, mode)
	if file == nil {
		println("FAIL: fopen")
	} else {
		println("PASS: fopen")
		if fclose(file) != 0 {
			println("FAIL: fclose")
		} else {
			println("PASS: fclose")
		}
	}

	// Test printf
	format := c.AllocaCStr("Test: %d\n")
	// defer c.Free(unsafe.Pointer(format))
	if printf(format, 42) < 0 {
		println("FAIL: printf")
	} else {
		println("PASS: printf")
	}
}

// func testTimeFunctions() {
// 	println("Testing time functions...")

// 	// Test time
// 	t := time(nil)
// 	if t == 0 {
// 		println("FAIL: time")
// 	} else {
// 		println("PASS: time")
// 	}

// 	// Test localtime
// 	tm := localtime(&t)
// 	if tm == nil {
// 		println("FAIL: localtime")
// 	} else {
// 		println("PASS: localtime")
// 	}
// }

// func testProcessFunctions() {
// 	println("Testing process functions...")

// 	// Test getpid
// 	pid := getpid()
// 	if pid <= 0 {
// 		println("FAIL: getpid")
// 	} else {
// 		println("PASS: getpid")
// 	}

// 	// Test getenv
// 	env := c.AllocaCStr("PATH")
// 	defer c.Free(unsafe.Pointer(env))
// 	path := getenv(env)
// 	if path == nil {
// 		println("FAIL: getenv")
// 	} else {
// 		println("PASS: getenv")
// 	}
// }

// func testDirectoryFunctions() {
// 	println("Testing directory functions...")

// 	// Test opendir/closedir
// 	dirname := c.AllocaCStr(".")
// 	defer c.Free(unsafe.Pointer(dirname))
// 	dir := opendir(dirname)
// 	if dir == nil {
// 		println("FAIL: opendir")
// 	} else {
// 		println("PASS: opendir")
// 		if closedir(dir) != 0 {
// 			println("FAIL: closedir")
// 		} else {
// 			println("PASS: closedir")
// 		}
// 	}
// }

// func testFileFunctions() {
// 	println("Testing file functions...")

// 	// Test open/close
// 	filename := c.AllocaCStr("testfile.txt")
// 	defer c.Free(unsafe.Pointer(filename))
// 	fd := open(filename, C.O_CREAT|C.O_WRONLY, 0644)
// 	if fd < 0 {
// 		println("FAIL: open")
// 	} else {
// 		println("PASS: open")
// 		if close(fd) != 0 {
// 			println("FAIL: close")
// 		} else {
// 			println("PASS: close")
// 		}
// 	}
// }

// func testSignalFunctions() {
// 	println("Testing signal functions...")

// 	// Test signal
// 	handler := signal(c.Int(syscall.SIGINT), unsafe.Pointer(syscall.SIG_IGN))
// 	if handler == nil {
// 		println("FAIL: signal")
// 	} else {
// 		println("PASS: signal")
// 	}

// 	// Test raise
// 	if raise(C.SIGINT) != 0 {
// 		println("FAIL: raise")
// 	} else {
// 		println("PASS: raise")
// 	}
// }

func main() {
	watchdog.Disable()
	println("Starting libc tests...")
	testStringFunctions()
	testMemoryFunctions()
	// testMathFunctions()
	testIOFunctions()
	// testTimeFunctions()
	// testProcessFunctions()
	// testDirectoryFunctions()
	// testFileFunctions()
	// testSignalFunctions()
	println("All tests completed!")
}
