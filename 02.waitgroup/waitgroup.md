### Exp1

        waitgroup主要是用于用wait阻塞当前协程，然后让其他协程调用done来减少wait的数量，然后是用add增加要等待的数量，另外waitgroup是不能够进行复制的

```go
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
```

    waitgroup的使用主要是三个Add(),Wait(),Done(),Done()的本质就是调用Add(-1),

### Exp2

        实验二则是通过waitgroup让三个协程顺序打印答案

```go
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
```

        通过这个方法就能够做到一步一步的进行释放，上锁阻塞，或许我可以换一个理解，waitgroup本质上是一个拓扑排序的过程

        不过上面的代码有一个非常严重的问题，waitgroup在执行wg[2].Wait的时候是有一段时间的延迟，在wait放行但是wait还没有结束的时候，如果马上执行了一个add，就会爆panic

### Exp3

        exp3是对waitgroup赋值的实验，通过channel能够进行值得拷贝，但是不能够对地址进行拷贝，waitgroup不能够进行地址拷贝，值拷贝得时候add得状态会保留，如果传地址的话就会报错

```go
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
	wg.Done()
	wg.Wait()
}

func Exp3() { //进程间channel通信能够传递waitgroup吗
	ch := make(chan sync.WaitGroup, 10)
	Print4(ch)
	Print5(ch)
}

```

### Exp4 Go方法的使用

        对于Go的使用，首先是go的源码

```go
func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer func() {
			if x := recover(); x != nil {
				//如果是panic进的defer，那么就需要捕获panic然后爆panic
                //避免直接done后还没来得及打印panic整个程序就结束了
				panic(x)
			}
			wg.Done()
		}()
		f()
	}()
}
```

WaitGroup.Go的本质就是让传入的函数**自动的加上add和done的组合拳**    

下面是使用

```go
func Exp4() { //Go方法是一个语法糖，封装了add和done
	wg := sync.WaitGroup{}
	wg.Go(func() {
		fmt.Println("等待函数执行完")
		time.Sleep(time.Second * 2)
	})

	wg.Wait()
	fmt.Println("完成执行")
}
```
