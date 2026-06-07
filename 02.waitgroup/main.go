package main

import (
	"fmt"
	"sync"
	"time"
)

func Exp1() { //waitgroup的核心流程是定义变量，设施done的次数，用done在协程内关闭，用waitgroup等待其他协程
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		fmt.Printf("等待协程完成任务。。。\n")
		time.Sleep(time.Second * 1)
		fmt.Println("任务完成")
		wg.Done()

	}()

	wg.Wait()
	fmt.Println("协程正常退出")

}

func main() {

	Exp1()

}
