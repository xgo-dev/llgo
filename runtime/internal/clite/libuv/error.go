package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/net"
	"github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
)

const (
	E2BIG           = Errno(syscall.E2BIG)
	EACCES          = Errno(syscall.EACCES)
	EADDRINUSE      = Errno(syscall.EADDRINUSE)
	EADDRNOTAVAIL   = Errno(syscall.EADDRNOTAVAIL)
	EAFNOSUPPORT    = Errno(syscall.EAFNOSUPPORT)
	EAGAIN          = Errno(syscall.EAGAIN)
	EALREADY        = Errno(syscall.EALREADY)
	EBADF           = Errno(syscall.EBADF)
	EBUSY           = Errno(syscall.EBUSY)
	ECANCELED       = Errno(syscall.ECANCELED)
	ECONNABORTED    = Errno(syscall.ECONNABORTED)
	ECONNREFUSED    = Errno(syscall.ECONNREFUSED)
	ECONNRESET      = Errno(syscall.ECONNRESET)
	EDESTADDRREQ    = Errno(syscall.EDESTADDRREQ)
	EEXIST          = Errno(syscall.EEXIST)
	EFAULT          = Errno(syscall.EFAULT)
	EFBIG           = Errno(syscall.EFBIG)
	EHOSTUNREACH    = Errno(syscall.EHOSTUNREACH)
	EINTR           = Errno(syscall.EINTR)
	EINVAL          = Errno(syscall.EINVAL)
	EIO             = Errno(syscall.EIO)
	EISCONN         = Errno(syscall.EISCONN)
	EISDIR          = Errno(syscall.EISDIR)
	ELOOP           = Errno(syscall.ELOOP)
	EMFILE          = Errno(syscall.EMFILE)
	EMSGSIZE        = Errno(syscall.EMSGSIZE)
	ENAMETOOLONG    = Errno(syscall.ENAMETOOLONG)
	ENETDOWN        = Errno(syscall.ENETDOWN)
	ENETUNREACH     = Errno(syscall.ENETUNREACH)
	ENFILE          = Errno(syscall.ENFILE)
	ENOBUFS         = Errno(syscall.ENOBUFS)
	ENODEV          = Errno(syscall.ENODEV)
	ENOENT          = Errno(syscall.ENOENT)
	ENOMEM          = Errno(syscall.ENOMEM)
	ENOPROTOOPT     = Errno(syscall.ENOPROTOOPT)
	ENOSPC          = Errno(syscall.ENOSPC)
	ENOSYS          = Errno(syscall.ENOSYS)
	ENOTCONN        = Errno(syscall.ENOTCONN)
	ENOTDIR         = Errno(syscall.ENOTDIR)
	ENOTEMPTY       = Errno(syscall.ENOTEMPTY)
	ENOTSOCK        = Errno(syscall.ENOTSOCK)
	ENOTSUP         = Errno(syscall.ENOTSUP)
	EOVERFLOW       = Errno(syscall.EOVERFLOW)
	EPERM           = Errno(syscall.EPERM)
	EPIPE           = Errno(syscall.EPIPE)
	EPROTO          = Errno(syscall.EPROTO)
	EPROTONOSUPPORT = Errno(syscall.EPROTONOSUPPORT)
	EPROTOTYPE      = Errno(syscall.EPROTOTYPE)
	ERANGE          = Errno(syscall.ERANGE)
	EROFS           = Errno(syscall.EROFS)
	ESHUTDOWN       = Errno(syscall.ESHUTDOWN)
	ESPIPE          = Errno(syscall.ESPIPE)
	ESRCH           = Errno(syscall.ESRCH)
	ETIMEDOUT       = Errno(syscall.ETIMEDOUT)
	ETXTBSY         = Errno(syscall.ETXTBSY)
	EXDEV           = Errno(syscall.EXDEV)
	ENXIO           = Errno(syscall.ENXIO)
	EMLINK          = Errno(syscall.EMLINK)
	EHOSTDOWN       = Errno(syscall.EHOSTDOWN)
	ENOTTY          = Errno(syscall.ENOTTY)
	//EFTYPE        = Errno(syscall.EFTYPE)
	EILSEQ          = Errno(syscall.EILSEQ)
	ESOCKTNOSUPPORT = Errno(syscall.ESOCKTNOSUPPORT)
)

const (
	EAI_ADDRFAMILY = Errno(net.EAI_ADDRFAMILY)
	EAI_AGAIN      = Errno(net.EAI_AGAIN)
	EAI_BADFLAGS   = Errno(net.EAI_BADFLAGS)
	EAI_BADHINTS   = Errno(net.EAI_BADHINTS)
	EAI_FAIL       = Errno(net.EAI_FAIL)
	EAI_FAMILY     = Errno(net.EAI_FAMILY)
	EAI_MEMORY     = Errno(net.EAI_MEMORY)
	EAI_NODATA     = Errno(net.EAI_NODATA)
	EAI_NONAME     = Errno(net.EAI_NONAME)
	EAI_OVERFLOW   = Errno(net.EAI_OVERFLOW)
	EAI_PROTOCOL   = Errno(net.EAI_PROTOCOL)
	EAI_SERVICE    = Errno(net.EAI_SERVICE)
	EAI_SOCKTYPE   = Errno(net.EAI_SOCKTYPE)
)

const (
	EAI_CANCELED Errno = -3003
	ECHARSET     Errno = -4080
	ENONET       Errno = -4056
	UNKNOWN      Errno = -4094
	EOF          Errno = -4095
	EREMOTEIO    Errno = -4030
	ERRNO_MAX    Errno = EOF - 1
)

type Errno c.Int

//go:linkname TranslateSysError C.uv_translate_sys_error
func TranslateSysError(sysErrno c.Int) Errno

//go:linkname Strerror C.uv_strerror
func Strerror(err Errno) *c.Char

//go:linkname StrerrorR C.uv_strerror_r
func StrerrorR(err Errno, buf *c.Char, bufLen uintptr) *c.Char

//go:linkname ErrName C.uv_err_name
func ErrName(err Errno) *c.Char

//go:linkname ErrNameR C.uv_err_name_r
func ErrNameR(err Errno, buf *c.Char, bufLen uintptr) *c.Char
