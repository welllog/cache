package cache

import (
	"encoding/binary"
	"io"
	"math"
)

var DefaultOrder = binary.BigEndian

const (
	BYTE = uint8(iota + 65)
	STRING
	INT
	UINT
	BOOL
	FLOAT32
	FLOAT64
	INT8
	INT16
	INT32
	INT64
	UINT8
	UINT16
	UINT32
	UINT64
	UNEXPECT
)

type kvItem struct {
	expire    []byte // int64  => 8byte
	totalSize []byte // uint32 => 4byte
	keySize   []byte // uint16 => 2byte
	key       []byte
	id        byte
	value     []byte
}

func (k *kvItem) Build(key string, value interface{}, expAt int64) bool {
	k.key = []byte(key)
	ksize := len(k.key)
	if ksize > 65535 {
		return false
	}

	k.id, k.value = baseTypeValue(value)
	if k.id == UNEXPECT {
		return false
	}

	if len(k.value) > 524288000 {
		return false
	}

	k.expire = make([]byte, 8)
	DefaultOrder.PutUint64(k.expire, uint64(expAt))

	k.totalSize = make([]byte, 4)
	tsize := uint32(3 + ksize + len(k.value))
	DefaultOrder.PutUint32(k.totalSize, tsize)

	k.keySize = make([]byte, 2)
	DefaultOrder.PutUint16(k.keySize, uint16(ksize))

	return true
}

func (k *kvItem) SaveTo(w io.Writer) {
	_, _ = w.Write(k.expire)
	_, _ = w.Write(k.totalSize)
	_, _ = w.Write(k.keySize)
	_, _ = w.Write(k.key)
	_, _ = w.Write([]byte{k.id})
	_, _ = w.Write(k.value)
}

func (k *kvItem) InitMetaFromReader(r io.Reader) bool {
	k.expire = make([]byte, 8)
	_, err := io.ReadAtLeast(r, k.expire, 8)
	if err != nil {
		return false
	}

	k.totalSize = make([]byte, 4)
	_, err = io.ReadAtLeast(r, k.totalSize, 4)
	if err != nil {
		return false
	}
	return true
}

func (k *kvItem) GetExpireAt() int64 {
	return int64(DefaultOrder.Uint64(k.expire))
}

func (k *kvItem) DiscardData(r io.Reader) {
	_, _ = io.CopyN(Discard, r, int64(DefaultOrder.Uint32(k.totalSize)))
}

func (k *kvItem) ResolveKvFromReader(r io.Reader) (key string, value interface{}, err error) {
	k.keySize = make([]byte, 2)
	_, err = io.ReadAtLeast(r, k.keySize, 2)
	if err != nil {
		return
	}

	ksize := int(DefaultOrder.Uint16(k.keySize))
	k.key = make([]byte, ksize)
	_, err = io.ReadAtLeast(r, k.key, ksize)
	if err != nil {
		return
	}

	rest := int(DefaultOrder.Uint32(k.totalSize)) - ksize - 2
	payload := make([]byte, rest)
	_, err = io.ReadAtLeast(r, payload, rest)
	if err != nil {
		return
	}
	value = baseValueRestore(payload[0], payload[1:])
	return string(k.key), value, nil
}

func baseValueRestore(id byte, data []byte) interface{} {
	switch id {
	case BYTE:
		res := make([]byte, len(data))
		copy(res, data)
		return res

	case STRING:
		return string(data)

	case INT:
		return int(DefaultOrder.Uint64(data))

	case BOOL:
		return data[0] != 0

	case FLOAT32:
		return math.Float32frombits(DefaultOrder.Uint32(data))

	case FLOAT64:
		return math.Float64frombits(DefaultOrder.Uint64(data))

	case INT64:
		return int64(DefaultOrder.Uint64(data))

	case INT32:
		return int32(DefaultOrder.Uint32(data))

	case INT16:
		return int16(DefaultOrder.Uint16(data))

	case INT8:
		return int8(data[0])

	case UINT64:
		return DefaultOrder.Uint64(data)

	case UINT32:
		return DefaultOrder.Uint32(data)

	case UINT16:
		return DefaultOrder.Uint16(data)

	case UINT8:
		return data[0]

	case UINT:
		return uint(DefaultOrder.Uint64(data))

	default:
		return nil
	}
}

func baseTypeValue(value interface{}) (byte, []byte) {
	var id byte
	var data []byte
	switch v := value.(type) {
	case []byte:
		id = BYTE
		data = v
	case string:
		id = STRING
		data = []byte(v)
	case int:
		id = INT
		data = make([]byte, 8)
		DefaultOrder.PutUint64(data, uint64(v))
	case bool:
		id = BOOL
		data = make([]byte, 1)
		if v {
			data[0] = 1
		} else {
			data[0] = 0
		}
	case float32:
		id = FLOAT32
		data = make([]byte, 4)
		DefaultOrder.PutUint32(data, math.Float32bits(v))
	case float64:
		id = FLOAT64
		data = make([]byte, 8)
		DefaultOrder.PutUint64(data, math.Float64bits(v))
	case int64:
		id = INT64
		data = make([]byte, 8)
		DefaultOrder.PutUint64(data, uint64(v))
	case int32:
		id = INT32
		data = make([]byte, 4)
		DefaultOrder.PutUint32(data, uint32(v))
	case int16:
		id = INT16
		data = make([]byte, 2)
		DefaultOrder.PutUint16(data, uint16(v))
	case int8:
		id = INT8
		data = make([]byte, 1)
		data[0] = byte(v)
	case uint64:
		id = UINT64
		data = make([]byte, 8)
		DefaultOrder.PutUint64(data, v)
	case uint32:
		id = UINT32
		data = make([]byte, 4)
		DefaultOrder.PutUint32(data, v)
	case uint16:
		id = UINT16
		data = make([]byte, 2)
		DefaultOrder.PutUint16(data, v)
	case uint8:
		id = UINT8
		data = make([]byte, 1)
		data[0] = v
	case uint:
		id = UINT
		data = make([]byte, 8)
		DefaultOrder.PutUint64(data, uint64(v))
	default:
		return UNEXPECT, nil
	}
	return id, data
}

var Discard io.Writer = discard{}

type discard struct{}

func (discard) Write(p []byte) (int, error) {
	return len(p), nil
}
