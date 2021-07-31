package cache

import (
	"bufio"
	"compress/zlib"
	"io"
	"time"
)

type Cache struct {
	cache   []*shared
	timer   *timer
	mask    uint32
	indexFn func(str string, mask uint32) uint32
	stop    chan struct{}
}

func NewCache(keyCountScale uint32, cleanInterval time.Duration) *Cache {
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
		stop:  make(chan struct{}),
	}
	if n == 1 {
		out.indexFn = func(str string, mask uint32) uint32 {
			return 0
		}
	} else {
		out.indexFn = func(str string, mask uint32) uint32 {
			return fnv32(str) & mask
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
		err      error
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

func (c *Cache) SaveBaseType(w io.Writer) {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	zw, _ := zlib.NewWriterLevel(bw, zlib.BestSpeed)
	defer zw.Close()

	for _, v := range c.cache {
		v.SaveBaseType(zw)
	}
}

func (c *Cache) LoadBaseType(r io.Reader) {
	br := bufio.NewReader(r)

	zr, _ := zlib.NewReader(br)
	defer zr.Close()

	now := time.Now().UnixNano()
	for {
		kv := &kvItem{}
		if !kv.InitMetaFromReader(zr) {
			return
		}

		expAt := kv.GetExpireAt()
		if expAt >= 0 && expAt < now {
			kv.DiscardData(zr)
		} else {
			key, value, err := kv.ResolveKvFromReader(zr)
			if err != nil {
				return
			}

			if key != "" && value != nil {
				if expAt < 0 {
					c.Set(key, value)
				} else {
					c.timer.Add(key, expAt)
					c.cache[c.indexFn(key, c.mask)].Set(key, value, expAt)
				}
			}
		}
	}
}

func (c *Cache) runCleanExpired() {
	go func() {
		c.timer.Run(c.stop)
	}()
}

const _prime32 = uint32(16777619)

func fnv32(str string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(str); i++ {
		hash *= _prime32
		hash ^= uint32(str[i])
	}
	return hash
}

func power2(n uint32) uint32 {
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
