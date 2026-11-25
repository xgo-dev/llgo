//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>

int objectiveCLength(void);
static int preambleLength(void) {
    return (int)[@"abc" length];
}
*/
import "C"

func main() {
	println("objc:", C.preambleLength(), C.objectiveCLength())
}
