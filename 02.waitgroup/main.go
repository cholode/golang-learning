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

func Print1(round *int, wg []*sync.WaitGroup) {
	for {
		wg[0].Wait()
		wg[0].Add(1)
		fmt.Printf("dog %d\n", round)

		*round = *round + 1
		time.Sleep(time.Second * 1)

		wg[1].Done()
	}
}

func Print2(round *int, wg []*sync.WaitGroup) {
	for {
		wg[1].Wait()
		wg[1].Add(1)
		fmt.Printf("cat %d\n", round)
		*round = *round + 1
		time.Sleep(time.Second * 1)
		wg[2].Done()
	}
}

func Print3(round *int, wg []*sync.WaitGroup) {
	for {
		wg[2].Wait()
		wg[2].Add(1)

		fmt.Printf("snake %d\n", round)
		*round = *round + 1
		time.Sleep(time.Second * 1)
		wg[0].Done()
	}
}

func Exp2() { //通过waitgroup进行交替打印

	wgs := make([]*sync.WaitGroup, 3)
	for i := 0; i < 3; i++ {
		wgs[i] = &sync.WaitGroup{}
	}
	round := 0
	wgs[1].Add(1)
	wgs[2].Add(1)
	go Print1(&round, wgs)
	go Print2(&round, wgs)
	go Print3(&round, wgs)
	for {
		if round > 15 {
			break
		}
	}
}

func Print4(ch chan sync.WaitGroup) {
	var wg sync.WaitGroup
	wg.Add(1)
	ch <- wg
	fmt.Println("wait for P4")
	time.Sleep(time.Second * 2)
	wg.Done()
}

func Print5(ch chan sync.WaitGroup) {
	wg := <-ch //channel是值传递
	//wg.Done()
	wg.Wait()
}

func Exp3() { //进程间channel通信能够传递waitgroup吗
	ch := make(chan sync.WaitGroup, 10)
	Print4(ch)
	Print5(ch)
}

func main() {

	Exp3()

}
