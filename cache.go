package cache

import (
	"hash/maphash"
	"time"
)

type Cache struct {
	cache   []*shared
	timer   *timer
	mask    uint64
	indexFn func(str string, mask uint64) uint64
	stop    chan struct{}
}

func NewCache(keyCountScale uint64, cleanInterval time.Duration) *Cache {
	n := keyCountScale / _avgKeys
	// 将n设为2^x
	n = power2(n)
	cache := make([]*shared, n)
	for i := range cache {
		cache[i] = newShared(_avgKeys)
	}

	out := &Cache{
		cache: cache,
		mask:  n - 1,
		stop: make(chan struct{}),
	}
	if n == 1 {
		out.indexFn = func(str string, mask uint64) uint64 {
			return 0
		}
	} else {
		out.indexFn = func(str string, mask uint64) uint64 {
			return hashNum(str) & mask
		}
	}
	out.timer = newTimer(cleanInterval, time.Now().UnixNano(), out)
	out.runCleanExpired()
	return out
}

func (c *Cache) Get(key string) (interface{}, error) {
	val, ok := c.cache[c.indexFn(key, c.mask)].Get(key)
	if !ok {
		return nil, ErrNil
	}
	return val, nil
}

func (c *Cache) GetOrLoadWithEx(key string, fn LoadFunc, exp int) (interface{}, error) {
	return c.getOrLoad(key, fn, exp)
}

func (c *Cache) GetOrLoad(key string, fn LoadFunc) (interface{}, error) {
	return c.getOrLoad(key, fn, -1)
}

func (c *Cache) getOrLoad(key string, fn LoadFunc, exp int) (interface{}, error) {
	i := c.indexFn(key, c.mask)
	val, ok := c.cache[i].Get(key)
	if ok {
		return val, nil
	}
	var (
		err error
		isConcur bool
	)
	val, err, isConcur = c.cache[i].Load(key, fn)
	if err != nil {
		return nil, err
	}
	if !isConcur {
		var expAt int64
		if exp < 0 {
			expAt = -1
		} else {
			expAt = time.Now().UnixNano() + sToNs(exp)
			c.timer.Add(key, expAt)
		}
		c.cache[i].Set(key, fn, expAt)
	}
	return val, nil
}

func (c *Cache) Set(key string, value interface{}) {
	c.cache[c.indexFn(key, c.mask)].Set(key, value, -1)
}

func (c *Cache) SetEx(key string, value interface{}, exp int) {
	expAt := time.Now().UnixNano() + sToNs(exp)
	c.timer.Add(key, expAt)
	c.cache[c.indexFn(key, c.mask)].Set(key, value, expAt)
}

func (c *Cache) Del(key string) {
	c.cache[c.indexFn(key, c.mask)].Del(key)
}

func (c *Cache) StopCleanExpired() {
	close(c.stop)
}

func (c *Cache) runCleanExpired() {
	go func() {
		c.timer.Run(c.stop)
	}()
}

func hashNum(str string) uint64 {
	h := new(maphash.Hash)
	h.SetSeed(_seed)
	_, _ = h.WriteString(str)
	return h.Sum64()
}

func power2(n uint64) uint64 {
	if n <= 1 {
		return 1
	}
	if n&(n-1) == 0 {
		return n
	}
	n = n - 1
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	if n > _maxShareds {
		return _maxShareds
	}
	return n + 1
}

func sToNs(s int) int64 {
	return int64(s) * 1000000000
}
