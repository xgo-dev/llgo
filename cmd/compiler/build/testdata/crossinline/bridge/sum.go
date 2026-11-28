package bridge

import "example.com/crossinline/dep"

func Sum(x, y uint64) uint64            { return dep.Sum(x, y) }
func Add(n dep.Number, x uint64) uint64 { return n.Add(x) }
