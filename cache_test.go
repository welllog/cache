package cache

import (
	"fmt"
	"os"
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

func TestCache_SaveBaseType(t *testing.T) {
	cache := NewCache(3000, time.Millisecond)
	cache.Set("t1", "asdsadsa")
	cache.Set("t2", 12)
	cache.SetEx("t3", 12, 1)
	time.Sleep(time.Second)
	cache.SetEx("t4", true, 10)
	f, err := os.Create("c.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cache.SaveBaseType(f)
}

func TestCache_LoadBaseType(t *testing.T) {
	f, err := os.Open("c.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	
	cache := NewCache(3000, time.Millisecond)
	cache.LoadBaseType(f)
	
	for _, v := range cache.cache {
		for key, entry := range v.entries {
			fmt.Println(key, "-----", entry.value, "-----", entry.expAt)
		}
	}
}
