package main

import "github.com/xgo-dev/llgo/_demo/go/ifaceprom-1559/foo"

type Game1 struct {
	*foo.Game
}

type Game2 struct {
}

func (p *Game2) initGame() {
}

func main() {
	var g1 any = &Game1{&foo.Game{}}
	var g2 any = &Game2{}
	v1, ok := g1.(foo.Gamer)
	println("OK", v1, ok)
	if ok {
		v1.Load()
	}
	v2, ok := g2.(foo.Gamer)
	println("FAIL", v2, ok)
}
