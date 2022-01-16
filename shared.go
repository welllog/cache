package cache

import (
	"sync"
	"time"
)

type shared struct {
	entries map[string]*entry
	//count   int
	//delCalled int
	mu     sync.RWMutex
	loader group
}

type entry struct {
	value interface{}
	expAt int64
}

func newShared(cap int) *shared {
	return &shared{
		entries: make(map[string]*entry, cap),
	}
}

func (s *shared) Get(key string) (interface{}, bool) {
	var (
		val   interface{}
		expAt int64
	)

	s.mu.RLock()

	r, ok := s.entries[key]
	if !ok {
		s.mu.RUnlock()
		return nil, false
	}

	val = r.value
	expAt = r.expAt
	s.mu.RUnlock()

	if expAt < 0 {
		return val, true
	}

	now := time.Now().UnixNano()
	if expAt > now {
		return val, true
	}

	s.mu.Lock()
	s.delBefore(key, expAt)
	s.mu.Unlock()

	return nil, false
}

func (s *shared) GetIgnoreExp(key string) (interface{}, int64, bool) {
	s.mu.RLock()
	r, ok := s.entries[key]
	if !ok {
		s.mu.RUnlock()
		return nil, 0, false
	}
	val, expAt := r.value, r.expAt
	s.mu.RUnlock()
	return val, expAt, true
}

func (s *shared) Set(key string, value interface{}, expAt int64) {
	s.mu.Lock()

	item, ok := s.entries[key]
	if ok {
		item.value = value
		item.expAt = expAt
	} else {
		s.entries[key] = &entry{
			value: value,
			expAt: expAt,
		}
		//s.count++
	}

	s.mu.Unlock()
}

func (s *shared) Del(key string) {
	s.mu.Lock()
	s.del(key)
	s.mu.Unlock()
}

func (s *shared) DelBefore(expAt int64, keys ...string) {
	s.mu.Lock()
	for _, key := range keys {
		s.delBefore(key, expAt)
	}
	s.mu.Unlock()
}

func (s *shared) Load(key string, fn LoadFunc) (interface{}, error, bool) {
	return s.loader.Do(key, fn)
}

func (s *shared) Scan(handle func(key string, value interface{}, expAt int64)) {
	s.mu.RLock()
	for k, v := range s.entries {
		handle(k, v.value, v.expAt)
	}
	s.mu.RUnlock()
}

func (s *shared) delBefore(key string, expAt int64) {
	val, ok := s.entries[key]
	if ok && val.expAt >= 0 && val.expAt <= expAt {
		s.del(key)
	}
}

func (s *shared) del(key string) {
	delete(s.entries, key)
	//s.delCalled++
	//s.count--
}
