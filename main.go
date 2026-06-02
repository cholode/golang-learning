package main

import (
	"fmt"
	"time"
)

func Exp1() {
	ch := make(chan string)
	msg := <-ch
	go func() {
		time.Sleep(time.Second * 2)
		ch <- "hello world"

		fmt.Println("goroutine exit successful")
	}()
	fmt.Println("waiting for channel...")

	fmt.Println("now, i get the channel")

	fmt.Println(msg)
}

func main() {

	Exp1()

}
