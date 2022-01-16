package cache

import (
	"errors"
)

const (
	_maxShareds = 1 << 10 // 最大分片数量，必须为2^x
	_prime32    = uint32(16777619)
)

var (
	ErrNil = errors.New("cache missing")
)

type LoadFunc func() (interface{}, error)

func ErrIsNotFound(err error) bool {
	return errors.Is(err, ErrNil)
}

func fnv32(str string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(str); i++ {
		hash *= _prime32
		hash ^= uint32(str[i])
	}
	return hash
}

func power2(n uint32) uint32 {
	if n <= 1 {
		return 1
	}
	if n&(n-1) == 0 {
		return n
	}
	n = n - 1
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	if n > _maxShareds {
		return _maxShareds
	}
	return n + 1
}
