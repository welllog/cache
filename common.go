package cache

import (
	"errors"
	"hash/maphash"
)

const (
	_avgKeys    = 600     // 分片初始容量
	_maxShareds = 1 << 10 // 最大分片数量，必须为2^x
)

var (
	_seed  = maphash.MakeSeed()
	ErrNil = errors.New("cache missing")
)

type LoadFunc func() (interface{}, error)

func ErrIsNotFound(err error) bool {
	return errors.Is(err, ErrNil)
}
