package utils

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"unsafe"
)

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

func Max[T numeric](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Min[T numeric](a, b T) T {
	if a > b {
		return b
	}
	return a
}

func ConvertToString(s any) (string, error) {
	if s == nil {
		return "", nil
	}
	rv := reflect.ValueOf(s)

	switch rv.Kind() {
	case reflect.String:
		return s.(string), nil
	case reflect.Bool:
		if rv.Bool() {
			return "1", nil
		}
		return "", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", rv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", rv.Float()), nil
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return string(rv.Bytes()), nil
		}
	case reflect.Struct:
		var f fmt.Stringer
		if rv.CanConvert(reflect.TypeOf(f)) {
			return rv.MethodByName("String").Call(nil)[0].String(), nil
		}
	}
	return "", fmt.Errorf("can not convert %v to string\n", s)
}

func ConvertToLong(s any) (int64, error) {
	if s == nil {
		return 0, nil
	}
	rv := reflect.ValueOf(s)

	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(rv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return int64(rv.Float()), nil
	case reflect.String:
		return strconv.ParseInt(rv.String(), 10, 64)
	case reflect.Slice:
		return If[int64](rv.Len() > 0, 1, 0), nil
	}

	return 0, fmt.Errorf("can not convert %v to long\n", s)
}

func ConvertToFloat(s any) (float64, error) {
	if s == nil {
		return 0, nil
	}

	rv := reflect.ValueOf(s)

	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return rv.Float(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), nil
	case reflect.Bool:
		if rv.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.String:
		return strconv.ParseFloat(rv.String(), 64)
	case reflect.Slice:
		return If(rv.Len() > 0, 1.0, 0.0), nil
	}

	return 0, fmt.Errorf("can not convert %v to float64\n", s)
}

func IsLittleEndian() bool {
	var value int32 = 1
	pointer := unsafe.Pointer(&value)
	pb := (*byte)(pointer)

	return *pb == 1
}

func MemSet(s []byte, c byte, n int) {
	tmp := bytes.Repeat([]byte{c}, n)
	copy(s[:n], tmp)
}

func If[T any](condition bool, trueBranch, falseBranch T) T {
	if condition {
		return trueBranch
	}
	return falseBranch
}

func PrintULongToBuf(buf []byte, num uint64) int {
	start := len(buf) - 1
	buf[start] = byte(num%10) + '0'
	num /= 10

	for num > 0 {
		start--
		buf[start] = byte(num%10) + '0'
	}

	return start
}

func PhpPackReverseInt32(arg uint32) uint32 {
	return ((arg & 0xFF) << 24) | ((arg & 0xFF00) << 8) | ((arg >> 8) & 0xFF00) | ((arg >> 24) & 0xFF)
}

func PhpPackReverseInt64(arg uint64) uint64 {
	slices := unsafe.Slice((*uint32)(unsafe.Pointer(&arg)), 2)
	slices[0] = PhpPackReverseInt32(slices[0])
	slices[1] = PhpPackReverseInt32(slices[1])

	return *(*uint64)(unsafe.Pointer(&slices[0]))
}

func PhpPackReverseInt16(arg uint16) uint16 {
	return ((arg & 0xFF) << 8) | ((arg >> 8) & 0xFF)
}

func PhpPackParseFloat(littleEndian bool, src []byte) float32 {
	var f float32
	fPtr := unsafe.Slice((*byte)(unsafe.Pointer(&f)), 4)
	copy(fPtr, src[:4])

	u32 := *(*uint32)(unsafe.Pointer(&fPtr[0]))
	if !IsLittleEndian() {
		if littleEndian {
			u32 = PhpPackReverseInt32(*(*uint32)(unsafe.Pointer(&fPtr[0])))
		}
	} else {
		if littleEndian {
			u32 = PhpPackReverseInt32(*(*uint32)(unsafe.Pointer(&fPtr[0])))
		}
	}

	return *(*float32)(unsafe.Pointer(&u32))
}

func PhpPackParseDouble(littleEndian bool, src []byte) float64 {
	var f float64
	fPtr := unsafe.Slice((*byte)(unsafe.Pointer(&f)), 8)
	copy(fPtr, src[:8])

	u64 := *(*uint64)(unsafe.Pointer(&fPtr[0]))
	if !IsLittleEndian() {
		if littleEndian {
			u64 = PhpPackReverseInt64(*(*uint64)(unsafe.Pointer(&fPtr[0])))
		}
	} else {
		if littleEndian {
			u64 = PhpPackReverseInt64(*(*uint64)(unsafe.Pointer(&fPtr[0])))
		}
	}

	return *(*float64)(unsafe.Pointer(&u64))
}
