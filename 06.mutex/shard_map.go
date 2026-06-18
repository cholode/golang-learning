package mutex

import "sync"

// ==================== 方案1：单map + 单把互斥锁 ====================
type SingleLockMap struct {
	mu sync.Mutex
	m  map[int]int
}

// NewSingleLockMap 预分配容量，避免运行时扩容干扰测试
func NewSingleLockMap(capacity int) *SingleLockMap {
	return &SingleLockMap{
		m: make(map[int]int, capacity),
	}
}

func (s *SingleLockMap) Set(key, val int) {
	s.mu.Lock()
	s.m[key] = val
	s.mu.Unlock()
}

// ==================== 方案2：25分片map + 25把独立锁 ====================
const ShardCount = 25 // 分片数量

type ShardedMap struct {
	shards []*shard
}

type shard struct {
	mu sync.Mutex
	m  map[int]int
}

// NewShardedMap 总容量均分到每个分片
func NewShardedMap(capacity int) *ShardedMap {
	perShard := capacity / ShardCount
	if perShard < 1 {
		perShard = 1
	}

	shards := make([]*shard, ShardCount)
	for i := 0; i < ShardCount; i++ {
		shards[i] = &shard{
			m: make(map[int]int, perShard),
		}
	}
	return &ShardedMap{shards: shards}
}

// 哈希分组：对key取模映射到对应分片（int类型key最简单高效的哈希方式）
func (s *ShardedMap) getShardIndex(key int) int {
	return key % ShardCount
}

func (s *ShardedMap) Set(key, val int) {
	idx := s.getShardIndex(key)
	sh := s.shards[idx]

	sh.mu.Lock()
	sh.m[key] = val
	sh.mu.Unlock()
}
