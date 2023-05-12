package unpack

import (
	"fmt"
	"github.com/xycczZ/php_pack/pack"
	"log"
	"reflect"
	"strconv"
	"testing"
)

func TestPHPUnpack(t *testing.T) {
	cases := []struct {
		Pack     string
		Unpack   string
		Arg      any
		Expected map[string]any
	}{
		// pack64.phpt
		{Pack: "Q", Unpack: "Q", Arg: 0xfffffffffffe, Expected: map[string]any{"1": int64(281474976710654)}},
		{Pack: "Q", Unpack: "Q", Arg: 0, Expected: map[string]any{"1": int64(0)}},
		{Pack: "Q", Unpack: "Q", Arg: -1, Expected: map[string]any{"1": int64(-1)}},
		// pack.phpt
		{Pack: "A", Unpack: "A", Arg: "hello world", Expected: map[string]any{"1": []byte("h")}},
		{Pack: "A*", Unpack: "A*", Arg: "hello world", Expected: map[string]any{"1": []byte("hello world")}},

		{Pack: "C", Unpack: "C", Arg: -127, Expected: map[string]any{"1": int64(129)}},
		{Pack: "C", Unpack: "C", Arg: 127, Expected: map[string]any{"1": int64(127)}},
		{Pack: "C", Unpack: "C", Arg: 255, Expected: map[string]any{"1": int64(255)}},
		{Pack: "C", Unpack: "C", Arg: -129, Expected: map[string]any{"1": int64(127)}},

		{Pack: "H", Unpack: "H", Arg: 0x04, Expected: map[string]any{"1": []byte(strconv.Itoa(4))}},

		{Pack: "I", Unpack: "I", Arg: 65534, Expected: map[string]any{"1": int64(65534)}},
		{Pack: "I", Unpack: "I", Arg: 0, Expected: map[string]any{"1": int64(0)}},
		{Pack: "I", Unpack: "I", Arg: -1000, Expected: map[string]any{"1": int64(4294966296)}},
		{Pack: "I", Unpack: "I", Arg: -64434, Expected: map[string]any{"1": int64(4294902862)}},
		{Pack: "I", Unpack: "I", Arg: 4294967296, Expected: map[string]any{"1": int64(0)}},
		{Pack: "I", Unpack: "I", Arg: -4294967296, Expected: map[string]any{"1": int64(0)}},

		{Pack: "L", Unpack: "L", Arg: 65534, Expected: map[string]any{"1": int64(65534)}},
		{Pack: "L", Unpack: "L", Arg: 0, Expected: map[string]any{"1": int64(0)}},
		{Pack: "L", Unpack: "L", Arg: 2147483650, Expected: map[string]any{"1": int64(2147483650)}},
		{Pack: "L", Unpack: "L", Arg: 4294967295, Expected: map[string]any{"1": int64(4294967295)}},
		{Pack: "L", Unpack: "L", Arg: -2147483648, Expected: map[string]any{"1": int64(2147483648)}},
	}

	for i := range cases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			pr, err := pack.PHPPack(cases[i].Pack, cases[i].Arg)
			if err != nil {
				t.Errorf("pack failed, format: %s, err: %v\n", cases[i].Pack, err)
				return
			}
			r, err := PHPUnpack(NewOption(cases[i].Unpack, pr))
			if err != nil {
				t.Errorf("unpack failed, format: %s, err: %v\n", cases[i].Unpack, err)
				return
			}

			if !mapEq(r, cases[i].Expected) {
				t.Errorf("unpack error, expected: %v, actual: %v\n", cases[i].Expected, r)
				return
			}
		})
	}
}

func TestPHPUnpack2(t *testing.T) {
	bin, err := pack.PHPPack("c2n2", 0x1234, 0x5678, 65, 66)
	if err != nil {
		t.Errorf("pack failed: %v\n", err)
		return
	}
	t.Logf("pack result: %v\n", bin)
	r, err := PHPUnpack(NewOption("c2chars/n2int", bin))
	if err != nil {
		t.Errorf("unpack failed: %v\n", err)
		return
	}
	t.Logf("result %v\n", r)
}

func mapEq(actual, expected map[string]any) bool {
	if len(actual) != len(expected) {
		log.Printf("length error, actual: %d, expected: %d", len(actual), len(expected))
		return false
	}

	for ak, av := range actual {
		if bv, ok := expected[ak]; ok {
			rav := reflect.ValueOf(av)
			rbv := reflect.ValueOf(bv)
			if rav.Type() != rbv.Type() {
				log.Printf("type not equals, actual: %s, expected: %s\n", rav.Type(), rbv.Type())
				return false
			}

			eq := false
			switch rav.Kind() {
			case reflect.Int64:
				eq = rav.Int() == rbv.Int()
			case reflect.Float64:
				eq = rav.Float() == rbv.Float()
			case reflect.Slice:
				eq = sliceEq(rav.Interface().([]byte), rbv.Interface().([]byte))
			default:
				log.Printf("unknown type: %s\n", rav.Kind())
				return false
			}
			if !eq {
				log.Printf("value not equal, actual: %v, expected: %v\n", av, bv)
				return eq
			}
		} else {
			log.Printf("key [%s] is not exists of map expected[%v]\n", ak, expected)
			return false
		}
	}

	return true
}

func sliceEq[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
