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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Add(fmt.Sprintf("key-%d", i), now)
		now += 1000000000
	}
}

func BenchmarkTimer_Add_Concurrent(b *testing.B) {
	now := time.Now().UnixNano()
	timer := newTimer(time.Second, time.Now().UnixNano(), nil)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			timer.Add(fmt.Sprintf("key-%d", rand.Int63()), now)
		}
	})
}

func BenchmarkWriteToCache(b *testing.B) {
	for _, num := range []int{1, 32, 64} {
		writeToCache(b, num)
		writeToCacheWithGC(b, num)
	}
}

func BenchmarkReadFromCache(b *testing.B) {
	for _, num := range []int{1, 32, 64} {
		readFromCache(b, num)
		readFromCacheWithGC(b, num)
	}
}

func writeToCache(b *testing.B, num int) {
	b.Run(fmt.Sprintf("%d-shared", num), func(b *testing.B) {
		cache := NewCache(num, 10000)
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Set(fmt.Sprintf("key-%d", rand.Int63()), 1)
			}
		})
	})
}

func writeToCacheWithGC(b *testing.B, num int) {
	b.Run(fmt.Sprintf("%d-shared-with-gc", num), func(b *testing.B) {
		cache := NewCacheWithGC(num, 10000, 50*time.Millisecond)
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.SetEx(fmt.Sprintf("key-%d", rand.Int63()), 1, 80*time.Millisecond)
			}
		})
	})
}

func readFromCache(b *testing.B, num int) {
	b.Run(fmt.Sprintf("%d-shared", num), func(b *testing.B) {
		cache := NewCache(num, 10000)

		for i := 0; i < b.N; i++ {
			cache.Set(strconv.Itoa(i), message)
		}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Get(strconv.Itoa(rand.Intn(b.N)))
			}
		})
	})
}

func readFromCacheWithGC(b *testing.B, num int) {
	b.Run(fmt.Sprintf("%d-shared-with-gc", num), func(b *testing.B) {
		cache := NewCacheWithGC(num, 10000, 50*time.Millisecond)

		for i := 0; i < b.N; i++ {
			cache.SetEx(strconv.Itoa(i), message, 80*time.Millisecond)
		}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Get(strconv.Itoa(rand.Intn(b.N)))
			}
		})
	})
}
