package main

/*
typedef float F4 __attribute__((vector_size(16)));
typedef F4 Alias;
typedef float A4[4];
typedef struct { char tag; F4 vector; char tail; } Box;
*/
import "C"
import "unsafe"

var Vector C.F4
var Alias C.Alias
var Array C.A4
var Box C.Box

var Layout = [4]uintptr{unsafe.Sizeof(Box), unsafe.Offsetof(Box.vector), unsafe.Alignof(Vector), unsafe.Alignof(Array)}

func main() { Vector[0] = 3; Alias = Vector; Array[0] = 7; Box.vector = Alias }
