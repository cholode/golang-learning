package main

import (
	"fmt"
	"sync"
	"time"
)

func Exp1() {
	var rw sync.RWMutex
	var wg sync.WaitGroup
	val := 0
	ch := make(chan int)
	// 3个读协程：共享读锁，可同时执行
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ch:
					return

				default:
					rw.RLock() // 加读锁
					time.Sleep(time.Millisecond * 200)
					fmt.Println("读到:", val) // 几乎同时打印
					rw.RUnlock()            // 解读锁
				}
			}
		}()
	}

	// 1个写协程：排他写锁，会阻塞所有读
	time.Sleep(time.Second * 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		rw.Lock() // 加写锁
		val = 42
		fmt.Println("写入中")
		time.Sleep(time.Millisecond * 1000)
		rw.Unlock() // 解写锁
	}()
	time.Sleep(time.Second * 2)
	ch <- 1
	ch <- 2
	ch <- 3
	wg.Wait()
}

func main() {
	Exp1()

}
