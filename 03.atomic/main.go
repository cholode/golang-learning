package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

var count int

func add1() {
	for i := 0; i < 100000; i++ {
		count++
	}
}

func unatomic() {
	go add1()
	go add1()
	time.Sleep(time.Second)
	fmt.Println(count) // 输出永远小于 20000
}

var count1 int64

func add2() {
	for i := 0; i < 100000; i++ {
		atomic.AddInt64(&count1, 1) // 原子加1，不可分割
	}
}

func isatomic() {
	go add2()
	go add2()
	time.Sleep(time.Second)
	fmt.Println(atomic.LoadInt64(&count1)) // 输出 20000，永远正确
}

func Exp1() {
	unatomic()
	isatomic()
}

func main() {
	Exp1()
}
