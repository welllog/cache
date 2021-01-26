package cache

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
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
		stop: make(chan struct{}),
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

func (c *Cache) SaveBaseType(w io.Writer) {
	nw := bufio.NewWriter(w)
	for _, v := range c.cache {
		v.SaveBaseType(nw)
	}
	_ = nw.Flush()
}

func (c *Cache) LoadBaseType(r io.Reader) {
	now := time.Now().UnixNano()
	scanner := bufio.NewScanner(r)
	// 每两行是一条完整记录
	var row int
	var skip bool
	var key string
	var expAt int64
	for scanner.Scan() {
		
		if (row & 1) == 0 { // 第一行
			b := scanner.Bytes()
			index := bytes.Index(b, []byte{_delim})
			expb := b[:index]
			expAt, _ = strconv.ParseInt(string(expb), 10, 64)
			if expAt >= 0 && expAt < now {
				row++
				skip = true
				continue
			}
			
			keyb := b[index + 1:]
			key = string(keyb)
		} else { // 第二行
			if !skip {
				b := scanner.Bytes()
				index := bytes.Index(b, []byte{_delim})
				idb := b[:index]
				valueb := b[index + 1:]
				value := baseValueRestore(idb[0], valueb)
				if value != nil {
					if expAt < 0 {
						c.Set(key, value)
					} else {
						c.timer.Add(key, expAt)
						c.cache[c.indexFn(key, c.mask)].Set(key, value, expAt)
					}
				}
			}
			skip = false
		}
		row++
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
