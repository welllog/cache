package cache

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	_defSlotNum = 1 << 5      // 时间轮每一层槽数
	_overflowTimerTickMultiple = 8 // 下一层时间轮的间隔时间相当于当前轮间隔时间的倍数
)

// 高层时间轮避免向下级联
type timer struct {
	curTime int64
	tick int64
	interval int64
	slotNum int
	slotMask int
	curSlot int
	mu sync.RWMutex
	slots []*bucket
	overflowSet int32
	overflowTimer unsafe.Pointer
	cache *Cache
}

// tick 间隔时间，单位秒
func newTimer(tick time.Duration, now int64, cache *Cache) *timer {
	buckets := make([]*bucket, _defSlotNum)
	for i := range buckets {
		buckets[i] = newBucket(_defSlicePool)
	}
	return &timer{
		curTime: now,
		tick: int64(tick),
		interval: (int64(_defSlotNum) - 1) * int64(tick),
		slotNum: _defSlotNum,
		slotMask: _defSlotNum - 1,
		slots: buckets,
		cache: cache,
	}
}

func (t *timer) Add(key string, expAt int64) {
	t.mu.RLock()
	delay := expAt - t.curTime
	if delay <= t.interval { // 加入当前的timer
		var moveSlot int
		if delay <= t.tick {
			moveSlot = 1
		} else {
			moveSlot = int(truncate(delay, t.tick) / t.tick)
		}
		slot := (t.curSlot + moveSlot) & t.slotMask
		t.slots[slot].Append(key)
		t.mu.RUnlock()

		return
	}
	t.mu.RUnlock()

	overflowWheel := atomic.LoadPointer(&t.overflowTimer)
	if overflowWheel == nil {
		if atomic.CompareAndSwapInt32(&t.overflowSet, 0, 1) { // 设置成功
			t.mu.RLock()
			overflowWheel = unsafe.Pointer(newTimer(time.Duration(t.tick) * _overflowTimerTickMultiple, t.curTime, t.cache))
			atomic.StorePointer(&t.overflowTimer, overflowWheel)
			t.mu.RUnlock()
		} else {
			for {
				time.Sleep(50 * time.Millisecond)
				overflowWheel = atomic.LoadPointer(&t.overflowTimer)
				if overflowWheel != nil {
					break
				}
			}
		}
	}
	(*timer)(overflowWheel).Add(key, expAt)
}

func (t *timer) advanceClock(now int64) {
	var curSlot int
	var nextTimer *timer
	var interval int64

	t.mu.Lock()
	t.curSlot++
	if t.curSlot > t.slotMask {
		t.curSlot = 0
	}
	t.curTime = now
	curSlot = t.curSlot

	if t.overflowTimer != nil {
		nextTimer = (*timer)(t.overflowTimer)
		interval = now - nextTimer.curTime
	}
	t.mu.Unlock()

	// ticker获取的时间不一定是精准的t.tick的_overflowTimerTickMultiple倍数
	if interval > ((int64(_overflowTimerTickMultiple) - 1) * t.tick + t.tick / 2) {
		nextTimer.advanceClock(now)
	}

	t.scan(curSlot)
}

func (t *timer) scan(slot int) {
	t.slots[slot].StopReceiving()
	cleanKeys := make(map[uint32][]string, t.cache.mask + 1)
	for _, s := range t.slots[slot].full {
		for index, key := range s.keys {
			if key != "" {
				s.keys[index] = ""
				cacheIndex := t.cache.indexFn(key, t.cache.mask)
				cleanKeys[cacheIndex] = append(cleanKeys[cacheIndex], key)
			}
		}
	}

	t.slots[slot].reset()

	for i := range cleanKeys {
		now := time.Now().UnixNano()
		t.cache.cache[i].mu.Lock()
		for _, key := range cleanKeys[i] {
			t.cache.cache[i].delBefore(key, now)
		}
		t.cache.cache[i].mu.Unlock()
	}
}

func (t *timer) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Duration(t.tick))
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			t.advanceClock(now.UnixNano())
		}
	}
}

func truncate(x, m int64) int64 {
	if m <= 0 {
		return x
	}
	return x + x%m
}