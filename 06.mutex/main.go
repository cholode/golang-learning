package main

import (
	// "fmt"
	"fmt"
	"sync"
)

func add(count *int, wg *sync.WaitGroup, mu *sync.Mutex, islock bool) {
	if islock {
		mu.Lock()
	}
	*count = *count + 1
	if islock {
		mu.Unlock()
	}
	wg.Done()
}

func Exp1() {
	var wg sync.WaitGroup
	wg.Add(20000)
	var mu sync.Mutex
	count := 0
	lockcount := 0
	for i := 0; i < 10000; i++ { //无锁加法
		go add(&count, &wg, &mu, false)
	}
	for i := 0; i < 10000; i++ { //加锁的加法
		go add(&lockcount, &wg, &mu, true)
	}
	wg.Wait()
	fmt.Printf("count is :%d\n", count)
	//count is :9821
	fmt.Printf("lockcount is :%d\n", lockcount)
	//lockcount is :10000
}

func main() {
	Exp1()
}
