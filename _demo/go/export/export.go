package main

import (
	"fmt"
	"runtime"
	"unsafe"

	C "github.com/xgo-dev/llgo/_demo/go/export/c"
)

// assert helper function for testing
func assert[T comparable](got, expected T, message string) {
	if got != expected {
		println("ASSERTION FAILED:", message)
		println("  Expected:", expected)
		println("  Got:     ", got)
		panic("assertion failed: " + message)
	}
	println("✓", message)
}

// Small struct
type SmallStruct struct {
	ID   int8 `json:"id"`
	Flag bool `json:"flag"`
}

// Large struct
type LargeStruct struct {
	ID       int64          `json:"id"`
	Name     string         `json:"name"`
	Values   [10]float64    `json:"values"`
	Metadata map[string]int `json:"metadata"`
	Children []SmallStruct  `json:"children"`
	Extra1   int32          `json:"extra1"`
	Extra2   uint64         `json:"extra2"`
	Extra3   float32        `json:"extra3"`
	Extra4   bool           `json:"extra4"`
	Extra5   uintptr        `json:"extra5"`
}

// Self-referential struct
type Node struct {
	Data int   `json:"data"`
	Next *Node `json:"next"`
}

// Named types
type MyInt int
type MyString string

// FuncInfoResult exposes the information resolved from a Callers PC so the C
// consumer can verify that funcinfo remains usable after loading a c-shared
// library (and from the equivalent c-archive output).
type FuncInfoResult struct {
	CallersCount  int32
	FramePC       uintptr
	FrameFunction string
	FrameFile     string
	FrameLine     int32
	FuncName      string
	FuncEntry     uintptr
	FuncFile      string
	FuncLine      int32
}

// Function types for callbacks
//
//llgo:type C
type IntCallback func(int) int

//llgo:type C
type StringCallback func(string) string

//llgo:type C
type VoidCallback func()

// Complex struct with mixed arrays and slices
type ComplexData struct {
	Matrix   [3][4]int32 `json:"matrix"`    // 2D array
	Slices   [][]string  `json:"slices"`    // slice of slices - commented out
	IntArray [5]int      `json:"int_array"` // 1D array
	DataList []float64   `json:"data_list"` // slice - commented out
}

//export HelloWorld
func HelloWorld() {
	println("Hello, World!")
}

//go:noinline
func captureCFuncInfo() FuncInfoResult {
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:])
	result := FuncInfoResult{CallersCount: int32(n)}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "main.captureCFuncInfo" {
			result.FramePC = frame.PC
			result.FrameFunction = frame.Function
			result.FrameFile = frame.File
			result.FrameLine = int32(frame.Line)

			// Callers PCs are return addresses. Apply Go's pc-1 convention
			// before asking FuncForPC for the containing function.
			if frame.PC > 0 {
				if fn := runtime.FuncForPC(frame.PC - 1); fn != nil {
					result.FuncName = fn.Name()
					result.FuncEntry = fn.Entry()
					result.FuncFile, frame.Line = fn.FileLine(frame.PC - 1)
					result.FuncLine = int32(frame.Line)
				}
			}
			break
		}
		if !more {
			break
		}
	}
	return result
}

//export GetFuncInfo
func GetFuncInfo() FuncInfoResult {
	return captureCFuncInfo()
}

//export RunGoroutine
func RunGoroutine(value int) int {
	ch := make(chan int, 1)
	go func() {
		ch <- value + 1
	}()
	return <-ch
}

//export FormatValue
func FormatValue(value string, number int) string {
	return fmt.Sprintf("%s:%d", value, number)
}

// Functions with small struct parameters and return values

//export CreateSmallStruct
func CreateSmallStruct(id int8, flag bool) SmallStruct {
	return SmallStruct{ID: id, Flag: flag}
}

//export ProcessSmallStruct
func ProcessSmallStruct(s SmallStruct) SmallStruct {
	s.ID += 1
	s.Flag = !s.Flag
	return s
}

//export ProcessSmallStructPtr
func ProcessSmallStructPtr(s *SmallStruct) *SmallStruct {
	if s != nil {
		s.ID *= 2
		s.Flag = !s.Flag
	}
	return s
}

// Functions with large struct parameters and return values

//export CreateLargeStruct
func CreateLargeStruct(id int64, name string) LargeStruct {
	return LargeStruct{
		ID:       id,
		Name:     name,
		Values:   [10]float64{1.1, 2.2, 3.3, 4.4, 5.5, 6.6, 7.7, 8.8, 9.9, 10.0},
		Metadata: map[string]int{"count": 42, "size": 100},
		Children: []SmallStruct{{ID: 1, Flag: true}, {ID: 2, Flag: false}},
		Extra1:   12345,
		Extra2:   67890,
		Extra3:   3.14,
		Extra4:   true,
		Extra5:   0x1000,
	}
}

//export ProcessLargeStruct
func ProcessLargeStruct(ls LargeStruct) int64 {
	total := ls.ID + int64(len(ls.Name))
	for _, v := range ls.Values {
		total += int64(v)
	}
	total += int64(len(ls.Children))
	total += int64(ls.Extra1) + int64(ls.Extra2) + int64(ls.Extra3)
	if ls.Extra4 {
		total += 1000
	}
	total += int64(ls.Extra5)
	return total
}

//export ProcessLargeStructPtr
func ProcessLargeStructPtr(ls *LargeStruct) *LargeStruct {
	if ls != nil {
		ls.ID += 100
		ls.Name = "processed_" + ls.Name
		ls.Extra1 *= 2
		ls.Extra4 = !ls.Extra4
	}
	return ls
}

// Functions with self-referential struct

//export CreateNode
func CreateNode(data int) *Node {
	return &Node{Data: data, Next: nil}
}

//export LinkNodes
func LinkNodes(first, second *Node) int {
	if first != nil && second != nil {
		first.Next = second
		return first.Data + second.Data // Return sum for verification
	}
	if first != nil {
		return first.Data + 1000 // Return data + offset if only first exists
	}
	return 2000 // Return fixed value if both are nil
}

//export TraverseNodes
func TraverseNodes(head *Node) int {
	count := 0
	current := head
	for current != nil {
		count++
		current = current.Next
		if count > 100 { // Safety check
			break
		}
	}
	return count
}

// Functions covering all basic types

//export ProcessBool
func ProcessBool(b bool) bool {
	return !b
}

//export ProcessInt8
func ProcessInt8(x int8) int8 {
	return x + 1
}

//export ProcessUint8
func ProcessUint8(x uint8) uint8 {
	return x + 1
}

//export ProcessInt16
func ProcessInt16(x int16) int16 {
	return x * 2
}

//export ProcessUint16
func ProcessUint16(x uint16) uint16 {
	return x * 2
}

//export ProcessInt32
func ProcessInt32(x int32) int32 {
	return x * 3
}

//export ProcessUint32
func ProcessUint32(x uint32) uint32 {
	return x * 3
}

//export ProcessInt64
func ProcessInt64(x int64) int64 {
	return x * 4
}

//export ProcessUint64
func ProcessUint64(x uint64) uint64 {
	return x * 4
}

//export ProcessInt
func ProcessInt(x int) int {
	return x * 11
}

//export ProcessUint
func ProcessUint(x uint) uint {
	return x * 21
}

//export ProcessUintptr
func ProcessUintptr(x uintptr) uintptr {
	return x + 300
}

//export ProcessFloat32
func ProcessFloat32(x float32) float32 {
	return x * 1.5
}

//export ProcessFloat64
func ProcessFloat64(x float64) float64 {
	return x * 2.5
}

//export ProcessString
func ProcessString(s string) string {
	return "processed_" + s
}

//export ProcessUnsafePointer
func ProcessUnsafePointer(p unsafe.Pointer) unsafe.Pointer {
	return p
}

// Functions with named types

//export ProcessMyInt
func ProcessMyInt(x MyInt) MyInt {
	return x * 10
}

//export ProcessMyString
func ProcessMyString(s MyString) MyString {
	return MyString("modified_" + string(s))
}

// Functions with arrays, slices, maps, channels

//export ProcessIntArray
func ProcessIntArray(arr [5]int) int {
	total := 0
	for _, v := range arr {
		total += v
	}
	return total
}

//export CreateComplexData
func CreateComplexData() ComplexData {
	return ComplexData{
		Matrix:   [3][4]int32{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}},
		Slices:   [][]string{{"helo"}},
		IntArray: [5]int{10, 20, 30, 40, 50},
		DataList: []float64{1.0},
	}
}

//export ProcessComplexData
func ProcessComplexData(data ComplexData) int32 {
	// Sum all matrix elements
	var sum int32
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			sum += data.Matrix[i][j]
		}
	}
	return sum
}

// Functions with multidimensional arrays as parameters and return values

//export ProcessMatrix2D
func ProcessMatrix2D(matrix [3][4]int32) int32 {
	var sum int32
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			sum += matrix[i][j]
		}
	}
	return sum
}

//export CreateMatrix1D
func CreateMatrix1D() [4]int32 {
	return [4]int32{1, 2, 3, 4}
}

//export CreateMatrix2D
func CreateMatrix2D() [3][4]int32 {
	return [3][4]int32{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}
}

//export ProcessMatrix3D
func ProcessMatrix3D(cube [2][3][4]uint8) uint32 {
	var sum uint32
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 4; k++ {
				sum += uint32(cube[i][j][k])
			}
		}
	}
	return sum
}

//export CreateMatrix3D
func CreateMatrix3D() [2][3][4]uint8 {
	var cube [2][3][4]uint8
	val := uint8(1)
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 4; k++ {
				cube[i][j][k] = val
				val++
			}
		}
	}
	return cube
}

//export ProcessGrid5x4
func ProcessGrid5x4(grid [5][4]float64) float64 {
	var sum float64
	for i := 0; i < 5; i++ {
		for j := 0; j < 4; j++ {
			sum += grid[i][j]
		}
	}
	return sum
}

//export CreateGrid5x4
func CreateGrid5x4() [5][4]float64 {
	var grid [5][4]float64
	val := 1.0
	for i := 0; i < 5; i++ {
		for j := 0; j < 4; j++ {
			grid[i][j] = val
			val += 0.5
		}
	}
	return grid
}

//export ProcessIntSlice
func ProcessIntSlice(slice []int) int {
	total := 0
	for _, v := range slice {
		total += v
	}
	return total
}

//export CreateIntSlice
func CreateIntSlice() []int {
	return []int{2, 4, 6, 8}
}

//export ProcessStringMap
func ProcessStringMap(m map[string]int) int {
	total := 0
	for _, v := range m {
		total += v
	}
	return total
}

//export CreateStringMap
func CreateStringMap() map[string]int {
	return map[string]int{"one": 10, "two": 20, "three": 30}
}

//export ProcessIntChannel
func ProcessIntChannel(ch chan int) int {
	select {
	case val := <-ch:
		return val
	default:
		return -1
	}
}

//export CreateIntChannel
func CreateIntChannel() chan int {
	ch := make(chan int, 1)
	ch <- 321
	return ch
}

// Functions with function callbacks

//export ProcessWithIntCallback
func ProcessWithIntCallback(x int, callback IntCallback) int {
	if callback != nil {
		return callback(x)
	}
	return x
}

//export ProcessWithStringCallback
func ProcessWithStringCallback(s string, callback StringCallback) string {
	if callback != nil {
		return callback(s)
	}
	return s
}

//export ProcessWithVoidCallback
func ProcessWithVoidCallback(callback VoidCallback) int {
	if callback != nil {
		callback()
		return 123 // Return non-zero to indicate callback was called
	}
	return 456 // Return different value if callback is nil
}

//export ProcessThreeUnnamedParams
func ProcessThreeUnnamedParams(a int, s string, b bool) float64 {
	result := float64(a) + float64(len(s))
	if b {
		result *= 1.5
	}
	return result
}

// Functions with interface

//export ProcessInterface
func ProcessInterface(i interface{}) int {
	switch v := i.(type) {
	case int:
		return v + 100
	case string:
		return len(v) * 10
	default:
		return 999 // Non-zero default to avoid false positives
	}
}

//export CreateIntInterface
func CreateIntInterface() interface{} {
	return 23
}

//export CreateStringInterface
func CreateStringInterface() interface{} {
	return "llgo"
}

// Functions with various parameter counts

//export NoParams
func NoParams() int {
	return 42
}

//export OneParam
func OneParam(x int) int {
	return x * 2
}

//export TwoParams
func TwoParams(a int, b string) string {
	return string(rune(a)) + b
}

//export ThreeParams
func ThreeParams(a int32, b float64, c bool) float64 {
	result := float64(a) + b
	if c {
		result *= 2
	}
	return result
}

//export MultipleParams
func MultipleParams(a int8, b uint16, c int32, d uint64, e float32, f float64, g string, h bool) string {
	result := g + "_" + string(rune('A'+a)) + string(rune('0'+b%10)) + string(rune('0'+c%10))
	if h {
		result += "_true"
	}
	return result + "_" + string(rune('0'+int(d%10))) + "_" + string(rune('0'+int(e)%10)) + "_" + string(rune('0'+int(f)%10))
}

//export NoParamNames
func NoParamNames(int8, int16, bool) int32 {
	return 789 // Return non-zero value for testing, params are unnamed by design
}

// Functions returning no value

//export NoReturn
func NoReturn(message string) {
	println("Message:", message)
}

// Functions using XType from c package

//export CreateXType
func CreateXType(id int32, name string, value float64, flag bool) C.XType {
	return C.XType{
		ID:    id,
		Name:  name,
		Value: value,
		Flag:  flag,
	}
}

//export ProcessXType
func ProcessXType(x C.XType) C.XType {
	x.ID += 100
	x.Name = "processed_" + x.Name
	x.Value *= 2.0
	x.Flag = !x.Flag
	return x
}

//export ProcessXTypePtr
func ProcessXTypePtr(x *C.XType) *C.XType {
	if x != nil {
		x.ID *= 2
		x.Name = "ptr_" + x.Name
		x.Value += 10.0
		x.Flag = !x.Flag
	}
	return x
}

func main() {
	println("=== Export Demo ===")

	// Test small struct
	small := CreateSmallStruct(5, true)
	assert(small.ID, int8(5), "CreateSmallStruct ID should be 5")
	assert(small.Flag, true, "CreateSmallStruct Flag should be true")
	println("Small struct:", small.ID, small.Flag)

	processed := ProcessSmallStruct(small)
	assert(processed.ID, int8(6), "ProcessSmallStruct should increment ID to 6")
	assert(processed.Flag, false, "ProcessSmallStruct should flip Flag to false")
	println("Processed small:", processed.ID, processed.Flag)

	// Test large struct
	large := CreateLargeStruct(12345, "test")
	assert(large.ID, int64(12345), "CreateLargeStruct ID should be 12345")
	assert(large.Name, "test", "CreateLargeStruct Name should be 'test'")
	println("Large struct ID:", large.ID, "Name:", large.Name)

	total := ProcessLargeStruct(large)
	// Expected calculation:
	// ID: 12345, Name len: 4, Values: 1+2+3+4+5+6+7+8+9+10=55, Children len: 2
	// Extra1: 12345, Extra2: 67890, Extra3: 3, Extra4: +1000, Extra5: 4096
	expectedTotal := int64(12345 + 4 + 55 + 2 + 12345 + 67890 + 3 + 1000 + 4096)
	assert(total, expectedTotal, "ProcessLargeStruct total should match expected calculation")
	println("Large struct total:", total)

	// Test self-referential struct
	node1 := CreateNode(100)
	node2 := CreateNode(200)
	linkResult := LinkNodes(node1, node2)
	assert(linkResult, 300, "LinkNodes should return sum of node data (100 + 200)")

	count := TraverseNodes(node1)
	assert(count, 2, "TraverseNodes should count 2 linked nodes")
	println("Node count:", count)

	// Test basic types with assertions
	assert(ProcessBool(true), false, "ProcessBool(true) should return false")
	assert(ProcessInt8(10), int8(11), "ProcessInt8(10) should return 11")
	f32Result := ProcessFloat32(3.14)
	// Float comparison with tolerance
	if f32Result < 4.7 || f32Result > 4.72 {
		println("ASSERTION FAILED: ProcessFloat32(3.14) should return ~4.71, got:", f32Result)
		panic("float assertion failed")
	}
	println("✓ ProcessFloat32(3.14) returns ~4.71")

	assert(ProcessString("hello"), "processed_hello", "ProcessString should prepend 'processed_'")

	println("Bool:", ProcessBool(true))
	println("Int8:", ProcessInt8(10))
	println("Float32:", ProcessFloat32(3.14))
	println("String:", ProcessString("hello"))

	// Test named types
	myInt := ProcessMyInt(MyInt(42))
	assert(myInt, MyInt(420), "ProcessMyInt(42) should return 420")
	println("MyInt:", int(myInt))

	myStr := ProcessMyString(MyString("world"))
	assert(myStr, MyString("modified_world"), "ProcessMyString should prepend 'modified_'")
	println("MyString:", string(myStr))

	// Test collections
	arr := [5]int{1, 2, 3, 4, 5}
	arrSum := ProcessIntArray(arr)
	assert(arrSum, 15, "ProcessIntArray([1,2,3,4,5]) should return 15")
	println("Array sum:", arrSum)

	slice := []int{10, 20, 30}
	sliceSum := ProcessIntSlice(slice)
	assert(sliceSum, 60, "ProcessIntSlice([10,20,30]) should return 60")
	println("Slice sum:", sliceSum)

	m := make(map[string]int)
	m["a"] = 100
	m["b"] = 200
	mapSum := ProcessStringMap(m)
	assert(mapSum, 300, "ProcessStringMap({'a':100,'b':200}) should return 300")
	println("Map sum:", mapSum)

	// Test multidimensional arrays
	matrix2d := CreateMatrix2D()
	matrix2dSum := ProcessMatrix2D(matrix2d)
	assert(matrix2dSum, int32(78), "ProcessMatrix2D should return 78 (sum of 1+2+...+12)")
	println("Matrix2D sum:", matrix2dSum)

	matrix3d := CreateMatrix3D()
	matrix3dSum := ProcessMatrix3D(matrix3d)
	assert(matrix3dSum, uint32(300), "ProcessMatrix3D should return 300")
	println("Matrix3D sum:", matrix3dSum)

	grid5x4 := CreateGrid5x4()
	gridSum := ProcessGrid5x4(grid5x4)
	assert(gridSum, 115.0, "ProcessGrid5x4 should return 115.0")
	println("Grid5x4 sum:", gridSum)

	// Test complex data with multidimensional arrays
	complexData := CreateComplexData()
	complexSum := ProcessComplexData(complexData)
	assert(complexSum, int32(78), "ProcessComplexData should return 78")
	println("ComplexData matrix sum:", complexSum)

	// Test various parameter counts
	assert(NoParams(), 42, "NoParams should return 42")
	assert(OneParam(5), 10, "OneParam(5) should return 10")
	assert(TwoParams(65, "_test"), "A_test", "TwoParams should return 'A_test'")
	assert(ThreeParams(10, 2.5, true), 25.0, "ThreeParams should return 25.0")
	assert(NoParamNames(1, 2, false), int32(789), "NoParamNames should return 789")

	println("NoParams:", NoParams())
	println("OneParam:", OneParam(5))
	println("TwoParams:", TwoParams(65, "_test"))
	println("ThreeParams:", ThreeParams(10, 2.5, true))
	println("MultipleParams:", MultipleParams(1, 2, 3, 4, 5.0, 6.0, "result", true))
	println("NoParamNames:", NoParamNames(1, 2, false))

	// Test XType from c package
	xtype := CreateXType(42, "test", 3.14, true)
	println("XType:", xtype.ID, xtype.Name, xtype.Value, xtype.Flag)

	processedX := ProcessXType(xtype)
	println("Processed XType:", processedX.ID, processedX.Name, processedX.Value, processedX.Flag)

	ptrX := ProcessXTypePtr(&xtype)
	if ptrX != nil {
		println("Ptr XType:", ptrX.ID, ptrX.Name, ptrX.Value, ptrX.Flag)
	}

	// Test callback functions
	intResult := ProcessWithIntCallback(10, func(x int) int { return x * 3 })
	println("IntCallback result:", intResult)

	stringResult := ProcessWithStringCallback("hello", func(s string) string { return s + "_callback" })
	println("StringCallback result:", stringResult)

	ProcessWithVoidCallback(func() { println("VoidCallback executed") })

	NoReturn("demo completed")
}
