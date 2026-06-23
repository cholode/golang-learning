### 源码

```go

package sync

import "runtime"

// RWMutex 读写互斥锁
// 支持「多读单写」：任意数量读者共享读锁；写锁排他，同一时间仅一个写者持有
// 零值即可直接使用，禁止值拷贝
type RWMutex struct {
	w           Mutex        // 写者互斥锁：保证同一时间最多一个写者进入写锁逻辑，同时保护写锁相关状态
	writerSem   uint32       // 写者信号量：写者等待读者全部完成时阻塞在此，最后一个读者读完后唤醒写者
	readerSem   uint32       // 读者信号量：有写者持有时，新读者阻塞在此，写者释放锁后批量唤醒所有等待读者
	readerCount atomic.Int32 // 读者复合计数器
	// 正数：当前活跃持有读锁的读者数量
	// 负数：有写者正在等待/持有写锁（实际值 = 真实读者数 - rwmutexMaxReaders）
	readerWait atomic.Int32 // 写者待等待读者数
	// 有写者等待时，记录「还在持有读锁的读者数量」，归零时唤醒等待的写者
}

// rwmutexMaxReaders 理论最大读者数，同时作为写锁标记位
// 写者加锁时会将 readerCount 减去该值，使 readerCount 变为负数，标记「有写者存在」
// 取值 1<<30 远大于实际并发读者数，同时留出符号位区分正负状态
const rwmutexMaxReaders = 1 << 30

// RLock 加读锁（共享锁）
// 多个 goroutine 可同时持有读锁，读锁之间不互斥
func (rw *RWMutex) RLock() {
	// 原子操作：读者计数 +1
	// 若结果 < 0：说明当前有写者正在等待/持有写锁，当前读者需要阻塞等待
	if rw.readerCount.Add(1) < 0 {
		// 阻塞在读者信号量上，等待写者释放锁后被唤醒
		runtime_SemacquireRWMutexR(&rw.readerSem, false, 0)
	}
}

// TryRLock 非阻塞尝试加读锁
// 有写者正在持有/等待锁时直接返回失败，不会进入阻塞
func (rw *RWMutex) TryRLock() bool {
	// 循环 CAS 重试，应对并发争抢场景
	for {
		c := rw.readerCount.Load()
		// 计数为负 → 有写者存在，加读锁失败
		if c < 0 {
			return false
		}
		// 原子 CAS 尝试将计数 +1，成功则加锁完成
		if rw.readerCount.CompareAndSwap(c, c+1) {
			return true
		}
	}
}

// RUnlock 释放读锁
// 必须与 RLock 配对调用，无锁状态下重复释放会触发进程致命错误
func (rw *RWMutex) RUnlock() {
	// 原子操作：读者计数 -1
	// 若结果 < 0：说明有写者正在等待锁，进入慢路径检查是否需要唤醒写者
	if r := rw.readerCount.Add(-1); r < 0 {
		rw.rUnlockSlow(r)
	}
}

// rUnlockSlow 读锁释放的慢路径：处理有写者等待的场景
// 当最后一个正在读的读者释放锁时，唤醒阻塞的写者
func (rw *RWMutex) rUnlockSlow(r int32) {
	// 非法解锁校验
	// r+1 == 0：原本就没有读者，无锁状态下调用 RUnlock
	// r+1 == -rwmutexMaxReaders：无任何读者的情况下重复解锁
	if r+1 == 0 || r+1 == -rwmutexMaxReaders {
		fatal("sync: RUnlock of unlocked RWMutex")
	}

	// 待等待读者数 -1，减到 0 说明所有正在读的读者都已完成
	if rw.readerWait.Add(-1) == 0 {
		// 释放写者信号量，唤醒阻塞的写者
		runtime_Semrelease(&rw.writerSem, false, 1)
	}
}

// Lock 加写锁（排他锁）
// 写锁与所有读锁、其他写锁都互斥，同一时间仅一个写者可持有
// 写优先：写锁到来后，后续新的读锁会被阻塞，避免写者饿死
func (rw *RWMutex) Lock() {
	// 第一步：抢占写者互斥锁，保证同一时间只有一个写者进入后续逻辑
	rw.w.Lock()

	// 第二步：标记写锁存在
	// 将 readerCount 减去 rwmutexMaxReaders，使其变为负数，标记「有写者在」
	// 计算得到 r = 减去最大值之前的读者数 = 当前正在持有读锁的活跃读者数量
	r := rw.readerCount.Add(-rwmutexMaxReaders) + rwmutexMaxReaders

	// r != 0：当前还有读者正在持锁，写者需要等待这些读者全部完成
	// readerWait.Add(r) != 0：将待等待读者数加上 r，若结果不为 0 则需要阻塞
	if r != 0 && rw.readerWait.Add(r) != 0 {
		// 阻塞在写者信号量上，等待最后一个读者读完后被唤醒
		runtime_SemacquireRWMutex(&rw.writerSem, false, 0)
	}
}

// TryLock 非阻塞尝试加写锁
// 抢锁失败立即返回，不会进入阻塞
func (rw *RWMutex) TryLock() bool {
	// 第一步：尝试抢占写者互斥锁，失败直接返回
	if !rw.w.TryLock() {
		return false
	}

	// 第二步：尝试将 readerCount 从 0 置为 -rwmutexMaxReaders
	// 仅当当前无任何读者、无其他写者时才能成功
	if !rw.readerCount.CompareAndSwap(0, -rwmutexMaxReaders) {
		// 失败则回退：释放已抢到的写者锁，返回 false
		rw.w.Unlock()
		return false
	}

	return true
}

// Unlock 释放写锁
// 必须与 Lock 配对调用，无锁状态下重复释放会触发进程致命错误
func (rw *RWMutex) Unlock() {
	// 将 readerCount 加回 rwmutexMaxReaders，恢复为正数，标记写锁释放
	// r = 写锁持有期间，阻塞等待读锁的读者数量
	r := rw.readerCount.Add(rwmutexMaxReaders)

	// 非法解锁校验：结果 >= 最大值，说明原本就没有写锁，重复调用 Unlock
	if r >= rwmutexMaxReaders {
		fatal("sync: Unlock of unlocked RWMutex")
	}

	// 批量唤醒所有阻塞等待读锁的 goroutine
	for i := 0; i < int(r); i++ {
		runtime_Semrelease(&rw.readerSem, false, 0)
	}

	// 释放写者互斥锁，允许其他写者参与抢锁
	rw.w.Unlock()
}

// syscall_hasWaitingReaders 判断写锁持有期间是否有读者在等待
// 仅供标准库系统调用内部使用，外部业务代码不会调用
func syscall_hasWaitingReaders(rw *RWMutex) bool {
	r := rw.readerCount.Load()
	// r < 0：有写者持有锁；r + 最大值 > 0：有读者在阻塞等待
	return r < 0 && r+rwmutexMaxReaders > 0
}

// RLocker 返回封装了读锁的 Locker 接口
// 方便将读写锁的读模式作为普通互斥锁，传递给需要 Locker 接口的场景
func (rw *RWMutex) RLocker() Locker {
	return (*rlocker)(rw)
}

// rlocker RWMutex 读锁包装器，实现 Locker 接口
type rlocker RWMutex

// Lock 实现 Locker 接口，对应加读锁
func (r *rlocker) Lock()   { (*RWMutex)(r).RLock() }

// Unlock 实现 Locker 接口，对应释放读锁
func (r *rlocker) Unlock() { (*RWMutex)(r).RUnlock() }

```