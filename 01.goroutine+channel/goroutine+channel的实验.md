# goroutine+channel的实验

        在普通的函数中<-chan会导致进程阻塞等待直到chan中传入信息

### channel源码

```go
type hchan struct {
    // ---------- 环形缓冲区核心字段 ----------

    // qcount 当前环形缓冲区中已存储的元素总数
    // 等于 0 表示缓冲区空，等于 dataqsiz 表示缓冲区满
    qcount   uint           

    // dataqsiz 环形缓冲区的总容量
    // 对应 make(chan T, N) 中的 N；无缓冲 channel 该值为 0
    dataqsiz uint           

    // buf 指向环形缓冲区数组的指针
    // 是一段连续内存，可存放 dataqsiz 个元素，无缓冲 channel 该值为 nil
    buf      unsafe.Pointer 

    // elemsize 单个元素的字节大小
    // 用于计算缓冲区偏移、内存拷贝长度
    elemsize uint16         

    // closed 通道关闭标记
    // 0 = 通道开启，非 0 = 通道已关闭
    // 用 uint32 支持原子操作
    closed   uint32         

    // timer 关联的定时器
    // 仅 time.After / time.Tick 等定时器通道会使用，普通通道为 nil
    // 定时器到期后自动向该 channel 发送信号
    timer    *timer         

    // elemtype 通道元素的类型元信息
    // 包含类型大小、对齐方式、GC 标记等，用于内存拷贝、类型校验、垃圾回收扫描
    elemtype *_type         

    // sendx 发送操作的数组下标（写指针）
    // 下一次写入数据时存放的位置，到达 dataqsiz 后回卷到 0，形成环形队列
    sendx    uint           

    // recvx 接收操作的数组下标（读指针）
    // 下一次读取数据时读取的位置，到达 dataqsiz 后回卷到 0
    recvx    uint           

    // ---------- 等待队列字段 ----------

    // recvq 接收等待队列
    // 所有因读 channel 被阻塞的 goroutine 都会挂在这个链表上
    // 有数据写入时，会从队头唤醒一个等待的读协程
    recvq    waitq          

    // sendq 发送等待队列
    // 所有因写 channel 被阻塞的 goroutine 都会挂在这个链表上
    // 有数据读出时，会从队头唤醒一个等待的写协程
    sendq    waitq          

    // bubble 同步测试调试字段
    // 仅用于 runtime/synctest 协程同步测试框架，普通业务场景无作用
    bubble   *synctestBubble

    // ---------- 并发安全字段 ----------

    // lock 运行时内部互斥锁
    // 保护 hchan 所有字段的并发安全，所有修改状态、入队出队的操作都必须先持有这把锁
    // 是 channel 并发安全的底层保障
    lock mutex
}

```


### Exp1() goroutine和channel的阻塞运行

```go
ch := make(chan string)
go func() {
    time.Sleep(time.Second * 2)
    ch <- "hello world"
}()
fmt.Println("waiting for channel...")
msg := <-ch
fmt.Println("now, i get the channel")

fmt.Println(msg)
```

        在上面的代码中，因为sleep而暂停的函数不会往ch中传入信息，随后优先输出waiting...然后再在msg:=<-ch处阻塞等待函数的运行,在sleep结束后就会接着运行后面的部分

        如果将msg:<-ch放在了go func的前面，比如下面这样，就会发生死锁然后直接报错退出

```go
    ch := make(chan string)
    msg := <-ch
    go func() {
        time.Sleep(time.Second * 2)
        ch <- "hello world"

        fmt.Println("goroutine exit successful")
    }()

    fmt.Println(msg)
```

        对于项目应用的思考：对于这个组合，可以用于不同协程中的信息传递，可以先进行信息流的设计，然后再进行实现，设计信息流应该要保证先后时序

### Exp2()  多个go顺序打印

        通过channel使得go协程循序输出答案

```go
func Print1(ch chan int, round *int) {
	for {
		msg := <-ch
		fmt.Printf("dog %d\n", msg)
		*round = *round + 1
		time.Sleep(time.Second * 1)
		ch <- msg
	}
}

func Print2(ch chan int, round *int) {
	for {
		msg := <-ch
		fmt.Printf("cat %d\n", msg)
		*round = *round + 1
		time.Sleep(time.Second * 1)
		ch <- msg
	}
}

func Print3(ch chan int, round *int) {
	for {
		msg := <-ch
		fmt.Printf("snake %d\n", msg)
		*round = *round + 1
		time.Sleep(time.Second * 1)
		ch <- msg
	}
}

func Exp2() {
	ch := make(chan int, 15)
	round := 0
	go Print1(ch, &round)
	go Print2(ch, &round)
	go Print3(ch, &round)
	ch <- 1
	for {
		if round > 15 {
			break
		}
	}
}


//结果
dog 1
cat 1
snake 1
dog 1
cat 1
snake 1
dog 1
cat 1
snake 1
dog 1
cat 1
snake 1
dog 1
cat 1
snake 1
```

        在channel中有一个接受队列，recvq用于记录阻塞的go协程的队列，所以channel是保持着一个先进先出的算法，先阻塞的go协程先执行，这也就使得go协程有一个顺序执行的过程，不过三个动物的顺序或许会不一样，但每组的顺序都会是一样的

```go
func Exp2() {
	ch := make(chan int, 15)
	round := 0
	go Print1(ch, &round)
	go Print2(ch, &round)
	go Print3(ch, &round)
	ch <- 1
	ch <- 2
	for {
		if round > 15 {
			break
		}
	}
}

```



        当加入一个令牌2时，cat和dog就会因为更先执行使得snake经常抢不到令牌而阻塞，但这并不是绝对的，有时候可能因为工作窃取机制导致snake抢到了令牌

        下面是在不断ctrl+C获得的结果

```
PS D:\go-project\golanglearning\01.goroutine+channel> go run main.go

cat 2

dog 1

exit status 0xffffffffc000013a

PS D:\go-project\golanglearning\01.goroutine+channel> go run main.go

cat 2

dog 1

exit status 0xffffffffc000013a

PS D:\go-project\golanglearning\01.goroutine+channel> go run main.go

dog 1

cat 2

exit status 0xffffffffc000013a

PS D:\go-project\golanglearning\01.goroutine+channel> go run main.go

snake 2

dog 1
```
