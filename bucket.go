package cache

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const _EMPTY_STR = ""

var _defSlicePool = newSlicePool(100, 100)

type slicePool struct {
	pool     []*slice
	mu       sync.Mutex
	cap      int
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
	sp.pool = append(sp.pool, s)
	sp.mu.Unlock()
}

type slice struct {
	idx  int32
	keys []string
}

func newSlice(cap int) *slice {
	return &slice{
		idx:  0,
		keys: make([]string, cap),
	}
}

// Append 不能接受空字符串
func (s *slice) Append(str string) bool {
	pos := atomic.AddInt32(&s.idx, 1)
	if int(pos) > cap(s.keys) || pos < 0 {
		return false
	}
	s.keys[pos-1] = str
	return true
}

func (s *slice) GetCap() int {
	return cap(s.keys)
}

// ExportKeys 导出所有keys,并清理当前keys
func (s *slice) ExportKeys(keys []string) []string {
	for i, key := range s.keys {
		if key == _EMPTY_STR {
			break
		}
		keys = append(keys, key)
		s.keys[i] = _EMPTY_STR
	}
	s.idx = 0
	return keys
}

type bucket struct {
	slicePool *slicePool
	full      []*slice
	current   unsafe.Pointer
	mu        sync.Mutex
}

func newBucket(pool *slicePool) *bucket {
	return &bucket{
		slicePool: pool,
	}
}

// Append之间存在并发调用
func (b *bucket) Append(str string) {
	if str == _EMPTY_STR {
		return
	}

	oldP := atomic.LoadPointer(&b.current)
	if oldP == nil {
		first := b.slicePool.Get()
		oldP = unsafe.Pointer(first)
		if !atomic.CompareAndSwapPointer(&b.current, nil, oldP) {
			b.slicePool.Put(first)
			oldP = atomic.LoadPointer(&b.current)
		}
	}
	oldS := (*slice)(oldP)

	if oldS.Append(str) {
		return
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
		}
		if s.Append(str) {
			return
		}
		// 仍然未写入(理论上存在)
		oldP = unsafe.Pointer(s)
		oldS = s
	}
}

// ExportKeys 导出当前bucket key,并进行清理
func (b *bucket) ExportKeys() []string {
	if b.current != nil {
		s := (*slice)(b.current)
		b.full = append(b.full, s)
		b.current = nil
	}
	if len(b.full) == 0 {
		return nil
	}
	keys := make([]string, 0, len(b.full)*b.full[0].GetCap())
	for i := range b.full {
		keys = b.full[i].ExportKeys(keys)
		b.slicePool.Put(b.full[i])
		b.full[i] = nil
	}
	b.full = b.full[:0]
	return keys
}

func (b *bucket) moveToFull(s *slice) {
	b.mu.Lock()
	b.full = append(b.full, s)
	b.mu.Unlock()
}
