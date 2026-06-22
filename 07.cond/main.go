package main

import (
	"fmt"
	"sync"
	"time"
)

type Config struct {
	mu      sync.Mutex
	version int
	cond    *sync.Cond
}

func NewConfig() *Config {
	c := &Config{}
	c.cond = sync.NewCond(&c.mu)
	return c
}

// Update 触发一次全局广播，可重复调用
func (c *Config) Update() {
	c.mu.Lock()
	c.version++
	c.mu.Unlock()
	c.cond.Broadcast() // 广播唤醒所有等待者
}

// WaitUpdate 阻塞等待版本更新，返回最新版本号
func (c *Config) WaitUpdate(lastVer int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.version == lastVer {
		c.cond.Wait()
	}
	return c.version
}

// 通过cond的broadcast进行批量的解除阻塞
func Exp1() {
	cfg := NewConfig()
	var wg sync.WaitGroup

	// 3个协程监听配置更新
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ver := 0
			for j := 0; j < 2; j++ { // 接收两次更新
				ver = cfg.WaitUpdate(ver)
				fmt.Printf("协程%d 收到更新，版本=%d\n", id, ver)
			}
		}(i)
	}

	// 触发两次更新，验证可重复广播
	time.Sleep(2 * time.Second)
	cfg.Update()
	time.Sleep(2 * time.Second)
	cfg.Update()

	wg.Wait()
}

func main() {
	Exp1()
}
