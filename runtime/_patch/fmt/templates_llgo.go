//go:build llgo

package fmt

//llgointernal:template:bind Printf gofmt format=format args=a result=stdout literal=llgoPrintfLiteral scalar=llgoPrintfScalars
//llgointernal:template:bind Sprintf gofmt format=format args=a result=string literal=llgoSprintfLiteral scalar=llgoSprintfScalars composite=true dynamic=llgoSprintfBuiltins
//llgointernal:template:bind Appendf gofmt format=format args=a result=append
//llgointernal:template:bind Fprintf gofmt format=format args=a result=file
//llgointernal:template:bind Println gofmt args=a scalar=llgoPrintfScalars
//llgointernal:template:bind Sprintln gofmt args=a scalar=llgoSprintfScalars

//llgointernal:template:source gofmt print.go
