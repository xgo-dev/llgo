package main

import "sort"

func main() {
	c1 := make(chan string)
	c2 := make(chan string, 1)
	go func() {
		c1 <- "ch1"
	}()
	go func() {
		c2 <- "ch2"
	}()

	var messages []string
	for i := 0; i < 2; i++ {
		select {
		case msg1 := <-c1:
			messages = append(messages, msg1)
		case msg2 := <-c2:
			messages = append(messages, msg2)
		}
	}
	// Select order is intentionally unspecified; compare the received values.
	sort.Strings(messages)
	for _, message := range messages {
		println(message)
	}
}
