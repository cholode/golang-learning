package rwmutex

import (
	"fmt"
)

func main() {
	fmt.Println("rwmutex test:")

}

type RWMutex struct {
	w           Mutex        // held if there are pending writers
	writerSem   uint32       // semaphore for writers to wait for completing readers
	readerSem   uint32       // semaphore for readers to wait for completing writers
	readerCount atomic.Int32 // number of pending readers
	readerWait  atomic.Int32 // number of departing readers
}

const rwmutexMaxReaders = 1 << 30

func (rw *RWMutex) RLock() {

	if rw.readerCount.Add(1) < 0 {
		runtime_SemacquireRWMutexR(&rw.readerSem, false, 0)
	}

}

func (rw *RWMutex) TryRLock() bool {

	for {
		c := rw.readerCount.Load()
		if c < 0 {
			return false
		}
		if rw.readerCount.CompareAndSwap(c, c+1) {
			return true
		}
	}
}

func (rw *RWMutex) RUnlock() {

	if r := rw.readerCount.Add(-1); r < 0 {
		rw.rUnlockSlow(r)
	}

}

func (rw *RWMutex) rUnlockSlow(r int32) {
	if r+1 == 0 || r+1 == -rwmutexMaxReaders {
		fatal("sync: RUnlock of unlocked RWMutex")
	}
	if rw.readerWait.Add(-1) == 0 {
		runtime_Semrelease(&rw.writerSem, false, 1)
	}
}

func (rw *RWMutex) Lock() {

	rw.w.Lock()
	r := rw.readerCount.Add(-rwmutexMaxReaders) + rwmutexMaxReaders
	if r != 0 && rw.readerWait.Add(r) != 0 {
		runtime_SemacquireRWMutex(&rw.writerSem, false, 0)
	}

}

func (rw *RWMutex) TryLock() bool {

	if !rw.w.TryLock() {
		return false
	}
	if !rw.readerCount.CompareAndSwap(0, -rwmutexMaxReaders) {
		rw.w.Unlock()
		return false
	}
	return true
}

func (rw *RWMutex) Unlock() {

	r := rw.readerCount.Add(rwmutexMaxReaders)
	if r >= rwmutexMaxReaders {
		fatal("sync: Unlock of unlocked RWMutex")
	}
	for i := 0; i < int(r); i++ {
		runtime_Semrelease(&rw.readerSem, false, 0)
	}
	rw.w.Unlock()

}

func syscall_hasWaitingReaders(rw *RWMutex) bool {
	r := rw.readerCount.Load()
	return r < 0 && r+rwmutexMaxReaders > 0
}

func (rw *RWMutex) RLocker() Locker {
	return (*rlocker)(rw)
}

type rlocker RWMutex

func (r *rlocker) Lock()   { (*RWMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*RWMutex)(r).RUnlock() }
