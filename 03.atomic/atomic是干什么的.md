## `atomic` 原子包：无锁并发的基石

### 1. 核心定义

`atomic` 包提供了**硬件级别的原子操作**，保证对单个内存地址的读写操作是**不可分割的**。

什么是原子性？

> 一个操作要么全部完成，要么完全没有发生，中间不会被任何其他 goroutine 打断。

### 2. 为什么需要 atomic？

普通的变量读写在并发环境下是不安全的。比如：

```go
var count int

func add() {
    for i := 0; i < 10000; i++ {
        count++
    }
}

func main() {
    go add()
    go add()
    time.Sleep(time.Second)
    fmt.Println(count) // 输出永远小于 20000
}
```

`count++` 不是原子操作，它会被拆成三步：

1. 从内存读取 `count` 的值到 CPU 寄存器
2. 在寄存器中加 1
3. 把结果写回内存

两个 goroutine 可能同时读到同一个值，各自加 1 后写回，导致一次加操作被覆盖。

而用 `atomic` 包就能解决这个问题：

```go
var count int64

func add() {
    for i := 0; i < 10000; i++ {
        atomic.AddInt64(&count, 1) // 原子加1，不可分割
    }
}

func main() {
    go add()
    go add()
    time.Sleep(time.Second)
    fmt.Println(atomic.LoadInt64(&count)) // 输出 20000，永远正确
}
```
