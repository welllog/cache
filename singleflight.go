package cache

import (
	"errors"
	"sync"
)

var _errLoad = errors.New("load func panic")

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type group struct {
	mu sync.Mutex // protects m
	m  map[string]*call
}

func (g *group) Do(key string, fn func() (interface{}, error)) (value interface{}, err error, shared bool) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err, true
	}

	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	nf := func() (val interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				val, err = nil, _errLoad
			}
		}()
		val, err = fn()
		return
	}

	c.val, c.err = nf()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err, false
}
