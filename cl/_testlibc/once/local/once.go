package local

// #include <pthread.h>
import "C"
import _ "unsafe"

const (
	LLGoFiles   = "_wrap/pthd.c"
	LLGoPackage = "link"
)

type Once C.pthread_once_t

//go:linkname OnceInit C.llgoSyncOnceInitVal
var OnceInit Once

// llgo:link (*Once).Do C.pthread_once
func (o *Once) Do(f func()) C.int { return 0 }
