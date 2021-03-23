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

var _kvs = map[string]interface{}{
	"t1": []byte("hello"),
	"t2": "world",
	"t3": true,
	"t4": false,
	"t5": -123456,
	"t6": int8(-128),
	"t7": int16(-129),
	"t8": int32(-130),
	"t9": int64(-131),
	"t10": uint8(127),
	"t11": uint16(128),
	"t12": uint32(129),
	"t13": uint64(130),
	"t14": uint(131),
	"t15": float32(11.11),
	"t16": float64(14.14),
}

func TestCache_SaveBaseType(t *testing.T) {
	cache := NewCache(10000000, time.Millisecond)
	for k, v := range _kvs {
		cache.Set(k, v)
	}
	//for i := 0; i < 10000000; i++  {
	//	cache.Set("k" + strconv.Itoa(i), i)
	//}
	cache.SetEx("t100", 12, 1)
	time.Sleep(time.Second)
	cache.SetEx("t101", true, 10)
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
	var i int
	
	for _, v := range cache.cache {
		for key, entry := range v.entries {
			i++
			if i & 1023 == 0 {
				return
			}
			fmt.Println(key, "-----", entry.value, "-----", entry.expAt)
		}
	}
}
