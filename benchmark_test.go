package cache

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

var message = "haha"

func BenchmarkTimer_Add(b *testing.B) {
	now := time.Now().UnixNano()
	timer := newTimer(time.Second, time.Now().UnixNano(), nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		timer.Add(fmt.Sprintf("key-%d", i), now)
		now += 1000000000
	}
}

func BenchmarkTimer_Add_Concurrent(b *testing.B) {
	now := time.Now().UnixNano()
	timer := newTimer(time.Second, time.Now().UnixNano(), nil)

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()
		for pb.Next() {
			timer.Add(fmt.Sprintf("key-%d", rand.Intn(b.N)), now)
			now += 1000000000
		}
	})
}

func BenchmarkWriteToCacheWith1Shard(b *testing.B) {
	m := "haha"
	cache := NewCache(600, time.Second)
	defer cache.StopCleanExpired()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), m)
	}
}

func BenchmarkWriteToCacheWithMultiShard(b *testing.B) {
	m := "haha"
	cache := NewCache(100000, time.Second)
	defer cache.StopCleanExpired()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), m)
	}
}

func BenchmarkWriteToCacheWith1ShardAndExp(b *testing.B) {
	m := "haha"
	cache := NewCache(600, time.Second)
	defer cache.StopCleanExpired()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.SetEx(fmt.Sprintf("key-%d", i), m, i)
	}
}

func BenchmarkWriteToCacheWithMultiShardAndExp(b *testing.B) {
	m := "haha"
	cache := NewCache(100000, time.Second)
	defer cache.StopCleanExpired()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.SetEx(fmt.Sprintf("key-%d", i), m, i)
	}
}

func BenchmarkWriteToCache(b *testing.B) {
	for _, count := range []int{10000, 100000, 500000} {
		b.Run(fmt.Sprintf("%d-scale", count), func(b *testing.B) {
			writeToCache(b, uint32(count))
		})
	}
}

func BenchmarkWriteToCacheWithExp(b *testing.B) {
	for _, count := range []int{10000, 100000, 500000} {
		b.Run(fmt.Sprintf("%d-scale", count), func(b *testing.B) {
			writeToCacheWithExp(b, uint32(count))
		})
	}
}

func BenchmarkReadFromCache(b *testing.B) {
	for _, count := range []int{600, 10000, 100000, 500000} {
		b.Run(fmt.Sprintf("%d-scale", count), func(b *testing.B) {
			readFromCache(b, uint32(count))
		})
	}
}

func BenchmarkReadFromCacheNonExistentKeys(b *testing.B) {
	for _, count := range []int{600, 10000, 100000, 500000} {
		b.Run(fmt.Sprintf("%d-scale", count), func(b *testing.B) {
			readFromCacheNonExistentKeys(b, uint32(count))
		})
	}
}

func writeToCache(b *testing.B, keyCountCale uint32) {
	cache := NewCache(keyCountCale, time.Second)
	defer cache.StopCleanExpired()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		id := rand.Int()
		counter := 0

		b.ReportAllocs()
		for pb.Next() {
			cache.Set(fmt.Sprintf("key-%d-%d", id, counter), message)
			counter = counter + 1
		}
	})
}

func writeToCacheWithExp(b *testing.B, keyCountCale uint32) {
	cache := NewCache(keyCountCale, time.Second)
	defer cache.StopCleanExpired()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		id := rand.Int()
		counter := 0

		b.ReportAllocs()
		for pb.Next() {
			cache.SetEx(fmt.Sprintf("key-%d-%d", id, counter), message, counter)
			counter = counter + 1
		}
	})
}

func readFromCache(b *testing.B, keyCountCale uint32) {
	cache := NewCache(keyCountCale, time.Second)
	defer cache.StopCleanExpired()

	for i := 0; i < b.N; i++ {
		cache.Set(strconv.Itoa(i), message)
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(rand.Intn(b.N)))
		}
	})
}

func readFromCacheNonExistentKeys(b *testing.B, keyCountCale uint32) {
	cache := NewCache(keyCountCale, time.Second)
	defer cache.StopCleanExpired()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(rand.Intn(b.N)))
		}
	})
}
