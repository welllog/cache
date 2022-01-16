package cache

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCacheWithGC(2, 50, time.Second)
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
	cache := NewCacheWithGC(2, 50, time.Millisecond)
	cache.SetEx("test", 123, 5*time.Millisecond)
	time.Sleep(time.Millisecond)
	fmt.Println(cache.Get("test"))
	time.Sleep(4 * time.Millisecond)
	fmt.Println(cache.Get("test"))
}

var _kvs = map[string]interface{}{
	"t1":  []byte("hello"),
	"t2":  "world",
	"t3":  true,
	"t4":  false,
	"t5":  -123456,
	"t6":  int8(-128),
	"t7":  int16(-129),
	"t8":  int32(-130),
	"t9":  int64(-131),
	"t10": uint8(127),
	"t11": uint16(128),
	"t12": uint32(129),
	"t13": uint64(130),
	"t14": uint(131),
	"t15": float32(11.11),
	"t16": float64(14.14),
}

func TestCache_SaveBaseType(t *testing.T) {
	cache := NewCache(2, 50)
	for k, v := range _kvs {
		cache.Set(k, v)
	}
	cache.SetEx("t100", 12, time.Second)
	time.Sleep(time.Second)
	cache.SetEx("t101", true, 10*time.Second)
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
	defer os.Remove("c.txt")
	defer f.Close()

	cache := NewCache(2, 50)
	_ = cache.LoadBaseType(f)

	cache.Scan(func(key string, value interface{}, expAt int64) {
		fmt.Println(key, "-----", value, "-----", expAt)
	})
}
