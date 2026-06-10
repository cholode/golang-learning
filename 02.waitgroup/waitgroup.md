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
