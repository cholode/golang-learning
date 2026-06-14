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

var config string
var ready bool

func setup() {
	config = "初始化完成的配置内容" // 写1：初始化数据
	ready = true          // 写2：标记完成
}

func Exp2() bool { //CPU指令重排可能导致config没有被执行，高并发有概率发生，单线程不可能发生
	go setup()

	// 主 goroutine 循环等待标记
	for !ready {
		// 空转
	}
	//println(config) // 可能打印空字符串！
	if config != "初始化完成的配置内容" {
		fmt.Println("初始化失败")
		return false
	}
	return true
}

func main() {
	for Exp2() {
	}

}
