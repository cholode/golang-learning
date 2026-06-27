# goroutine+channel的实验

        在普通的函数中<-chan会导致进程阻塞等待直到chan中传入信息

### channel源码

##### channel 管道的数据结构

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

##### 内存分配的源码

```go
// makechan 是 Go 运行时创建 channel 的底层实现函数
// 对应用户层代码 make(chan T, size) 的底层调用
// 参数：
//   t    *chantype - channel 的类型元信息，包含元素类型、大小、对齐、是否含指针等属性
//   size int       - channel 缓冲区容量，即 make 时指定的缓冲大小
// 返回值：
//   *hchan - 初始化完成的 channel 底层结构体指针
func makechan(t *chantype, size int) *hchan {
	// 提取 channel 存储元素的类型元信息
	elem := t.Elem

	// ========== 第一阶段：参数合法性校验 ==========

	// 限制单个元素大小不超过 64KB（1<<16 = 65536 字节）
	// 过大的元素会导致 channel 拷贝开销剧增，Go 直接禁止该场景
	if elem.Size_ >= 1<<16 {
		throw("makechan: invalid channel element type")
	}

	// 内存对齐合法性校验
	// 1. hchan 结构体自身大小必须是系统最大对齐数的整数倍，保证结构体内存布局对齐
	// 2. 元素的对齐要求不能超过系统最大对齐上限，否则缓冲区内存无法正确排布
	if hchanSize%maxAlign != 0 || elem.Align_ > maxAlign {
		throw("makechan: bad alignment")
	}

	// ========== 第二阶段：安全计算缓冲区总内存 ==========

	// math.MulUintptr 安全计算「单个元素大小 × 缓冲区容量」的总内存
	// 返回值 mem 为总字节数，overflow 标记乘法是否发生整数溢出
	// 相比普通乘法，可避免大数相乘导致的溢出漏洞
	mem, overflow := math.MulUintptr(elem.Size_, uintptr(size))
	// 三种非法情况直接 panic：
	// 1. 乘法计算溢出；2. 总内存超过 Go 单对象最大可分配上限；3. 缓冲区大小为负数
	if overflow || mem > maxAlloc-hchanSize || size < 0 {
		panic(plainError("makechan: size out of range"))
	}

	// ========== 第三阶段：分场景分配内存（核心设计） ==========
	// 根据元素特性、缓冲区大小，分三种策略分配 hchan 结构体 + 缓冲区内存
	var c *hchan
	switch {
	// 场景1：缓冲区内存为 0
	// 对应两种情况：无缓冲 channel（size=0），或元素是零大小类型（如 struct{}）
	case mem == 0:
		// 仅分配 hchan 结构体本身的内存，无需单独分配数据缓冲区
		// mallocgc 第三个参数为 true 表示分配后自动清零内存
		c = (*hchan)(mallocgc(hchanSize, nil, true))
		// buf 指向 hchan 内部的竞态检测地址
		// 无缓冲 channel 不存储实际数据，该地址仅用于 race 竞态检测
		c.buf = c.raceaddr()

	// 场景2：元素不包含指针
	// 此时 hchan 结构体 + 缓冲区可分配为一段连续内存
	// 优势：提升内存局部性、减少 GC 扫描开销（无指针则 GC 无需遍历缓冲区）
	case !elem.Pointers():
		// 一次性分配「hchan 结构体 + 数据缓冲区」的连续内存块
		c = (*hchan)(mallocgc(hchanSize+mem, nil, true))
		// 缓冲区起始地址 = hchan 结构体首地址 + 结构体自身大小
		// 通过地址偏移直接定位到连续内存中的缓冲区区域
		c.buf = add(unsafe.Pointer(c), hchanSize)

	// 场景3：元素包含指针（默认分支）
	// 缓冲区需单独分配，因为 GC 需要扫描缓冲区中的指针引用
	// 分开分配可避免 hchan 结构体干扰缓冲区的指针扫描逻辑
	default:
		// 先分配 hchan 结构体本身
		c = new(hchan)
		// 单独分配缓冲区内存，传入元素类型信息
		// 帮助 GC 正确识别、扫描缓冲区中的指针对象
		c.buf = mallocgc(mem, elem, true)
	}

	// ========== 第四阶段：初始化 hchan 核心字段 ==========

	// 单个元素的字节大小，用 uint16 存储以节省结构体内存空间
	c.elemsize = uint16(elem.Size_)
	// 保存元素类型元信息，后续用于元素拷贝、GC 扫描、类型校验等
	c.elemtype = elem
	// 环形缓冲区的总容量（channel 的缓冲大小）
	c.dataqsiz = uint(size)

	// 初始化 channel 内部的互斥锁
	// lockRankHchan 指定锁的优先级等级，Go 运行时通过锁排序避免死锁
	lockInit(&c.lock, lockRankHchan)

	// ========== 第五阶段：调试输出 ==========
	// 开启 channel 调试开关（GODEBUG 环境变量控制）时，打印创建详情
	if debugChan {
		print("makechan: chan=", c, "; elemsize=", elem.Size_, "; dataqsiz=", size, "\n")
	}

	// 返回初始化完成的 channel 底层结构体
	return c
}
```

##### channel 通道发送源码

```go

// chansend1 是单元素通道发送的包装入口
// 对应用户层代码 `ch <- x` 的底层调用，默认阻塞模式发送
// 参数：
//   c    *hchan         - 目标 channel 底层结构体
//   elem unsafe.Pointer - 待发送元素的内存地址
func chansend1(c *hchan, elem unsafe.Pointer) {
	// 调用核心发送函数，block=true 表示默认阻塞等待
	// sys.GetCallerPC() 获取调用方程序计数器，用于性能追踪与竞态检测
	chansend(c, elem, true, sys.GetCallerPC())
}

// chansend 是 channel 发送操作的核心实现
// 参数：
//   c        *hchan         - 目标 channel
//   ep       unsafe.Pointer - 待发送元素的指针
//   block    bool           - 是否阻塞模式：true=阻塞等待，false=非阻塞（select 场景使用）
//   callerpc uintptr        - 调用者 PC 地址，用于 trace 和性能分析
// 返回值：bool - 发送是否成功
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
	// ========== 阶段1：nil channel 特殊处理 ==========
	if c == nil {
		// 非阻塞模式下向 nil channel 发送，直接返回失败
		if !block {
			return false
		}
		// 阻塞模式下向 nil channel 发送：永久挂起当前 goroutine
		// gopark 会让出 CPU，且不会被主动唤醒，对应语言规范：nil channel 发送永远阻塞
		gopark(nil, nil, waitReasonChanSendNilChan, traceBlockForever, 2)
		// 理论上永远执行不到这里，属于防御性代码
		throw("unreachable")
	}

	// channel 调试开关：开启时打印发送日志
	if debugChan {
		print("chansend: chan=", c, "\n")
	}

	// ========== 阶段2：非阻塞快速路径（无锁预判） ==========
	// 非阻塞模式 + channel 未关闭 + channel 已满 → 直接返回失败
	// 这是无锁的快速判断，避免不必要的加锁开销，提升 select 非阻塞场景性能
	// 存在极小的竞态窗口，但后续加锁后会二次校验，保证正确性
	if !block && c.closed == 0 && full(c) {
		return false
	}

	// 阻塞性能采样：记录发送开始时间，用于统计阻塞耗时
	var t0 int64
	if blockprofilerate > 0 {
		t0 = cputicks()
	}

	// ========== 阶段3：加锁进入临界区，核心发送逻辑 ==========
	lock(&c.lock)

	// 校验：向已关闭的 channel 发送数据，直接 panic
	// 语言规范：关闭的 channel 禁止发送，仅允许接收剩余数据
	if c.closed != 0 {
		unlock(&c.lock)
		panic(plainError("send on closed channel"))
	}

	// 发送路径1：有接收者正在等待 → 直接发送，跳过缓冲区
	// 从接收等待队列 recvq 中取出一个阻塞的接收 goroutine
	if sg := c.recvq.dequeue(); sg != nil {
		// send 函数完成三件事：
		// 1. 将待发送元素直接拷贝到接收者的内存地址
		// 2. 唤醒接收 goroutine
		// 3. 释放 channel 锁
		// 这是最快的发送路径，无需经过环形缓冲区拷贝
		send(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true
	}

	// 发送路径2：缓冲区有空位 → 写入环形缓冲区
	if c.qcount < c.dataqsiz {
		// 计算当前发送索引 sendx 对应的缓冲区内存地址
		qp := chanbuf(c, c.sendx)

		// 竞态检测开启时，记录该缓冲区位置的写入事件
		if raceenabled {
			racenotify(c, c.sendx, nil)
		}

		// 类型安全的内存拷贝：将元素 ep 拷贝到缓冲区位置 qp
		typedmemmove(c.elemtype, qp, ep)

		// 发送索引后移，形成环形队列：到达末尾则归零
		c.sendx++
		if c.sendx == c.dataqsiz {
			c.sendx = 0
		}
		// 缓冲区元素计数 +1
		c.qcount++

		unlock(&c.lock)
		return true
	}

	// 发送路径3：缓冲区已满 + 非阻塞模式 → 直接返回失败
	if !block {
		unlock(&c.lock)
		return false
	}

	// ========== 阶段4：阻塞模式，当前 goroutine 入队休眠 ==========
	// 获取当前 goroutine 结构体
	gp := getg()
	// 从 sudog 缓存池申请一个等待节点（sudog 是 goroutine 等待队列的封装）
	mysg := acquireSudog()
	// 初始化释放时间标记：-1 表示需要统计阻塞时长
	mysg.releasetime = 0
	if t0 != 0 {
		mysg.releasetime = -1
	}

	// 绑定待发送元素的地址，被唤醒后接收方会直接从这里读取数据
	mysg.elem.set(ep)
	mysg.waitlink = nil
	// 绑定当前 goroutine
	mysg.g = gp
	// 标记是否处于 select 多路复用场景
	mysg.isSelect = false
	// 绑定当前 channel
	mysg.c.set(c)
	// 当前 goroutine 标记自身正在等待该 sudog
	gp.waiting = mysg
	gp.param = nil

	// 将当前等待节点加入 channel 的发送等待队列尾部
	c.sendq.enqueue(mysg)
	// 标记 goroutine 正在 channel 上挂起，用于 GC 安全处理
	gp.parkingOnChan.Store(true)
	reason := waitReasonChanSend

	// 挂起当前 goroutine，让出 CPU
	// chanparkcommit 会在挂起前原子性地释放 channel 锁，避免唤醒竞态
	gopark(chanparkcommit, unsafe.Pointer(&c.lock), reason, traceBlockChanSend, 2)
	// 保活元素指针：防止挂起期间 GC 回收 ep 指向的内存
	KeepAlive(ep)

	// ========== 阶段5：goroutine 被唤醒，后续处理 ==========
	// 防御性校验：等待节点必须与自身绑定，否则队列结构损坏
	if mysg != gp.waiting {
		throw("G waiting list is corrupted")
	}
	// 清空等待标记
	gp.waiting = nil
	gp.activeStackChans = false

	// 判断唤醒原因：success=false 表示是 channel 关闭导致的唤醒
	closed := !mysg.success
	gp.param = nil

	// 统计阻塞耗时，写入性能分析器
	if mysg.releasetime > 0 {
		blockevent(mysg.releasetime-t0, 2)
	}

	// 解除与 channel 的绑定，归还 sudog 到缓存池
	mysg.c.set(nil)
	releaseSudog(mysg)

	// 如果是因 channel 关闭被唤醒，触发 panic
	if closed {
		// 二次校验 channel 确实已关闭，防止虚假唤醒
		if c.closed == 0 {
			throw("chansend: spurious wakeup")
		}
		panic(plainError("send on closed channel"))
	}

	// 正常唤醒：发送成功
	return true
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
