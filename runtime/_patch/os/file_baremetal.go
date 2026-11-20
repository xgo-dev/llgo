//go:build llgo && linux && baremetal && llgo.baremetal.stdio && go1.25

package os

//llgo:skip newFileFromNewFile
func newFileFromNewFile(fd uintptr, name string) *File {
	return llgoConsoleFile(fd, name)
}
