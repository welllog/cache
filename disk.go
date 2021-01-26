package cache

import (
    "fmt"
    "io"
    "strconv"
)

const _delim = ' '

const (
    BYTE = uint8(iota)
    STRING
    INT
    INT8
    INT16
    INT32
    INT64
    UINT
    UINT8
    UINT16
    UINT32
    UINT64
    FLOAT32
    FLOAT64
    BOOL
)

func flushBase2Disk(key string, value interface{}, expAt int64, w io.Writer) {
    id, data := baseTypeValue(value)
    if id != nil {  // 只保存基本类型的数据
        w.Write([]byte(strconv.FormatInt(expAt, 10)))
        w.Write([]byte{_delim})
        w.Write([]byte(key))
        w.Write([]byte(fmt.Sprintln()))
        w.Write(id)
        w.Write([]byte{_delim})
        w.Write(data)
        w.Write([]byte(fmt.Sprintln()))
    }
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
        res, _ := strconv.Atoi(string(data))
        return res

    case BOOL:
        res, _ := strconv.ParseBool(string(data))
        return res

    case FLOAT32:
        res, _ := strconv.ParseFloat(string(data), 32)
        return float32(res)

    case FLOAT64:
        res, _ := strconv.ParseFloat(string(data), 64)
        return res

    case UINT:
        res, _ := strconv.ParseUint(string(data), 10, 64)
        return uint(res)

    case INT64:
        res, _ := strconv.ParseInt(string(data), 10, 64)
        return res

    case INT32:
        res, _ := strconv.ParseInt(string(data), 10, 32)
        return int32(res)

    case UINT32:
        res, _ := strconv.ParseUint(string(data), 10, 32)
        return uint32(res)
        
    case UINT64:
        res, _ := strconv.ParseUint(string(data), 10, 64)
        return res

    case INT8:
        res, _ := strconv.ParseInt(string(data), 10, 8)
        return int8(res)

    case INT16:
        res, _ := strconv.ParseInt(string(data), 10, 16)
        return int16(res)

    case UINT8:
        res, _ := strconv.ParseUint(string(data), 10, 8)
        return uint8(res)
        
    case UINT16:
        res, _ := strconv.ParseUint(string(data), 10, 16)
        return uint16(res)

    default:
        return nil
    }
}

func baseTypeValue(value interface{}) ([]byte, []byte) {
    switch v := value.(type) {
    case []byte:
        return []byte{BYTE}, v
    case string:
        return []byte{STRING}, []byte(v)
    case int:
        return []byte{INT}, []byte(strconv.Itoa(v))
    case bool:
        return []byte{BOOL}, []byte(strconv.FormatBool(v))
    case float32:
        return []byte{FLOAT32}, []byte(strconv.FormatFloat(float64(v), 'x', -1, 64))
    case float64:
        return []byte{FLOAT64}, []byte(strconv.FormatFloat(v, 'x', -1, 64))
    case uint:
        return []byte{UINT}, []byte(strconv.FormatUint(uint64(v), 10))
    case int64:
        return []byte{INT64}, []byte(strconv.FormatInt(v, 10))
    case int32:
        return []byte{INT32}, []byte(strconv.FormatInt(int64(v), 10))
    case uint32:
        return []byte{UINT32}, []byte(strconv.FormatUint(uint64(v), 10))
    case uint64:
        return []byte{UINT64}, []byte(strconv.FormatUint(v, 10))
    case int8:
        return []byte{INT8}, []byte(strconv.FormatInt(int64(v), 10))
    case int16:
        return []byte{INT16}, []byte(strconv.FormatInt(int64(v), 10))
    case uint8:
        return []byte{UINT8}, []byte(strconv.FormatUint(uint64(v), 10))
    case uint16:
        return []byte{UINT16}, []byte(strconv.FormatUint(uint64(v), 10))
    default:
        return nil, nil
    }
}
