package pack

import (
	"fmt"
	"log"
	"math"
	"php_pack/internal/utils"
	"strconv"
	"unsafe"
)

var byteMap [1]int
var intMap [4]int
var machineEndianShortMap [2]int
var bigEndianShortMap [2]int
var littleEndianShortMap [2]int
var machineEndianLongMap [4]int
var bigEndianLongMap [4]int
var littleEndianLongMap [4]int
var machineEndianLongLongMap [8]int
var bigEndianLongLongMap [8]int
var littleEndianLongLongMap [8]int

func PHPPack(format string, args ...any) ([]byte, error) {
	formatLen := len(format)
	formatCount := 0
	currentArg := 0
	numArgs := len(args)

	formatCodes := make([]uint8, formatLen)
	formatArgs := make([]int, formatLen)

	outputPos := 0
	outputSize := 0

	for i := 0; i < formatLen; formatCount++ {
		code := format[i]
		i++
		arg := 1

		if i < formatLen {
			c := format[i]
			if c == '*' {
				arg = -1
				i++
			} else if c >= '0' && c <= '9' {
				var err error
				arg, err = strconv.Atoi(string(format[i]))
				if err != nil {
					return nil, err
				}
				i++
				for i < formatLen && format[i] >= '0' && format[i] <= '9' {
					argNext, err := strconv.Atoi(string(format[i]))
					if err != nil {
						return nil, err
					}
					arg = arg*10 + argNext
					i++
				}
			}
		}

		switch code {
		// Never uses any args
		case 'x', 'X', '@':
			if arg < 0 {
				log.Printf("type: %c: '*' ignored", code)
				arg = 1
			}
		// Always uses one arg
		case 'a', 'A', 'Z', 'h', 'H':
			if currentArg >= numArgs {
				return nil, fmt.Errorf("type %c: not enough arguments", code)
			}
			if arg < 0 {
				argStr, err := utils.ConvertToString(args[currentArg])
				if err != nil {
					return nil, err
				}
				arg = len(argStr)
				if code == 'Z' {
					// add one because Z is always NUL-terminated:
					// pack("Z*", "aa") == "aa\0"
					// pack("Z2", "aa") == "a\0"
					arg++
				}
			}

			currentArg++
		case 'q', 'Q', 'J', 'P', 'c', 'C',
			's', 'S', 'i', 'I', 'l', 'L', 'n', 'N',
			'v', 'V', 'f', 'g', 'G', 'd', 'e', 'E':
			if arg < 0 {
				arg = numArgs - currentArg
			}
			if currentArg > math.MaxInt-arg {
				return nil, fmt.Errorf("type %c: too few arguments", code)
			}
			currentArg += arg
			if currentArg > numArgs {
				return nil, fmt.Errorf("type %c: too few arguments", code)
			}
		default:
			return nil, fmt.Errorf("type %c: unknown format code", code)
		}

		formatCodes[formatCount] = code
		formatArgs[formatCount] = arg
	}

	if currentArg < numArgs {
		log.Printf("%d arguments unused", numArgs-currentArg)
	}

	for i := 0; i < formatCount; i++ {
		code := formatCodes[i]
		arg := formatArgs[i]
		switch code {
		case 'h', 'H':
			// INC_OUTPUTPOS
			if err := incOutputPos((arg+(arg%2))/2, 1, code, &outputPos); err != nil {
				return nil, err
			}
		case 'a', 'A', 'Z', 'c', 'C', 'x':
			if err := incOutputPos(arg, 1, code, &outputPos); err != nil {
				return nil, err
			}
		case 's', 'S', 'n', 'v':
			if err := incOutputPos(arg, 2, code, &outputPos); err != nil {
				return nil, err
			}
		case 'i', 'I':
			// sizeof(int)
			if err := incOutputPos(arg, 4, code, &outputPos); err != nil {
				return nil, err
			}
		case 'l', 'L', 'N', 'V':
			if err := incOutputPos(arg, 4, code, &outputPos); err != nil {
				return nil, err
			}
		case 'q', 'Q', 'J', 'P':
			if err := incOutputPos(arg, 8, code, &outputPos); err != nil {
				return nil, err
			}
		case 'f', 'g', 'G':
			// sizeof float
			if err := incOutputPos(arg, 4, code, &outputPos); err != nil {
				return nil, err
			}
		case 'd', 'e', 'E':
			// sizeof double
			if err := incOutputPos(arg, 8, code, &outputPos); err != nil {
				return nil, err
			}
		case 'X':
			outputPos -= arg
			if outputPos < 0 {
				log.Printf("type %c: outside of string", code)
				outputPos = 0
			}
		case '@':
			outputPos = arg
		}

		if outputSize < outputPos {
			outputSize = outputPos
		}
	}

	output := make([]byte, outputSize)
	outputPos = 0
	currentArg = 0

	// do actual packing
	for i := 0; i < formatCount; i++ {
		code := formatCodes[i]
		arg := formatArgs[i]

		switch code {
		case 'a', 'A', 'Z':
			argCp := utils.If(code != 'Z', arg, utils.Max(0, arg-1))

			utils.MemSet(output[outputPos:], utils.If[byte](code == 'a' || code == 'Z', '\000', ' '), arg)
			argStr, err := utils.ConvertToString(args[currentArg])
			if err != nil {
				return nil, err
			}
			copyLen := utils.Min(len(argStr), argCp)
			copy(output[outputPos:(outputPos+copyLen)], []byte(argStr)[:copyLen])

			outputPos += arg
			currentArg++
		case 'h', 'H':
			nibbleShift := utils.If(code == 'h', 0, 4)
			first := 1
			str, err := utils.ConvertToString(args[currentArg])
			if err != nil {
				return nil, err
			}

			v := []byte(str)
			outputPos--
			if arg > len(str) {
				log.Printf("type %c: not enough characters in string", code)
				arg = len(str)
			}

			vpos := 0
			for ; arg > 0; arg-- {
				n := v[vpos]
				vpos++
				if n >= '0' && n <= '9' {
					n -= '0'
				} else if n >= 'A' && n <= 'F' {
					n -= ('A' - 10)
				} else if n >= 'a' && n <= 'f' {
					n -= ('a' - 10)
				} else {
					log.Printf("type %c: illegal hex digit %c", code, n)
					n = 0
				}

				if first != 0 {
					first--
					outputPos++
					output[outputPos] = 0
				} else {
					first = 1
				}

				output[outputPos] |= (n << nibbleShift)
				nibbleShift = (nibbleShift + 4) & 7
			}

			outputPos++
			currentArg++
		case 'c', 'C':
			for arg > 0 {
				arg--
				if err := pack(args[currentArg], 1, byteMap[:], output[outputPos:(outputPos+1)]); err != nil {
					return nil, err
				}
				currentArg++
				outputPos++
			}
		case 's', 'S', 'n', 'v':
			eMap := machineEndianShortMap[:]
			if code == 'n' {
				eMap = bigEndianShortMap[:]
			} else if code == 'v' {
				eMap = littleEndianShortMap[:]
			}

			for arg > 0 {
				arg--
				if err := pack(args[currentArg], 2, eMap, output[outputPos:(outputPos+2)]); err != nil {
					return nil, err
				}
				currentArg++
				outputPos += 2
			}
		case 'i', 'I':
			for arg > 0 {
				arg--
				if err := pack(args[currentArg], 4, intMap[:], output[outputPos:(outputPos+4)]); err != nil {
					return nil, err
				}
				outputPos += 4
				currentArg++
			}
		case 'l', 'L', 'N', 'V':
			eMap := machineEndianLongMap[:]
			if code == 'N' {
				eMap = bigEndianLongMap[:]
			} else if code == 'V' {
				eMap = littleEndianLongMap[:]
			}

			for arg > 0 {
				arg--
				if err := pack(args[currentArg], 4, eMap, output[outputPos:(outputPos+4)]); err != nil {
					return nil, err
				}
				currentArg++
				outputPos += 4
			}
		case 'q', 'Q', 'J', 'P':
			eMap := machineEndianLongLongMap[:]
			if code == 'J' {
				eMap = bigEndianLongLongMap[:]
			} else if code == 'P' {
				eMap = littleEndianLongLongMap[:]
			}

			for arg > 0 {
				arg--
				if err := pack(args[currentArg], 8, eMap, output[outputPos:(outputPos+8)]); err != nil {
					return nil, err
				}
				currentArg++
				outputPos += 8
			}
		case 'f':
			for arg > 0 {
				arg--

				v, err := utils.ConvertToFloat(args[currentArg])
				if err != nil {
					return nil, err
				}
				vv := float32(v)
				s := unsafe.Slice((*byte)(unsafe.Pointer(&vv)), 4)
				copy(output[outputPos:(outputPos+4)], s)
				outputPos += 4
				currentArg++
			}
		case 'g':
			for arg > 0 {
				arg--
				v, err := utils.ConvertToFloat(args[currentArg])
				if err != nil {
					return nil, err
				}
				phpPackCopyFloat(true, output[outputPos:(outputPos+4)], float32(v))
				outputPos += 4
				currentArg++
			}
		case 'G':
			for arg > 0 {
				arg--
				v, err := utils.ConvertToFloat(args[currentArg])
				if err != nil {
					return nil, err
				}
				phpPackCopyFloat(false, output[outputPos:(outputPos+4)], float32(v))
				outputPos += 4
				currentArg++
			}
		case 'd':
			for arg > 0 {
				arg--
				v, err := utils.ConvertToFloat(args[currentArg])
				if err != nil {
					return nil, err
				}
				copy(output[outputPos:(outputPos+8)], unsafe.Slice((*byte)(unsafe.Pointer(&v)), 8))
				outputPos += 8
				currentArg++
			}
		case 'e':
			for arg > 0 {
				arg--
				v, err := utils.ConvertToFloat(args[currentArg])
				if err != nil {
					return nil, err
				}
				phpPackCopyDouble(true, output[outputPos:(outputPos+8)], v)
				currentArg++
				outputPos += 8
			}
		case 'E':
			for arg > 0 {
				arg--
				v, err := utils.ConvertToFloat(args[currentArg])
				if err != nil {
					return nil, err
				}
				phpPackCopyDouble(false, output[outputPos:(outputPos+8)], v)
				currentArg++
				outputPos += 8
			}
		case 'x':
			utils.MemSet(output[outputPos:], '\000', arg)
			outputPos += arg
		case 'X':
			outputPos -= arg
			if outputPos < 0 {
				outputPos = 0
			}
		case '@':
			if arg > outputPos {
				utils.MemSet(output[outputPos:], '\000', arg-outputPos)
			}
			outputPos = arg
		}
	}

	return output, nil
}

// #define INC_OUTPUTPOS(a,b)
//
//	if ((a) < 0 || ((INT_MAX - outputpos)/((int)b)) < (a)) {
//	   efree(formatcodes);
//	   efree(formatargs);
//	   zend_value_error("Type %c: integer overflow in format string", code);
//	   RETURN_THROWS();
//	}
//
//	outputpos += (a)*(b);
func incOutputPos(a, b int, code uint8, outputPos *int) error {
	if a < 0 || ((math.MaxInt-(*outputPos))/b) < a {
		return fmt.Errorf("type %c: integer overflow in format string", code)
	}
	*outputPos += a * b
	return nil
}

// pack.c php_pack(zval *val, size_t size, int *map, char *output)
func pack(val any, size int, byteMap []int, output []byte) error {
	lv, err := utils.ConvertToLong(val)
	if err != nil {
		return err
	}

	vp := (*byte)(unsafe.Pointer(&lv))
	for i := 0; i < size; i++ {
		output[i] = *(*byte)(unsafe.Add(unsafe.Pointer(vp), byteMap[i]))
	}
	return nil
}

func phpPackCopyFloat(littleEndian bool, dst []byte, f float32) {
	u := *(*uint32)(unsafe.Pointer(&f))
	fv := f
	if !utils.IsLittleEndian() {
		if littleEndian {
			ut := utils.PhpPackReverseInt32(u)
			fv = *(*float32)(unsafe.Pointer(&ut))
		}
	} else {
		if !littleEndian {
			ut := utils.PhpPackReverseInt32(u)
			fv = *(*float32)(unsafe.Pointer(&ut))
		}
	}

	copy(dst, unsafe.Slice((*byte)(unsafe.Pointer(&fv)), 4))
}

func phpPackCopyDouble(littleEndian bool, dst []byte, f float64) {
	u := *(*uint64)(unsafe.Pointer(&f))
	fv := f
	if !utils.IsLittleEndian() {
		if littleEndian {
			ut := utils.PhpPackReverseInt64(u)
			fv = *(*float64)(unsafe.Pointer(&ut))
		}
	} else {
		if !littleEndian {
			ut := utils.PhpPackReverseInt64(u)
			fv = *(*float64)(unsafe.Pointer(&ut))
		}
	}

	copy(dst, unsafe.Slice((*byte)(unsafe.Pointer(&fv)), 8))
}

func init() {
	if utils.IsLittleEndian() {
		byteMap[0] = 0
		for i := 0; i < 4; i++ {
			intMap[i] = i
		}

		machineEndianShortMap[0] = 0
		machineEndianShortMap[1] = 1
		bigEndianShortMap[0] = 1
		bigEndianShortMap[1] = 0
		littleEndianShortMap[0] = 0
		littleEndianShortMap[1] = 1

		machineEndianLongMap[0] = 0
		machineEndianLongMap[1] = 1
		machineEndianLongMap[2] = 2
		machineEndianLongMap[3] = 3
		bigEndianLongMap[0] = 3
		bigEndianLongMap[1] = 2
		bigEndianLongMap[2] = 1
		bigEndianLongMap[3] = 0
		littleEndianLongMap[0] = 0
		littleEndianLongMap[1] = 1
		littleEndianLongMap[2] = 2
		littleEndianLongMap[3] = 3

		machineEndianLongLongMap[0] = 0
		machineEndianLongLongMap[1] = 1
		machineEndianLongLongMap[2] = 2
		machineEndianLongLongMap[3] = 3
		machineEndianLongLongMap[4] = 4
		machineEndianLongLongMap[5] = 5
		machineEndianLongLongMap[6] = 6
		machineEndianLongLongMap[7] = 7
		bigEndianLongLongMap[0] = 7
		bigEndianLongLongMap[1] = 6
		bigEndianLongLongMap[2] = 5
		bigEndianLongLongMap[3] = 4
		bigEndianLongLongMap[4] = 3
		bigEndianLongLongMap[5] = 2
		bigEndianLongLongMap[6] = 1
		bigEndianLongLongMap[7] = 0
		littleEndianLongLongMap[0] = 0
		littleEndianLongLongMap[1] = 1
		littleEndianLongLongMap[2] = 2
		littleEndianLongLongMap[3] = 3
		littleEndianLongLongMap[4] = 4
		littleEndianLongLongMap[5] = 5
		littleEndianLongLongMap[6] = 6
		littleEndianLongLongMap[7] = 7
	} else {
		size := 8
		byteMap[0] = size - 1
		for i := 0; i < 4; i++ {
			intMap[i] = size - (4 - i)
		}

		machineEndianShortMap[0] = size - 2
		machineEndianShortMap[1] = size - 1
		bigEndianShortMap[0] = size - 2
		bigEndianShortMap[1] = size - 1
		littleEndianShortMap[0] = size - 1
		littleEndianShortMap[1] = size - 2

		machineEndianLongMap[0] = size - 4
		machineEndianLongMap[1] = size - 3
		machineEndianLongMap[2] = size - 2
		machineEndianLongMap[3] = size - 1
		bigEndianLongMap[0] = size - 4
		bigEndianLongMap[1] = size - 3
		bigEndianLongMap[2] = size - 2
		bigEndianLongMap[3] = size - 1
		littleEndianLongMap[0] = size - 1
		littleEndianLongMap[1] = size - 2
		littleEndianLongMap[2] = size - 3
		littleEndianLongMap[3] = size - 4

		machineEndianLongLongMap[0] = size - 8
		machineEndianLongLongMap[1] = size - 7
		machineEndianLongLongMap[2] = size - 6
		machineEndianLongLongMap[3] = size - 5
		machineEndianLongLongMap[4] = size - 4
		machineEndianLongLongMap[5] = size - 3
		machineEndianLongLongMap[6] = size - 2
		machineEndianLongLongMap[7] = size - 1
		bigEndianLongLongMap[0] = size - 8
		bigEndianLongLongMap[1] = size - 7
		bigEndianLongLongMap[2] = size - 6
		bigEndianLongLongMap[3] = size - 5
		bigEndianLongLongMap[4] = size - 4
		bigEndianLongLongMap[5] = size - 3
		bigEndianLongLongMap[6] = size - 2
		bigEndianLongLongMap[7] = size - 1
		littleEndianLongLongMap[0] = size - 1
		littleEndianLongLongMap[1] = size - 2
		littleEndianLongLongMap[2] = size - 3
		littleEndianLongLongMap[3] = size - 4
		littleEndianLongLongMap[4] = size - 5
		littleEndianLongLongMap[5] = size - 6
		littleEndianLongLongMap[6] = size - 7
		littleEndianLongLongMap[7] = size - 8
	}
}
