package unpack

import (
	"fmt"
	"github.com/xycczZ/php_pack/internal/utils"
	"log"
	"math"
	"strconv"
	"unsafe"
)

type Option struct {
	Format string
	Val    []byte
	Offset int
}

func NewOption(format string, val []byte) *Option {
	return &Option{
		Format: format,
		Val:    val,
		Offset: 0,
	}
}

// PHPUnpack a,A,Z,h,H 返回[]byte,
// 返回整数的统一都返回int64, 因为PHP都是用zend_long接收的: c, C, s, S, n, v, i, I, l, L, N, V, q, Q, J, P
// 返回浮点数统一都返回float64, f, g, G | d, e, E
// x, X, @: 不返回值
// 如果在format中没有指明key的，默认设置为字符串的索引, 从1开始, "1", "2"...
func PHPUnpack(option *Option) (map[string]any, error) {
	result := make(map[string]any)

	format := []byte(option.Format)
	formatLen := len(format)
	input := option.Val
	inputLen := len(input)
	inputPos := 0
	offset := option.Offset

	if offset < 0 || offset > inputLen {
		return nil, fmt.Errorf("offset error: %d\n", offset)
	}

	input = input[offset:]
	inputLen -= offset

	formatPos := 0

	for formatLen > 0 {
		formatLen--
		repetitions := 1

		theType := format[formatPos]
		formatPos++
		var argb int
		var name []byte
		var nameLen int
		size := 0

		var c byte
		if formatLen > 0 {
			c = format[formatPos]
			if c >= '0' && c <= '9' {
				var err error
				repetitions, err = strconv.Atoi(string(format[formatPos]))
				if err != nil {
					return nil, err
				}

				formatPos++
				formatLen--
				for formatLen > 0 && format[formatPos] >= '0' && format[formatPos] <= '9' {
					repNext, err := strconv.Atoi(string(format[formatPos]))
					if err != nil {
						return nil, err
					}
					repetitions = repetitions*10 + repNext
					formatPos++
					formatLen--
				}
			} else if c == '*' {
				repetitions = -1
				formatPos++
				formatLen--
			}
		}

		namePos := formatPos
		name = format[:]
		argb = repetitions

		for formatLen > 0 && format[formatPos] != '/' {
			formatLen--
			formatPos++
		}

		nameLen = formatPos - namePos

		if nameLen > 200 {
			nameLen = 200
		}

		switch theType {
		// Never use any input
		case 'X':
			size = -1
			if repetitions < 0 {
				log.Printf("type %c: '*' ignored", theType)
				repetitions = 1
			}
		case '@':
			size = 0
		case 'a', 'A', 'Z':
			size = repetitions
			repetitions = 1
		case 'h', 'H':
			size = utils.If(repetitions > 0, (repetitions+(repetitions%2))/2, repetitions)
			repetitions = 1
		case 'c', 'C', 'x':
			size = 1
		case 's', 'S', 'n', 'v':
			size = 2
		case 'i', 'I':
			size = 4 // size_of(int)
		case 'l', 'L', 'N', 'V':
			size = 4
		case 'q', 'Q', 'J', 'P':
			size = 8
		case 'f', 'g', 'G':
			size = 4 // sizeof(float)
		case 'd', 'e', 'E':
			size = 8 // sizeof(double)
		default:
			return nil, fmt.Errorf("invalid format type %c\n", theType)
		}

		if size != 0 && size != -1 && size < 0 {
			return nil, fmt.Errorf("type %c: integer overflow", theType)
		}

		keyPos := 0
		// Do actual unpacking
		for i := 0; i != repetitions; i++ {
			if size != 0 && size != -1 && math.MaxInt-size+1 < inputPos {
				return nil, fmt.Errorf("type %c: integer overflow", theType)
			}

			realName := []byte{}
			if (inputPos + size) <= inputLen {
				if repetitions == 1 && nameLen > 0 {
					// use a part of the formatarg argument directly as the name
					realName = name[namePos:(namePos + nameLen)]
				} else {
					// need to add the 1-based element number to the name
					buf := make([]byte, 20)
					end := utils.PrintULongToBuf(buf, uint64(i+1))
					realName = append(name[namePos:(namePos+nameLen)], buf[end:]...)
				}

				var key string
				if len(realName) > 0 {
					key = string(realName)
				} else {
					keyPos++
					key = fmt.Sprintf("%d", keyPos)
				}
				switch theType {
				case 'a':
					length := inputLen - inputPos
					if size >= 0 && length > size {
						length = size
					}
					size = length
					s := input[inputPos:(inputPos + length)]
					result[key] = s
				case 'A':
					var padn byte = '\000'
					var pads byte = ' '
					var padt byte = '\t'
					var padc byte = '\r'
					var padl byte = '\n'

					length := inputLen - inputPos
					if size >= 0 && length > size {
						length = size
					}

					size = length
					length--

					for length >= 0 {
						if input[inputPos+length] != padn &&
							input[inputPos+length] != pads &&
							input[inputPos+length] != padt &&
							input[inputPos+length] != padc &&
							input[inputPos+length] != padl {
							break
						}
					}

					s := input[inputPos:(inputPos + length + 1)]
					result[key] = s
				case 'Z':
					var pad byte = '\000'
					length := inputLen - inputPos

					if size >= 0 && length > size {
						length = size
					}
					size = length

					for s := 0; s < length; s++ {
						if input[inputPos+s] == pad {
							length = s
							break
						}
					}

					s := input[inputPos:(inputPos + length)]
					result[key] = s
				case 'h', 'H':
					length := (inputLen - inputPos) * 2
					nibbleShift := utils.If(theType == 'h', 0, 4)
					first := 1

					if size >= 0 && length > (size*2) {
						length = size * 2
					}

					if length > 0 && argb > 0 {
						length -= argb % 2
					}

					buf := make([]byte, length)

					ipos := 0
					opos := 0
					for ; opos < length; opos++ {
						cc := (input[inputPos+ipos] >> nibbleShift) & 0xf

						if cc < 10 {
							cc += '0'
						} else {
							cc += 'a' - 10
						}

						buf[opos] = cc
						nibbleShift = (nibbleShift + 4) & 7

						if first == 0 {
							ipos++
							first = 1
						} else {
							first--
						}
					}

					result[key] = buf
				case 'c', 'C':
					x := input[inputPos]
					if theType == 'c' {
						// signed
						result[key] = int64(x)
					} else {
						result[key] = int64(x)
					}
				case 's', 'S', 'n', 'v':
					x := *(*uint16)(unsafe.Pointer(&input[inputPos]))
					if theType == 's' {
						result[key] = int64(x)
					} else if (theType == 'n' && utils.IsLittleEndian()) || (theType == 'v' && !utils.IsLittleEndian()) {
						result[key] = int64(utils.PhpPackReverseInt16(x))
					} else {
						result[key] = int64(x)
					}
				case 'i', 'I':
					if theType == 'i' {
						result[key] = int64(*(*int)(unsafe.Pointer(&input[inputPos])))
					} else {
						result[key] = int64(*(*uint)(unsafe.Pointer(&input[inputPos])))
					}
				case 'l', 'L', 'N', 'V':
					x := *(*uint32)(unsafe.Pointer(&input[inputPos]))
					if theType == 'l' {
						result[key] = int64(x)
					} else if (theType == 'N' && utils.IsLittleEndian()) || (theType == 'V' && !utils.IsLittleEndian()) {
						result[key] = utils.PhpPackReverseInt32(x)
					} else {
						result[key] = int64(x) // result[key] = x
					}
				case 'q', 'Q', 'J', 'P':
					x := *(*uint64)(unsafe.Pointer(&input[inputPos]))
					if theType == 'q' {
						result[key] = int64(x)
					} else if (theType == 'J' && utils.IsLittleEndian()) || (theType == 'P' && !utils.IsLittleEndian()) {
						result[key] = int64(utils.PhpPackReverseInt64(x))
					} else {
						result[key] = int64(x)
					}
				case 'f', 'g', 'G':
					if theType == 'g' {
						result[key] = float64(utils.PhpPackParseFloat(true, input[inputPos:(inputPos+4)]))
					} else if theType == 'G' {
						result[key] = float64(utils.PhpPackParseFloat(false, input[inputPos:(inputPos+4)]))
					} else {
						var v []byte
						copy(v, input[inputPos:(inputPos+4)])
						result[key] = float64(*(*float32)(unsafe.Pointer(&v[0])))
					}
				case 'd', 'e', 'E':
					if theType == 'e' {
						result[key] = utils.PhpPackParseDouble(true, input[inputPos:(inputPos+8)])
					} else if theType == 'E' {
						result[key] = utils.PhpPackParseDouble(false, input[inputPos:(inputPos+8)])
					} else {
						var v []byte
						copy(v, input[inputPos:(inputPos+8)])
						result[key] = *(*float64)(unsafe.Pointer(&v[0]))
					}
				case 'x':
					log.Printf("format x: do nothing")
				case 'X':
					if inputPos < size {
						inputPos = -size
						i = repetitions - 1

						if repetitions >= 0 {
							log.Printf("type %c: outside of string", theType)
						}
					}
				case '@':
					if repetitions <= inputLen {
						inputPos = repetitions
					} else {
						log.Printf("type %c: outside of string", theType)
					}
					i = repetitions - 1
				}

				inputPos += size
				if inputPos < 0 {
					if size != -1 {
						log.Printf("type %c: outside of string", theType)
					}
					inputPos = 0
				}
			} else if repetitions < 0 {
				break
			} else {
				return nil, fmt.Errorf("type %c: not enough input, need %d, have %d", theType, size, inputLen-inputPos)
			}
		}

		if formatLen > 0 {
			formatLen--
			formatPos++
		}
	}

	return result, nil
}
