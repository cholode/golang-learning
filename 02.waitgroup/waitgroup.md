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
