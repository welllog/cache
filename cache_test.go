package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCache(2000, time.Second)
	if cache.mask != 3 {
		t.Error("mask error")
	}
	cache.Set("t1", 123)
	val, err := cache.Get("t1")
	if err != nil {
		t.Error(err)
	}
	if val.(int) != 123 {
		t.Error("value error")
	}
}

func TestSetEx(t *testing.T) {
	cache := NewCache(300, time.Millisecond)
	cache.SetEx("test", 123, 2)
	time.Sleep(time.Second)
	fmt.Println(cache.cache[0].entries["test"])
	time.Sleep(2 * time.Second)
	fmt.Println(cache.cache[0].entries["test"])
}
