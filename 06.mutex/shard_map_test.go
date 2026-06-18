package mutex

import (
	"math/rand"
	"testing"
)

const (
	totalKeySpace = 25000 // 总key空间：2500个不同的key
	writePerRound = 10000 // 每轮写入次数：1000次随机写入
)

// 预生成统一的随机写入序列，保证两个方案输入完全一致，测试公平
func genWriteKeys(count int) []int {
	keys := make([]int, count)
	for i := 0; i < count; i++ {
		keys[i] = rand.Intn(totalKeySpace) // 随机落在2500个key范围内
	}
	return keys
}

// ==================== 串行场景测试（单goroutine写入） ====================
func BenchmarkSingleLock_Serial(b *testing.B) {
	keys := genWriteKeys(writePerRound)
	m := NewSingleLockMap(totalKeySpace)

	b.ResetTimer() // 排除初始化开销
	for i := 0; i < b.N; i++ {
		for k := 0; k < writePerRound; k++ {
			m.Set(keys[k], k)
		}
	}
}

func BenchmarkSharded_Serial(b *testing.B) {
	keys := genWriteKeys(writePerRound)
	m := NewShardedMap(totalKeySpace)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k := 0; k < writePerRound; k++ {
			m.Set(keys[k], k)
		}
	}
}

// ==================== 并发场景测试（多goroutine同时写入） ====================
func BenchmarkSingleLock_Parallel(b *testing.B) {
	keys := genWriteKeys(writePerRound)
	m := NewSingleLockMap(totalKeySpace)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			key := keys[idx%writePerRound]
			m.Set(key, idx)
			idx++
		}
	})
}

func BenchmarkSharded_Parallel(b *testing.B) {
	keys := genWriteKeys(writePerRound)
	m := NewShardedMap(totalKeySpace)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			key := keys[idx%writePerRound]
			m.Set(key, idx)
			idx++
		}
	})
}
