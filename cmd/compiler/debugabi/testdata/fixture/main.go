package main

type AliasInt = int

type NamedInt int

type Node struct {
	Value int
	Next  *Node
}

type Aggregate struct {
	Name   string
	Values [3]NamedInt
	Node   *Node
}

type Greeter interface {
	Greet(string) string
}

type Prefix string

func (prefix Prefix) Greet(name string) string {
	return string(prefix) + name
}

type Counter struct {
	Base int
}

func (counter *Counter) Add(value int) int {
	return counter.Base + value
}

func addOne(value int) int {
	return value + 1
}

var GlobalAggregate = Aggregate{
	Name:   "global",
	Values: [3]NamedInt{1, 2, 3},
	Node:   &Node{Value: 5},
}

func debuggerFixtures() {
	integer := 42
	truth := true
	floating := 3.5
	var alias AliasInt = 17
	named := NamedInt(18)
	recursive := &Node{Value: 1, Next: &Node{Value: 2}}
	aggregate := Aggregate{
		Name:   "local",
		Values: [3]NamedInt{4, 5, 6},
		Node:   recursive,
	}
	text := "hello 世界"
	values := []int{7, 8, 9}
	mapping := map[string]int{"answer": 42}
	queue := make(chan int, 3)
	queue <- 10
	queue <- 11
	var greeter Greeter = Prefix("hello ")
	plain := addOne
	base := 20
	closure := func(value int) int { return base + value }
	counter := &Counter{Base: 30}
	bound := counter.Add
	plainResult := plain(1)
	closureResult := closure(2)
	boundResult := bound(3)
	interfaceResult := greeter.Greet("LLGo")
	mapResult := mapping["answer"]

	ready := make(chan struct{})
	release := make(chan struct{})
	result := make(chan int, 1)
	go func() {
		close(ready)
		<-release
		result <- 81
	}()
	<-ready
	println(
		integer, truth, floating, alias, named, recursive, &aggregate,
		text, values, mapping, queue, greeter,
		plain, closure, bound,
		plainResult, closureResult, boundResult, interfaceResult, mapResult,
		&GlobalAggregate,
	) // DEBUGGER_BREAK: all_values
	close(release)
	if got := <-result; got != 81 {
		panic("goroutine result mismatch")
	}
}

func main() {
	debuggerFixtures()
}
