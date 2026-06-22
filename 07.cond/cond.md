# cond

### 源码

```go
// Cond 条件变量：Go 并发同步原语
// 核心能力：让一个/多个 goroutine 阻塞等待某个条件，条件满足时由其他 goroutine 唤醒
// 强制约束：必须配合互斥锁 Locker 一同使用，保护共享条件的并发安全；禁止值拷贝，必须指针传递
type Cond struct {
	// noCopy：编译期静态防拷贝标记
	noCopy noCopy

	// L：保护共享条件的互斥锁
	L Locker

	// notify：goroutine 等待队列（底层由 runtime 维护的链表结构）
	// Signal 唤醒队头一个，Broadcast 唤醒全部
	notify notifyList

	// checker：运行期动态防拷贝检查器
	// 和 noCopy 形成双层防护：noCopy 管编译期，checker 管运行时
	// 每次调用公开方法都会执行检查，发现值拷贝直接 panic，避免出现更隐蔽的并发 bug
	checker copyChecker
}

// NewCond Cond 构造函数
// 设计细节：返回指针类型，从根源上避免调用方值拷贝
// 参数 l：用户传入的互斥锁，用于保护条件变量关联的共享状态
func NewCond(l Locker) *Cond {
	return &Cond{L: l}
}

// Wait 阻塞当前 goroutine，等待条件满足
// ⚠️ 调用前必须先持有 c.L 锁，否则会出现并发安全问题
// 执行流程：登记等待队列 → 释放锁 → 阻塞挂起 → 被唤醒 → 重新加锁 → 返回
func (c *Cond) Wait() {
	// 第一步：运行时检查 Cond 是否被值拷贝，拷贝过直接 panic
	c.checker.check()

	// 第二步：将当前 goroutine 登记到等待队列中，返回等待位置票据 t
	// 这一步仅登记身份，不会真正阻塞
	t := runtime_notifyListAdd(&c.notify)

	// 第三步：释放互斥锁 —— 这是条件变量最核心的设计
	// 为什么必须解锁？
	// 如果占着锁阻塞，其他 goroutine 永远无法修改共享条件，也就永远无法唤醒等待者，直接形成死锁
	c.L.Unlock()

	// 第四步：真正陷入内核阻塞，挂起当前 goroutine
	// 直到被 Signal / Broadcast 唤醒，或者被意外唤醒（虚假唤醒）
	runtime_notifyListWait(&c.notify, t)

	// 第五步：被唤醒后重新获取互斥锁
	// 保证方法返回后，调用方依然持有锁，可以安全地检查共享条件
	c.L.Lock()
}

// Signal 唤醒等待队列中最早等待的 1 个 goroutine（FIFO 顺序）
// 最佳实践：调用前持有 c.L 锁；不持锁也能调用，但可能出现唤醒丢失
// 注意：唤醒 ≠ 条件满足，被唤醒的 goroutine 必须重新检查条件
func (c *Cond) Signal() {
	c.checker.check()
	runtime_notifyListNotifyOne(&c.notify)
}

// Broadcast 唤醒等待队列中**全部** goroutine
// 适用场景：条件发生全局性变化，所有等待者都需要重新检查状态
func (c *Cond) Broadcast() {
	c.checker.check()
	runtime_notifyListNotifyAll(&c.notify)
}

// -------------------- 辅助机制：运行时拷贝检查 --------------------

// copyChecker 运行期拷贝检查器，底层是 uintptr 整数
// 原理：存储自身的内存地址，每次调用方法时比对地址是否变化
// 地址变化 = 结构体被值拷贝了，直接 panic 终止
type copyChecker uintptr

// check 执行拷贝校验
func (c *copyChecker) check() {
	// 三层判断，兼顾性能与准确性：
	// 1. 快速路径：存储的地址 == 当前自身地址 → 没被拷贝，直接返回
	if uintptr(*c) != uintptr(unsafe.Pointer(c)) &&
		//    CAS 返回 false = 已经被其他 goroutine 初始化过了
		!atomic.CompareAndSwapUintptr((*uintptr)(c), 0, uintptr(unsafe.Pointer(c))) &&
		// 3. 最终确认：CAS 失败后再次比对，排除并发初始化的误判
		uintptr(*c) != uintptr(unsafe.Pointer(c)) {
		// 确认发生值拷贝，直接 panic
		panic("sync.Cond is copied")
	}
}

// -------------------- 辅助机制：编译期静态防拷贝 --------------------

// noCopy 静态防拷贝标记，空结构体，不占用任何内存
type noCopy struct{}

// Lock 空实现，仅用于满足 Locker 接口，无实际逻辑
func (*noCopy) Lock()   {}

// Unlock 空实现，仅用于满足 Locker 接口，无实际逻辑
func (*noCopy) Unlock() {}


int
```

### Exp1


使用：
# sync.Cond 核心使用场景与选型指南
`sync.Cond` 是 Go 标准库中专门实现 **「条件等待 + 唤醒通知」** 的同步原语，核心定位是：
> 让多个 goroutine 阻塞等待**某个共享条件成立**；当条件被其他协程修改后，可以精准唤醒「一个」或「全部」等待协程。

它必须和互斥锁（`sync.Locker`）绑定使用，本质是「互斥锁 + 等待队列」的组合，专门解决 **「共享状态 + 多协程等待状态变化」** 的问题。单纯的互斥锁只能保护临界区，无法实现「条件不满足时主动让出CPU、条件满足时被唤醒」；而 channel 是消息传递模型，在多条件、重复广播、精准唤醒场景下不如 Cond 高效。

---

## 一、四大经典使用场景
### Exp1：有界队列 · 生产者-消费者模型（最核心场景）
这是条件变量的教科书级应用，也是操作系统中「生产者-消费者」问题的标准实现。
#### 场景说明
存在一个**固定容量**的共享队列：
- 生产者：往队列里放入数据，**队列满时必须阻塞**，直到消费者取走数据腾出空间；
- 消费者：从队列里取出数据，**队列空时必须阻塞**，直到生产者放入数据。

#### 为什么用 Cond 而不是 channel？
channel 本身就是有界队列，但它是黑盒的；如果你需要自定义队列结构、批量操作、多维度条件控制，用 `Mutex + Cond` 会更灵活：
- 可以用两个独立的 Cond 分别控制「队列不满」「队列不空」两个条件，**精准唤醒，避免惊群效应**；
- 没有消息拷贝开销，纯内存操作，性能更高。

#### 代码示例
```go
package main

import (
	"fmt"
	"sync"
)

// BoundedQueue 有界队列
type BoundedQueue struct {
	mu       sync.Mutex
	items    []int
	capacity int

	notFull  *sync.Cond // 队列不满：生产者等待的条件
	notEmpty *sync.Cond // 队列不空：消费者等待的条件
}

func NewBoundedQueue(cap int) *BoundedQueue {
	q := &BoundedQueue{
		capacity: cap,
		items:    make([]int, 0, cap),
	}
	q.notFull = sync.NewCond(&q.mu)
	q.notEmpty = sync.NewCond(&q.mu)
	return q
}

// Put 生产者：放入元素
func (q *BoundedQueue) Put(item int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 必须用 for 循环检查：防止虚假唤醒
	for len(q.items) == q.capacity {
		q.notFull.Wait() // 队列满了，阻塞等待「不满」条件
	}

	q.items = append(q.items, item)
	q.notEmpty.Signal() // 放入成功，唤醒一个等待的消费者
}

// Get 消费者：取出元素
func (q *BoundedQueue) Get() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 {
		q.notEmpty.Wait() // 队列空了，阻塞等待「不空」条件
	}

	item := q.items[0]
	q.items = q.items[1:]
	q.notFull.Signal() // 取出成功，唤醒一个等待的生产者
	return item
}

func main() {
	q := NewBoundedQueue(3)
	var wg sync.WaitGroup

	// 3个生产者
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 2; j++ {
				q.Put(id*10 + j)
				fmt.Printf("生产者%d 放入: %d\n", id, id*10+j)
			}
		}(i)
	}

	// 2个消费者
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 3; j++ {
				item := q.Get()
				fmt.Printf("消费者%d 取出: %d\n", id, item)
			}
		}(i)
	}

	wg.Wait()
}
```

---

### 场景2：可重复的全局广播通知
#### 场景说明
某个全局事件（配置热更新、服务暂停/恢复、开关切换）发生时，**所有正在等待的工作协程都要收到通知并继续执行**；并且这个事件可以**多次触发**。

#### 为什么用 Cond 而不是 channel？
- `close(channel)` 只能实现**一次广播**，关闭后无法复用；
- 如果用 channel 实现重复广播，需要不断创建新 channel、替换旧 channel，代码繁琐且有并发安全问题；
- `Cond.Broadcast()` 可以**反复调用**，每次都能唤醒所有等待者，语义清晰，开销极低。

#### 典型业务场景
- 配置中心推送更新：配置变更后，所有工作协程重新加载配置；
- 服务优雅降级：开关打开时所有协程进入降级逻辑，开关关闭时恢复。

#### 代码示例（配置热更新广播）
```go
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

// Update 配置更新，广播通知所有等待的协程
func (c *Config) Update() {
	c.mu.Lock()
	c.version++
	c.mu.Unlock()
	c.cond.Broadcast() // 广播：唤醒所有等待配置更新的协程
}

// WaitUpdate 协程阻塞等待配置更新
func (c *Config) WaitUpdate(lastVersion int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	for c.version == lastVersion {
		c.cond.Wait() // 版本没变化，阻塞等待
	}
	return c.version
}
```

---

### 场景3：多协程同步启动（发令枪模型）
#### 场景说明
N 个工作协程先各自完成初始化，全部就绪后等待同一个「启动信号」；信号到来后，**所有协程同时开始执行任务**。
类比赛跑：运动员各自热身（初始化），全部到起跑线后等待枪响，枪响后同时开跑。

#### 为什么用 Cond？
- `WaitGroup` 是「等待所有协程完成」，反过来「所有协程等待同一个启动信号」的场景，Cond 更直接；
- 可以实现「一对多精准同步」，启动信号可以重复触发。

#### 代码示例
```go
func main() {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	ready := false
	var wg sync.WaitGroup

	// 启动5个工作协程
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("协程%d 初始化完成，等待启动...\n", id)

			mu.Lock()
			for !ready {
				cond.Wait() // 阻塞等待启动信号
			}
			mu.Unlock()

			fmt.Printf("协程%d 开始执行任务\n", id)
		}(i)
	}

	// 主协程准备一秒后发令
	time.Sleep(time.Second)
	fmt.Println("=== 发令：所有协程启动 ===")

	mu.Lock()
	ready = true
	mu.Unlock()
	cond.Broadcast() // 广播：唤醒所有等待的协程

	wg.Wait()
}
```

---

### 场景4：资源池/连接池的空闲等待
#### 场景说明
数据库连接池、协程池、对象池这类资源管理器：
- 当没有空闲资源时，申请资源的协程需要**阻塞等待**；
- 当有资源被归还时，**唤醒一个等待的协程**去获取资源。

#### 为什么用 Cond？
- 资源池的状态是共享的，必须用锁保护；
- 「有空闲资源」是一个典型的等待条件，Cond 语义完全匹配；
- 归还资源时用 `Signal` 唤醒一个等待者，不会造成惊群效应，性能远高于轮询。

#### 极简代码示意
```go
type Pool struct {
	mu      sync.Mutex
	conns   []*Conn
	cond    *sync.Cond
}

// 获取连接：没有空闲则阻塞等待
func (p *Pool) Get() *Conn {
	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.conns) == 0 {
		p.cond.Wait()
	}

	conn := p.conns[0]
	p.conns = p.conns[1:]
	return conn
}

// 归还连接：唤醒一个等待者
func (p *Pool) Put(conn *Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.conns = append(p.conns, conn)
	p.cond.Signal() // 唤醒一个等待连接的协程
}
```

---

## 二、选型对比：什么时候不用 Cond？
Cond 是底层同步原语，API 偏原始，容易用错。简单场景优先使用更高级的同步工具。

| 场景 | 优先选择 | 不推荐 | 原因 |
|------|----------|--------|------|
| 有界队列、资源池、多条件精准唤醒 | `sync.Cond` | channel | 多条件控制灵活，无消息拷贝，性能更高 |
| 单次广播通知（比如服务关闭） | `close(channel)` | Cond | channel 用法更简单，无需配合锁 |
| 等待 N 个任务全部完成 | `sync.WaitGroup` | Cond | WaitGroup API 更简洁，语义更清晰 |
| 一对一消息传递、带超时的等待 | channel | Cond | channel 支持 `select`、超时、关闭，语义更直观 |
| 简单的互斥访问共享资源 | `sync.Mutex` | Cond | 不需要条件等待，直接用锁即可 |

---

## 三、使用最佳实践（必看）
1.  **`Wait` 必须包在 `for` 循环里，绝对不能用 `if`**
    应对虚假唤醒、条件被其他协程抢先修改、广播后抢锁失败等场景，保证被唤醒后条件一定成立。

2.  **调用 `Wait` 前必须持有锁**
    Wait 内部会自动解锁、唤醒后自动加锁；调用前没加锁会触发「对未加锁的锁执行 Unlock」的 panic。

3.  **优先用 `Signal`，少用 `Broadcast`**
    只有一个等待者能满足条件时，用 Signal 精准唤醒，避免惊群效应；只有全局状态变化、所有等待者都需要感知时，才用 Broadcast。

4.  **Cond 永远传指针，禁止值拷贝**
    拷贝后的 Cond 内部等待队列、锁状态全部错乱，运行时会直接 panic。

5.  **不要为了用而用**
    简单的同步场景优先用 channel、WaitGroup；只有在「共享状态 + 多协程条件等待 + 精准/重复唤醒」的场景下，Cond 才是最优解。