//go:build llgo && amd64 && (linux || darwin)

package cgo

/*
#include <stdint.h>
typedef struct {float first,second;} llgo_sysv_float_pair;
typedef struct {int64_t first,second;} llgo_sysv_integer_pair;
typedef struct {double first;int64_t second;} llgo_sysv_mixed_pair;
#define FP_ARGS double a,double b,double c,double d,double e,double f,double g,double h
#define FP_SUM (a+b+c+d+e+f+g+h)
double llgo_test_sysv_float_pair(FP_ARGS,llgo_sysv_float_pair pair,double tail){return FP_SUM+pair.first+pair.second+tail;}
int64_t llgo_test_sysv_integer_pair(int64_t a,int64_t b,int64_t c,int64_t d,int64_t e,llgo_sysv_integer_pair pair,int64_t tail){return a+b+c+d+e+pair.first+pair.second+tail;}
double llgo_test_sysv_mixed_gp(int64_t a,int64_t b,int64_t c,int64_t d,int64_t e,int64_t f,llgo_sysv_mixed_pair pair,double tail){return a+b+c+d+e+f+pair.first+pair.second+tail;}
double llgo_test_sysv_mixed_fp(FP_ARGS,llgo_sysv_mixed_pair pair,int64_t tail){return FP_SUM+pair.first+pair.second+tail;}
static void *llgo_sysv_float_address(void){return (void *)&llgo_test_sysv_float_pair;}
*/
import "C"

import "unsafe"

//llgo:cdecl
type sysVFloatPairEntry func(float64, float64, float64, float64, float64, float64, float64, float64, C.llgo_sysv_float_pair, float64) float64

//go:linkname linkedSysVFloatPair C.llgo_test_sysv_float_pair
func linkedSysVFloatPair(float64, float64, float64, float64, float64, float64, float64, float64, C.llgo_sysv_float_pair, float64) float64

func directSysVFloatPair() float64 {
	return float64(C.llgo_test_sysv_float_pair(1, 2, 3, 4, 5, 6, 7, 8, C.llgo_sysv_float_pair{first: 1, second: 2}, 3))
}
func indirectSysVFloatPair() float64 {
	address := C.llgo_sysv_float_address()
	function := *(*sysVFloatPairEntry)(unsafe.Pointer(&address))
	return function(1, 2, 3, 4, 5, 6, 7, 8, C.llgo_sysv_float_pair{first: 1, second: 2}, 3)
}
func linkedSysVFloatPairCall() float64 {
	return linkedSysVFloatPair(1, 2, 3, 4, 5, 6, 7, 8, C.llgo_sysv_float_pair{first: 1, second: 2}, 3)
}
func partialSysVIntegerPair() int64 {
	return int64(C.llgo_test_sysv_integer_pair(1, 2, 3, 4, 5, C.llgo_sysv_integer_pair{first: 6, second: 7}, 14))
}
func rollbackSysVMixedGP() float64 {
	return float64(C.llgo_test_sysv_mixed_gp(1, 2, 3, 4, 5, 6, C.llgo_sysv_mixed_pair{first: 7, second: 8}, 6))
}
func rollbackSysVMixedFP() float64 {
	return float64(C.llgo_test_sysv_mixed_fp(1, 2, 3, 4, 5, 6, 7, 8, C.llgo_sysv_mixed_pair{first: 1, second: 2}, 3))
}
