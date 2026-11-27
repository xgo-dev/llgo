package main

func main() {
	backing := [4]byte{1, 2, 3, 4}
	short := backing[:1]
	ptr := (*[4]byte)(short)
	value := [4]byte(short)
	println(ptr[0], ptr[3], value[0], value[3])
}
