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

func Print1(ch chan int, round *int) {
	for {
		msg := <-ch
		fmt.Printf("dog %d\n", msg)
		*round = *round + 1
		time.Sleep(time.Second * 1)
		ch <- msg
	}
}

func Print2(ch chan int, round *int) {
	for {
		msg := <-ch
		fmt.Printf("cat %d\n", msg)
		*round = *round + 1
		time.Sleep(time.Second * 1)
		ch <- msg
	}
}

func Print3(ch chan int, round *int) {
	for {
		msg := <-ch
		fmt.Printf("snake %d\n", msg)
		*round = *round + 1
		time.Sleep(time.Second * 1)
		ch <- msg
	}
}

func Exp2() {
	ch := make(chan int, 15)
	round := 0
	go Print1(ch, &round)
	go Print2(ch, &round)
	go Print3(ch, &round)
	ch <- 1
	ch <- 2
	for {
		if round > 15 {
			break
		}
	}
}

func main() {

	Exp2()

}
