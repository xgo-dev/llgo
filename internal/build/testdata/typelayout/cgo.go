package main

/*
#include "types.h"
*/
import "C"

type Bool C._Bool
type Complex64 C.AuditComplex64
type Complex128 C.AuditComplex128
type Char C.char
type Uchar C.uchar
type Short C.short
type Ushort C.ushort
type Int C.int
type Uint C.uint
type Long C.long
type Ulong C.ulong
type LongLong C.longlong
type UlongLong C.ulonglong
type Float C.float
type Double C.double
type Pointer *C.char
type SizeT C.AuditSizeT
type SsizeT C.AuditSsizeT
type IntptrT C.AuditIntptrT
type UintptrT C.AuditUintptrT
type WcharT C.AuditWcharT
type Enum C.AuditEnum
type Record C.AuditRecord

// Globals keep the actual cgo -> gossa -> LLGo compilation path observable.
var ValueBool Bool
var ValueComplex64 Complex64
var ValueComplex128 Complex128
var ValueChar Char
var ValueUchar Uchar
var ValueShort Short
var ValueUshort Ushort
var ValueInt Int
var ValueUint Uint
var ValueLong Long
var ValueUlong Ulong
var ValueLongLong LongLong
var ValueUlongLong UlongLong
var ValueFloat Float
var ValueDouble Double
var ValuePointer Pointer
var ValueSizeT SizeT
var ValueSsizeT SsizeT
var ValueIntptrT IntptrT
var ValueUintptrT UintptrT
var ValueWcharT WcharT
var ValueEnum Enum
var ValueRecord Record

func main() {}
