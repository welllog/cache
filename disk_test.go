package cache

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestKvItem_SaveTo(t *testing.T) {
	f, err := os.Create("c.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	now := time.Now().UnixNano()
	for k, v := range _kvs {
		item := &kvItem{}
		if item.Build(k, v, now) {
			item.SaveTo(f)
		}
	}
}

func TestKvItem_ResolveKvFromReader(t *testing.T) {
	f, err := os.Open("c.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("c.txt")
	defer f.Close()

	var num int
	for {
		item := &kvItem{}
		if !item.InitMetaFromReader(f) {
			break
		}

		fmt.Println(item.ResolveKvFromReader(f))
		num++
	}

	if num != len(_kvs) {
		t.Fatal("load kvs less")
	}
}
