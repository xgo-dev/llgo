//go:build llgo

package fmt

import "io"

//llgointernal:template:gofmt format=format args=a result=stdout literal=llgoPrintfLiteral scalar=llgoPrintfScalars
func Printf(format string, a ...any) (int, error)

//llgointernal:template:gofmt format=format args=a result=string literal=llgoSprintfLiteral scalar=llgoSprintfScalars composite=true dynamic=llgoSprintfBuiltins
func Sprintf(format string, a ...any) string

//llgointernal:template:gofmt format=format args=a result=append
func Appendf(b []byte, format string, a ...any) []byte

//llgointernal:template:gofmt format=format args=a result=file
func Fprintf(w io.Writer, format string, a ...any) (int, error)

//llgointernal:template:gofmt args=a scalar=llgoPrintfScalars
func Println(a ...any) (int, error)

//llgointernal:template:gofmt args=a scalar=llgoSprintfScalars
func Sprintln(a ...any) string

//llgointernal:template:source gofmt print.go
