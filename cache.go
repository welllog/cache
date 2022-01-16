package cache

import (
	"bufio"
	"compress/zlib"
	"io"
	"time"
)

type Cache struct {
	s sharedSet
}

func NewCache(sharedNum, sharedCap int) *Cache {
	return &Cache{
		s: newCache(sharedNum, sharedCap),
	}
}

func NewCacheWithGC(sharedNum, sharedCap int, gcInterval time.Duration) *Cache {
	return &Cache{
		s: newCacheTimer(sharedNum, sharedCap, gcInterval),
	}
}

func (c *Cache) Get(key string) (interface{}, error) {
	value, ok := c.s.Get(c.s.Index(key), key)
	if !ok {
		return nil, ErrNil
	}
	return value, nil
}

func (c *Cache) Set(key string, value interface{}) {
	c.s.Set(c.s.Index(key), key, value)
}

func (c *Cache) SetEx(key string, value interface{}, ttl time.Duration) {
	if ttl < 0 {
		c.Set(key, value)
		return
	}
	expAt := time.Now().UnixNano() + int64(ttl)
	c.s.SetEx(c.s.Index(key), key, value, expAt)
}

func (c *Cache) Del(key string) {
	c.s.Del(c.s.Index(key), key)
}

func (c *Cache) LoadWithEx(key string, fn LoadFunc, ttl time.Duration) (interface{}, error) {
	return c.load(key, fn, ttl)
}

func (c *Cache) LoadAsyncWithEx(key string, fn LoadFunc, ttl time.Duration) (interface{}, error) {
	return c.loadAsync(key, fn, ttl)
}

func (c *Cache) Load(key string, fn LoadFunc) (interface{}, error) {
	return c.load(key, fn, -1)
}

func (c *Cache) Scan(handle func(key string, value interface{}, expAt int64)) {
	c.s.Scan(handle)
}

func (c *Cache) SaveBaseType(w io.Writer) {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	zw, _ := zlib.NewWriterLevel(bw, zlib.BestSpeed)
	defer zw.Close()

	now := time.Now().UnixNano()
	c.s.Scan(func(key string, value interface{}, expAt int64) {
		if expAt > now || expAt < 0 {
			kv := &kvItem{}
			if kv.Build(key, value, expAt) {
				kv.SaveTo(zw)
			}
		}
	})
}

func (c *Cache) LoadBaseType(r io.Reader) error {
	br := bufio.NewReader(r)

	zr, err := zlib.NewReader(br)
	if err != nil {
		return err
	}
	defer zr.Close()

	now := time.Now().UnixNano()
	for {
		kv := &kvItem{}
		if !kv.InitMetaFromReader(zr) {
			return nil
		}

		expAt := kv.GetExpireAt()
		if expAt >= 0 && expAt < now {
			kv.DiscardData(zr)
		} else {
			key, value, err := kv.ResolveKvFromReader(zr)
			if err != nil {
				return err
			}

			if key != "" && value != nil {
				if expAt < 0 {
					c.s.Set(c.s.Index(key), key, value)
				} else {
					c.s.SetEx(c.s.Index(key), key, value, expAt)
				}
			}
		}
	}
}

func (c *Cache) load(key string, fn LoadFunc, ttl time.Duration) (interface{}, error) {
	i := c.s.Index(key)
	value, ok := c.s.Get(i, key)
	if ok {
		return value, nil
	}
	var (
		err        error
		concurrent bool
	)
	value, err, concurrent = c.s.Load(i, key, fn)
	if err != nil {
		return nil, err
	}
	if !concurrent {
		if ttl < 0 {
			c.s.Set(i, key, value)
		} else {
			c.s.SetEx(i, key, value, time.Now().UnixNano()+int64(ttl))
		}
	}
	return value, nil
}

func (c *Cache) loadAsync(key string, fn LoadFunc, ttl time.Duration) (interface{}, error) {
	i := c.s.Index(key)
	value, expAt, ok := c.s.GetIgnoreExp(i, key)
	if ok {
		if expAt >= 0 && expAt < time.Now().UnixNano() { // 过期异步加载
			go func(k string, e time.Duration, i uint32, f func() (interface{}, error)) {
				v, err, concurrent := c.s.Load(i, k, f)
				if err != nil {
					return
				}
				if concurrent {
					return
				}
				if e < 0 {
					c.s.Set(i, k, v)
				} else {
					c.s.SetEx(i, k, v, time.Now().UnixNano()+int64(e))
				}
			}(key, ttl, i, fn)
		}
		return value, nil
	}
	var (
		err        error
		concurrent bool
	)
	// 不存在同步加载
	value, err, concurrent = c.s.Load(i, key, fn)
	if err != nil {
		return nil, err
	}
	if !concurrent {
		if ttl < 0 {
			c.s.Set(i, key, value)
		} else {
			c.s.SetEx(i, key, value, time.Now().UnixNano()+int64(ttl))
		}
	}
	return value, nil
}
