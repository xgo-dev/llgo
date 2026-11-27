package main

type greeting string

func (g greeting) Greet() string { return string(g) }

func main() {
	text := "browser runtime"
	values := []int{3, 5, 8}
	mapping := map[string]int{"answer": 42}
	queue := make(chan int, 1)
	var greeter interface{ Greet() string } = greeting(text)
	closure := func() int { return len(text) + values[0] }
	queue <- mapping["answer"]
	println(text, len(values), <-queue, greeter.Greet(), closure())
}
