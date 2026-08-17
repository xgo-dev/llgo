package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	FS_UNKNOWN   FsType = -1
	FS_CUSTOM    FsType = 0
	FS_OPEN      FsType = 1
	FS_CLOSE     FsType = 2
	FS_READ      FsType = 3
	FS_WRITE     FsType = 4
	FS_SENDFILE  FsType = 5
	FS_STAT      FsType = 6
	FS_LSTAT     FsType = 7
	FS_FSTAT     FsType = 8
	FS_FTRUNCATE FsType = 9
	FS_UTIME     FsType = 10
	FS_FUTIME    FsType = 11
	FS_ACCESS    FsType = 12
	FS_CHMOD     FsType = 13
	FS_FCHMOD    FsType = 14
	FS_FSYNC     FsType = 15
	FS_FDATASYNC FsType = 16
	FS_UNLINK    FsType = 17
	FS_RMDIR     FsType = 18
	FS_MKDIR     FsType = 19
	FS_MKDTEMP   FsType = 20
	FS_RENAME    FsType = 21
	FS_SCANDIR   FsType = 22
	FS_LINK      FsType = 23
	FS_SYMLINK   FsType = 24
	FS_READLINK  FsType = 25
	FS_CHOWN     FsType = 26
	FS_FCHOWN    FsType = 27
	FS_REALPATH  FsType = 28
	FS_COPYFILE  FsType = 29
	FS_LCHOWN    FsType = 30
	FS_OPENDIR   FsType = 31
	FS_READDIR   FsType = 32
	FS_CLOSEDIR  FsType = 33
	FS_STATFS    FsType = 34
	FS_MKSTEMP   FsType = 35
	FS_LUTIME    FsType = 36
)

const (
	DirentUnknown DirentType = iota
	DirentFile
	DirentDir
	DirentLink
	DirentFifo
	DirentSocket
	DirentChar
	DirentBlock
)

type FsType c.Int

type DirentType c.Int

type File c.Int

// ----------------------------------------------

/* Handle types. */

type Fs struct {
	Unused [440]byte
}

type FsEvent struct {
	Unused [0]byte
}

type FsPoll struct {
	Unused [0]byte
}

type Dirent struct {
	Name *c.Char
	Type DirentType
}

type Stat struct {
	Unused [0]byte
}

// ----------------------------------------------

/* Function type */

// llgo:type C
type FsCb func(req *Fs)

// llgo:type C
type FsEventCb func(handle *FsEvent, filename *c.Char, events c.Int, status c.Int)

// llgo:type C
type FsPollCb func(handle *FsPoll, status c.Int, events c.Int)

// ----------------------------------------------

/* Fs related function and method */

// llgo:link (*Fs).GetType C.uv_fs_get_type
func (req *Fs) GetType() FsType {
	return 0
}

// llgo:link (*Fs).GetPath C.uv_fs_get_path
func (req *Fs) GetPath() *c.Char {
	return nil
}

// llgo:link (*Fs).GetResult C.uv_fs_get_result
func (req *Fs) GetResult() c.Int {
	return 0
}

// llgo:link (*Fs).GetPtr C.uv_fs_get_ptr
func (req *Fs) GetPtr() c.Pointer {
	return nil
}

// llgo:link (*Fs).GetSystemError C.uv_fs_get_system_error
func (req *Fs) GetSystemError() c.Int {
	return 0
}

// llgo:link (*Fs).GetStatBuf C.uv_fs_get_statbuf
func (req *Fs) GetStatBuf() *Stat {
	return nil
}

// llgo:link (*Fs).ReqCleanup C.uv_fs_req_cleanup
func (req *Fs) ReqCleanup() {
	// No return value needed for this method
}

//go:linkname FsOpen C.uv_fs_open
func FsOpen(loop *Loop, req *Fs, path *c.Char, flags c.Int, mode c.Int, cb FsCb) c.Int

//go:linkname FsClose C.uv_fs_close
func FsClose(loop *Loop, req *Fs, file File, cb FsCb) c.Int

//go:linkname FsRead C.uv_fs_read
func FsRead(loop *Loop, req *Fs, file File, bufs *Buf, nbufs c.Uint, offset c.LongLong, cb FsCb) c.Int

//go:linkname FsWrite C.uv_fs_write
func FsWrite(loop *Loop, req *Fs, file File, bufs *Buf, nbufs c.Uint, offset c.LongLong, cb FsCb) c.Int

//go:linkname FsUnlink C.uv_fs_unlink
func FsUnlink(loop *Loop, req *Fs, path *c.Char, cb FsCb) c.Int

//go:linkname FsMkdir C.uv_fs_mkdir
func FsMkdir(loop *Loop, req *Fs, path *c.Char, mode c.Int, cb FsCb) c.Int

//go:linkname FsMkdtemp C.uv_fs_mkdtemp
func FsMkdtemp(loop *Loop, req *Fs, tpl *c.Char, cb FsCb) c.Int

//go:linkname FsMkStemp C.uv_fs_mkstemp
func FsMkStemp(loop *Loop, req *Fs, tpl *c.Char, cb FsCb) c.Int

//go:linkname FsRmdir C.uv_fs_rmdir
func FsRmdir(loop *Loop, req *Fs, path *c.Char, cb FsCb) c.Int

//go:linkname FsStat C.uv_fs_stat
func FsStat(loop *Loop, req *Fs, path *c.Char, cb FsCb) c.Int

//go:linkname FsFstat C.uv_fs_fstat
func FsFstat(loop *Loop, req *Fs, file File, cb FsCb) c.Int

//go:linkname FsRename C.uv_fs_rename
func FsRename(loop *Loop, req *Fs, path *c.Char, newPath *c.Char, cb FsCb) c.Int

//go:linkname FsFsync C.uv_fs_fsync
func FsFsync(loop *Loop, req *Fs, file File, cb FsCb) c.Int

//go:linkname FsFdatasync C.uv_fs_fdatasync
func FsFdatasync(loop *Loop, req *Fs, file File, cb FsCb) c.Int

//go:linkname FsFtruncate C.uv_fs_ftruncate
func FsFtruncate(loop *Loop, req *Fs, file File, offset c.LongLong, cb FsCb) c.Int

//go:linkname FsSendfile C.uv_fs_sendfile
func FsSendfile(loop *Loop, req *Fs, outFd c.Int, inFd c.Int, inOffset c.LongLong, length c.Int, cb FsCb) c.Int

//go:linkname FsAccess C.uv_fs_access
func FsAccess(loop *Loop, req *Fs, path *c.Char, flags c.Int, cb FsCb) c.Int

//go:linkname FsChmod C.uv_fs_chmod
func FsChmod(loop *Loop, req *Fs, path *c.Char, mode c.Int, cb FsCb) c.Int

//go:linkname FsFchmod C.uv_fs_fchmod
func FsFchmod(loop *Loop, req *Fs, file File, mode c.Int, cb FsCb) c.Int

//go:linkname FsUtime C.uv_fs_utime
func FsUtime(loop *Loop, req *Fs, path *c.Char, atime c.Int, mtime c.Int, cb FsCb) c.Int

//go:linkname FsFutime C.uv_fs_futime
func FsFutime(loop *Loop, req *Fs, file File, atime c.Int, mtime c.Int, cb FsCb) c.Int

//go:linkname FsLutime C.uv_fs_lutime
func FsLutime(loop *Loop, req *Fs, path *c.Char, atime c.Int, mtime c.Int, cb FsCb) c.Int

//go:linkname FsLink C.uv_fs_link
func FsLink(loop *Loop, req *Fs, path *c.Char, newPath *c.Char, cb FsCb) c.Int

//go:linkname FsSymlink C.uv_fs_symlink
func FsSymlink(loop *Loop, req *Fs, path *c.Char, newPath *c.Char, flags c.Int, cb FsCb) c.Int

//go:linkname FsReadlink C.uv_fs_read
func FsReadlink(loop *Loop, req *Fs, path *c.Char, cb FsCb) c.Int

//go:linkname FsRealpath C.uv_fs_realpath
func FsRealpath(loop *Loop, req *Fs, path *c.Char, cb FsCb) c.Int

//go:linkname FsCopyfile C.uv_fs_copyfile
func FsCopyfile(loop *Loop, req *Fs, path *c.Char, newPath *c.Char, flags c.Int, cb FsCb) c.Int

//go:linkname FsScandir C.uv_fs_scandir
func FsScandir(loop *Loop, req *Fs, path *c.Char, flags c.Int, cb FsCb) c.Int

//go:linkname FsScandirNext C.uv_fs_scandir_next
func FsScandirNext(req *Fs, ent *Dirent) c.Int

//go:linkname FsOpenDir C.uv_fs_opendir
func FsOpenDir(loop *Loop, req *Fs, path *c.Char, cb FsCb) c.Int

//go:linkname FsReaddir C.uv_fs_readdir
func FsReaddir(loop *Loop, req *Fs, dir c.Int, cb FsCb) c.Int

//go:linkname FsCloseDir C.uv_fs_closedir
func FsCloseDir(loop *Loop, req *Fs) c.Int

//go:linkname FsStatfs C.uv_fs_statfs
func FsStatfs(loop *Loop, req *Fs, path *c.Char, cb FsCb) c.Int

//go:linkname FsChown C.uv_fs_chown
func FsChown(loop *Loop, req *Fs, path *c.Char, uid c.Int, gid c.Int, cb FsCb) c.Int

//go:linkname FsFchown C.uv_fs_fchown
func FsFchown(loop *Loop, req *Fs, file File, uid c.Int, gid c.Int, cb FsCb) c.Int

//go:linkname FsLchown C.uv_fs_lchown
func FsLchown(loop *Loop, req *Fs, path *c.Char, uid c.Int, gid c.Int, cb FsCb) c.Int

//go:linkname FsLstat C.uv_fs_lstat
func FsLstat(loop *Loop, req *Fs, path *c.Char, cb FsCb) c.Int

// ----------------------------------------------

/* FsEvent related function and method */

//go:linkname FsEventInit C.uv_fs_event_init
func FsEventInit(loop *Loop, handle *FsEvent) c.Int

// llgo:link (*FsEvent).Start C.uv_fs_event_start
func (handle *FsEvent) Start(cb FsEventCb, path *c.Char, flags c.Int) c.Int {
	return 0
}

// llgo:link (*FsEvent).Stop C.uv_fs_event_stop
func (handle *FsEvent) Stop() c.Int {
	return 0
}

// llgo:link (*FsEvent).Close C.uv_fs_event_close
func (handle *FsEvent) Close() c.Int {
	return 0
}

// llgo:link (*FsEvent).Getpath C.uv_fs_event_getpath
func (handle *FsEvent) Getpath() *c.Char {
	return nil
}

// ----------------------------------------------

/* FsPoll related function and method */

//go:linkname FsPollInit C.uv_fs_poll_init
func FsPollInit(loop *Loop, handle *FsPoll) c.Int

// llgo:link (*FsPoll).Start C.uv_fs_poll_start
func (handle *FsPoll) Start(cb FsPollCb, path *c.Char, interval uint) c.Int {
	return 0
}

// llgo:link (*FsPoll).Stop C.uv_fs_poll_stop
func (handle *FsPoll) Stop() c.Int {
	return 0
}

// llgo:link (*FsPoll).Close C.uv_fs_poll_close
func (handle *FsPoll) Close() c.Int {
	return 0
}

// llgo:link (*FsPoll).GetPath C.uv_fs_poll_getpath
func (handle *FsPoll) GetPath() *c.Char {
	return nil
}
