package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestShared_Get(t *testing.T) {
	s := newShared(10)
	var w sync.WaitGroup
	w.Add(3)
	go func() {
		defer w.Done()
		s.Set("t1", 1, -1)
	}()
	go func() {
		defer w.Done()
		s.Set("t2", 1, time.Now().UnixNano())
	}()
	go func() {
		defer w.Done()
		s.Set("t3", 1, time.Now().UnixNano())
	}()
	w.Wait()

	val1, ok1 := s.Get("t1")
	val2, ok2 := s.Get("t2")
	val3, expAt, ok3 := s.GetIgnoreExp("t3")
	if !ok1 {
		t.Fatal("t1 not found")
	}
	if ok2 {
		t.Fatal("t2 should be del")
	}
	if !ok3 {
		t.Fatal("t3 not found")
	}
	fmt.Println(val1, ok1)
	fmt.Println(val2, ok2)
	fmt.Println(val3, expAt, ok3)

	s.Scan(func(key string, value interface{}, expAt int64) {
		fmt.Println(key, "--", value, "--", expAt)
	})
}

func TestShared_DelBefore(t *testing.T) {
	s := newShared(10)
	now := time.Now()
	s.Set("t1", 1, now.Add(5*time.Millisecond).UnixNano())
	s.Set("t2", 1, -1)
	s.Set("t3", 1, now.Add(1*time.Millisecond).UnixNano())

	time.Sleep(2 * time.Millisecond)
	s.DelBefore(time.Now().UnixNano(), "t1", "t2", "t3")

	s.Scan(func(key string, value interface{}, expAt int64) {
		fmt.Println(key, "--", value, "--", expAt)
	})

	if _, _, ok := s.GetIgnoreExp("t1"); !ok {
		t.Fatal("t1 should exists")
	}

	if _, _, ok := s.GetIgnoreExp("t2"); !ok {
		t.Fatal("t2 should exists")
	}

	if _, _, ok := s.GetIgnoreExp("t3"); ok {
		t.Fatal("t3 should be del")
	}
}

func TestShared_Load(t *testing.T) {
	s := newShared(10)
	concurrent := 10
	ch := make(chan bool, concurrent+2)
	var w sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			_, _, ok := s.Load("t1", func() (interface{}, error) {
				time.Sleep(time.Microsecond)
				return 1, nil
			})
			ch <- ok
		}()
	}
	w.Add(1)
	go func() {
		defer w.Done()
		_, err, ok := s.Load("t1", func() (interface{}, error) {
			time.Sleep(time.Microsecond)
			panic("test panic")
		})
		ch <- ok
		fmt.Println(err)
	}()

	w.Wait()
	close(ch)

	var num int
	for ok := range ch {
		if !ok {
			num++
		}
	}
	fmt.Println(num)

	if num > 3 {
		t.Fatal("concurrent set shared")
	}
}
