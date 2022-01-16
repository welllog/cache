package cache

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestSlicePool(t *testing.T) {
	poolCap := 10
	pool := newSlicePool(poolCap, 2)

	var w sync.WaitGroup
	for i := 0; i < poolCap; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			_ = pool.Get()
		}()
	}
	w.Wait()

	fmt.Println(len(pool.pool))
	if len(pool.pool) > 0 {
		t.Fatal("pool should be empty")
	}

	sliceArr := make([]*slice, poolCap+2)
	for i := 0; i < poolCap+2; i++ {
		sliceArr[i] = pool.Get()
	}
	for _, s := range sliceArr {
		w.Add(1)
		go func(s *slice) {
			defer w.Done()
			pool.Put(s)
		}(s)
	}
	w.Wait()

	fmt.Println(len(pool.pool))
	if len(pool.pool) > pool.cap {
		t.Fatal("pool cap should be ", pool.cap)
	}
}

func TestSlice(t *testing.T) {
	sCap := 10
	s := newSlice(sCap)

	var w sync.WaitGroup
	for i := 0; i < sCap; i++ {
		w.Add(1)
		go func(n int) {
			defer w.Done()
			s.Append(strconv.Itoa(n))
		}(i)
	}

	w.Wait()

	for i := sCap; i < sCap+3; i++ {
		w.Add(1)
		go func(n int) {
			defer w.Done()
			s.Append(strconv.Itoa(n))
		}(i)
	}

	w.Wait()
	if s.Append("test") {
		t.Fatal("slice should be full")
	}

	keys := make([]string, 0, sCap)
	keys = s.ExportKeys(keys)
	keyMap := map[string]bool{}
	for _, key := range keys {
		keyMap[key] = true
	}
	fmt.Println(keys)

	for i := 0; i < sCap; i++ {
		if !keyMap[strconv.Itoa(i)] {
			t.Fatal("append key failed, key: ", i)
		}
	}

	if s.idx > 0 {
		t.Fatal("slice idx should be zero")
	}
}

func TestBucket(t *testing.T) {
	bucket := newBucket(newSlicePool(10, 5))

	n := 22
	var w sync.WaitGroup
	for i := 0; i < n; i++ {
		w.Add(1)
		go func(i int) {
			defer w.Done()
			bucket.Append(strconv.Itoa(i))
		}(i)
	}

	w.Wait()
	fmt.Println(len(bucket.full))

	keys := bucket.ExportKeys()
	keyMap := map[string]bool{}
	for _, key := range keys {
		keyMap[key] = true
	}
	fmt.Println(keys)
	for i := 0; i < n; i++ {
		if !keyMap[strconv.Itoa(i)] {
			t.Fatal("append key failed, key: ", i)
		}
	}

	if len(bucket.full) > 0 || bucket.current != nil {
		t.Fatal("bucket cache should be cleaned")
	}

	if len(bucket.ExportKeys()) > 0 {
		t.Fatal("bucket cache should be empty")
	}
}
