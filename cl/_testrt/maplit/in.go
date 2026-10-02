// LITTEST
// Scope: common
package main

import "go/constant"

// Large static map literals lower to a counted loop of mapassign over
// private constant key/value arrays, matching cmd/compile's maplit (>25 entries).
// CHECK: maplit{{.*}}.k" = private unnamed_addr constant [26 x
// CHECK: maplit{{.*}}.v" = private unnamed_addr constant [26 x

// CHECK-LABEL: define {{.*}} @main.big(){{.*}} {
// CHECK: icmp slt
// CHECK: {{MapAssignFastStr|MapAssign}}

func big() map[string]int {
	return map[string]int{
		"k0": 0, "k1": 1, "k2": 2, "k3": 3, "k4": 4,
		"k5": 5, "k6": 6, "k7": 7, "k8": 8, "k9": 9,
		"k10": 10, "k11": 11, "k12": 12, "k13": 13, "k14": 14,
		"k15": 15, "k16": 16, "k17": 17, "k18": 18, "k19": 19,
		"k20": 20, "k21": 21, "k22": 22, "k23": 23, "k24": 24,
		"k25": 25,
	}
}

// Runtime values stay unrolled. Spilling them into alloca [N x T] makes
// LLVM default<Os> SLP scalarize the array (ixgo/pkg/unicode).
// CHECK-LABEL: define {{.*}} @main.live(i64 %0){{.*}} {
// CHECK-NOT: alloca [
// CHECK-NOT: icmp slt
// CHECK: {{MapAssignFastStr|MapAssign}}
func live(n int) map[string]int {
	return map[string]int{
		"k0": n, "k1": n, "k2": n, "k3": n, "k4": n,
		"k5": n, "k6": n, "k7": n, "k8": n, "k9": n,
		"k10": n, "k11": n, "k12": n, "k13": n, "k14": n,
		"k15": n, "k16": n, "k17": n, "k18": n, "k19": n,
		"k20": n, "k21": n, "k22": n, "k23": n, "k24": n,
		"k25": n,
	}
}

type untypedConst struct {
	Typ   string
	Value constant.Value
}

// CHECK-LABEL: define {{.*}} @main.uc(){{.*}} {
// CHECK-NOT: MakeInt64
// CHECK-NOT: NewItab
// CHECK: icmp slt
// CHECK: {{MapAssignFastStr|MapAssign}}

func uc() map[string]untypedConst {
	return map[string]untypedConst{
		"k0":  {"untyped int", constant.MakeInt64(0)},
		"k1":  {"untyped int", constant.MakeInt64(1)},
		"k2":  {"untyped int", constant.MakeInt64(2)},
		"k3":  {"untyped int", constant.MakeInt64(3)},
		"k4":  {"untyped int", constant.MakeInt64(4)},
		"k5":  {"untyped int", constant.MakeInt64(5)},
		"k6":  {"untyped int", constant.MakeInt64(6)},
		"k7":  {"untyped int", constant.MakeInt64(7)},
		"k8":  {"untyped int", constant.MakeInt64(8)},
		"k9":  {"untyped int", constant.MakeInt64(9)},
		"k10": {"untyped int", constant.MakeInt64(10)},
		"k11": {"untyped int", constant.MakeInt64(11)},
		"k12": {"untyped int", constant.MakeInt64(12)},
		"k13": {"untyped int", constant.MakeInt64(13)},
		"k14": {"untyped int", constant.MakeInt64(14)},
		"k15": {"untyped int", constant.MakeInt64(15)},
		"k16": {"untyped int", constant.MakeInt64(16)},
		"k17": {"untyped int", constant.MakeInt64(17)},
		"k18": {"untyped int", constant.MakeInt64(18)},
		"k19": {"untyped int", constant.MakeInt64(19)},
		"k20": {"untyped int", constant.MakeInt64(20)},
		"k21": {"untyped int", constant.MakeInt64(21)},
		"k22": {"untyped int", constant.MakeInt64(22)},
		"k23": {"untyped int", constant.MakeInt64(23)},
		"k24": {"untyped int", constant.MakeInt64(24)},
		"k25": {"untyped bool", constant.MakeBool(true)},
	}
}

func main() {
	m := big()
	if len(m) != 26 || m["k0"] != 0 || m["k25"] != 25 {
		panic(m)
	}
	l := live(7)
	if len(l) != 26 || l["k0"] != 7 || l["k25"] != 7 {
		panic(l)
	}
	u := uc()
	if len(u) != 26 {
		panic(u)
	}
	v0, _ := constant.Int64Val(u["k0"].Value)
	if u["k0"].Typ != "untyped int" || v0 != 0 || constant.BoolVal(u["k25"].Value) != true {
		panic(u)
	}
	println("ok")
}
