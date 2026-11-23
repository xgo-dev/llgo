//go:build llgo && linux && baremetal && llgo.baremetal.stdio && !go1.25

package os

//llgo:skip NewFile
func NewFile(fd uintptr, name string) *File {
	return llgoConsoleFile(fd, name)
}
