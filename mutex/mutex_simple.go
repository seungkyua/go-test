package main

import (
	"fmt"
	"sync"
	"time"
)

var counter int
var mu sync.Mutex

func increment() {
	mu.Lock()
	counter++
	mu.Unlock()
}

func main() {
	for i := 0; i < 1000; i++ {
		go increment()
	}
	time.Sleep(time.Second)
	fmt.Println(counter)
}
