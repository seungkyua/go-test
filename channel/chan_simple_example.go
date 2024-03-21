package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int)
	go func() {
		time.Sleep(2000 * time.Millisecond)
		ch <- 1
	}()
	fmt.Println(<-ch)
}
