package cache

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

var _defSlicePool = newSlicePool(100, 100)

type slicePool struct {
	pool []*slice
	mu sync.Mutex
	cap int
	sliceCap int
}

func newSlicePool(poolCap, sliceCap int) *slicePool {
	return &slicePool{
		pool:     make([]*slice, 0, poolCap),
		cap:      poolCap,
		sliceCap: sliceCap,
	}
}

func (sp *slicePool) Get() *slice {
	sp.mu.Lock()

	idle := len(sp.pool)
	if idle == 0 {
		sp.mu.Unlock()
		return newSlice(sp.sliceCap)
	}

	index := idle - 1
	source := sp.pool[index]
	sp.pool[index] = nil
	sp.pool = sp.pool[:index]
	sp.mu.Unlock()
	return source
}

func (sp *slicePool) Put(s *slice) {
	sp.mu.Lock()
	if sp.cap == len(sp.pool) {
		sp.mu.Unlock()
		return
	}
	s.idx = 0
	sp.pool = append(sp.pool, s)
	sp.mu.Unlock()
}

type slice struct {
	idx  int32
	cap  int32
	keys []string
}

func newSlice(cap int) *slice {
	return &slice{
		idx:  0,
		cap:  int32(cap),
		keys: make([]string, cap),
	}
}

func (s *slice) Append(str string) bool {
	pos := atomic.AddInt32(&s.idx, 1)
	if pos > s.cap || pos < 0 {
		return false
	}
	s.keys[pos - 1] = str
	return true
}

func (s *slice) Full() {
	atomic.AddInt32(&s.idx, s.cap + 1)
}

type bucket struct {
	slicePool *slicePool
	full []*slice
	current unsafe.Pointer
	mu sync.Mutex
}

func newBucket(pool *slicePool) *bucket {
	return &bucket{
		slicePool: pool,
		current:   unsafe.Pointer(newSlice(100)),
	}
}

// Append之间存在并发调用
func (b *bucket) Append(str string) bool {
	oldP := atomic.LoadPointer(&b.current)
	//if oldP == nil {
	//   return false
	//}

	oldS := (*slice)(oldP)
	if oldS.Append(str) {
		return true
	}

	// 写满时，重新获取一个slice
	for {
		s := b.slicePool.Get()
		if atomic.CompareAndSwapPointer(
			&b.current,
			oldP,
			unsafe.Pointer(s),
		) { // 替换成功，将旧的放入full
			b.moveToFull(oldS)
		} else { // 其它线程重新加载
			b.slicePool.Put(s)
			s = (*slice)(atomic.LoadPointer(&b.current))
			//if s == nil {
			//    return false
			//}
		}
		if s.Append(str) {
			return true
		}
		// 仍然未写入(理论上存在)
		oldP = unsafe.Pointer(s)
		oldS = s
	}
}

func (b *bucket) StopReceiving() {
	s := (*slice)(b.current)
	b.full = append(b.full, s)
	b.current = nil
}

func (b *bucket) reset() {
	b.current = unsafe.Pointer(b.full[0])
	b.full[0] = nil
	for i := 1; i < len(b.full); i++ {
		b.slicePool.Put(b.full[i])
		b.full[i] = nil
	}
	b.full = b.full[:0]
}

func (b *bucket) moveToFull(s *slice) {
	b.mu.Lock()
	b.full = append(b.full, s)
	b.mu.Unlock()
}