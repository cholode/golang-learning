package main

import (
	"fmt"
	"sync"
)

func Print1() {
	fmt.Println("我是1号")
}

func Print2() {
	fmt.Println("我是2号")
}

func Exp1() {
	var oc sync.Once
	oc.Do(Print1)
	oc.Do(Print2)
}

func main() {
	Exp1()
}
