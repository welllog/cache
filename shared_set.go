package cache

import (
	"time"
)

var (
	_ sharedSet = (*cache)(nil)
	_ sharedSet = (*cacheTimer)(nil)
)

type sharedSet interface {
	Index(key string) uint32
	Get(index uint32, key string) (interface{}, bool)
	GetIgnoreExp(index uint32, key string) (interface{}, int64, bool)
	Set(index uint32, key string, value interface{})
	SetEx(index uint32, key string, value interface{}, expAt int64)
	Del(index uint32, key string)
	Scan(handle func(key string, value interface{}, expAt int64))
	Load(index uint32, key string, fn LoadFunc) (interface{}, error, bool)
}

type cache struct {
	indexFn func(str string, mask uint32) uint32
	sharers []*shared
	mask    uint32
}

type cacheTimer struct {
	*cache
	stop   chan struct{}
	timer  *timer
	groups [][]string
}

func newCache(sharedNum, sharedCap int) *cache {
	if sharedNum <= 1 {
		return &cache{
			indexFn: func(str string, mask uint32) uint32 {
				return 0
			},
			sharers: []*shared{newShared(sharedCap)},
		}
	}

	num := power2(uint32(sharedNum))
	sharers := make([]*shared, num)
	for i := 0; i < int(num); i++ {
		sharers[i] = newShared(sharedCap)
	}
	return &cache{
		indexFn: func(str string, mask uint32) uint32 {
			return fnv32(str) & mask
		},
		sharers: sharers,
		mask:    num - 1,
	}
}

func newCacheTimer(sharedNum, sharedCap int, cleanInterval time.Duration) *cacheTimer {
	c := newCache(sharedNum, sharedCap)
	ct := &cacheTimer{
		cache: c,
		stop:  make(chan struct{}),
	}

	realSharedNum := len(c.sharers)
	if realSharedNum > 1 {
		ct.groups = make([][]string, realSharedNum)
		for i := 0; i < realSharedNum; i++ {
			ct.groups[i] = make([]string, 0, 100)
		}
	}

	ct.timer = newTimer(cleanInterval, time.Now().UnixNano(), ct.CleanExpiredKeys)
	go func() {
		ct.timer.Run(ct.stop)
	}()

	return ct
}

func (c *cache) Index(key string) uint32 {
	return c.indexFn(key, c.mask)
}

func (c *cache) Get(index uint32, key string) (interface{}, bool) {
	return c.sharers[index].Get(key)
}

func (c *cache) GetIgnoreExp(index uint32, key string) (interface{}, int64, bool) {
	return c.sharers[index].GetIgnoreExp(key)
}

func (c *cache) Set(index uint32, key string, value interface{}) {
	c.sharers[index].Set(key, value, -1)
}

func (c *cache) SetEx(index uint32, key string, value interface{}, expAt int64) {
	c.sharers[index].Set(key, value, expAt)
}

func (c *cache) Del(index uint32, key string) {
	c.sharers[index].Del(key)
}

func (c *cache) Scan(handle func(key string, value interface{}, expAt int64)) {
	for _, s := range c.sharers {
		s.Scan(handle)
	}
}

func (c *cache) Load(index uint32, key string, fn LoadFunc) (interface{}, error, bool) {
	return c.sharers[index].Load(key, fn)
}

func (ct *cacheTimer) SetEx(index uint32, key string, value interface{}, expAt int64) {
	ct.sharers[index].Set(key, value, expAt)
	ct.timer.Add(key, expAt)
}

func (ct *cacheTimer) CleanExpiredKeys(unixNano int64, keys []string) {
	if ct.mask == 0 {
		ct.sharers[0].DelBefore(time.Now().UnixNano(), keys...)
		return
	}

	keysGroup := make(map[uint32][]string, len(ct.sharers))
	for _, key := range keys {
		index := ct.indexFn(key, ct.mask)
		g, ok := keysGroup[index]
		if !ok {
			groupL := len(ct.groups)
			g = ct.groups[groupL-1]
			ct.groups = ct.groups[:groupL-1]
		}
		keysGroup[index] = append(g, key)
	}

	now := time.Now().UnixNano()
	for i, group := range keysGroup {
		ct.sharers[i].DelBefore(now, group...)
		ct.groups = append(ct.groups, group[:0])
	}
}
