package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestTimer_Add(t *testing.T) {
	now := time.Now()
	timer := newTimer(time.Millisecond, now.UnixNano(), func(now int64, strings []string) {
		fmt.Println("time: ", now, ", keys: ", strings)
		fmt.Println("-----")
	})
	go func() {
		timer.Run(make(chan struct{}))
	}()

	timer.Add("t1", now.Add(time.Millisecond).UnixNano())
	timer.Add("t2", now.Add(500*time.Microsecond).UnixNano())
	timer.Add("t3", now.Add(31*time.Millisecond).UnixNano())
	timer.Add("t4", now.Add(32*time.Millisecond).UnixNano())
	timer.Add("t5", now.Add(33*time.Millisecond).UnixNano())
	timer.Add("t6", now.Add(41*time.Millisecond).UnixNano())
	timer.Add("t7", now.Add(49*time.Millisecond).UnixNano())
	time.Sleep(100 * time.Millisecond)
}
