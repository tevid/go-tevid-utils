package binary

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

//自定义默认标签名称
const DefaultTagName = "binary"

var (
	ErrCannotSet         = errors.New("binary: field can not set")
	ErrUnsupportType     = errors.New("binary: unsupported type")
	ErrPackFormat        = errors.New("format pack: error format string")
	ErrPackFormatDataLen = errors.New("format pack: error format, because data is wrong")
	ErrNotImplemented    = errors.New("format pack: value type not implemented")
	regexBinary          = regexp.MustCompile("bigEndian|littleEndian|null-terminated|(stringsize)=(\\d+)")
	regexFormat          = regexp.MustCompile("([><]?)(\\w+)")
)

//字节序结构接口
type (
	//操作对象
	binaryObject struct {
		val                reflect.Value
		byteorderType      binary.ByteOrder
		stringsize         int
		terminatedWithZero bool
	}

	binaryStruct interface {
		serialize(*binaryObject) error
	}

	//普通结构
	structBinaryStruct struct {
		size int
	}

	//编包结构
	packBinaryStruct struct {
		order  binary.ByteOrder
		writer io.Writer
	}

	//解包结构
	unPackBinaryStruct struct {
		order  binary.ByteOrder
		reader io.Reader
	}
)

func PackTlv(tag int16, data []byte, order binary.ByteOrder) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, order, tag); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, order, int16(len(data))); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, order, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnPackTlv(b []byte, order binary.ByteOrder) (int16, []byte, error) {
	buf := bytes.NewBuffer(b)
	var tag, length int16
	if err := binary.Read(buf, order, &tag); err != nil {
		return 0, nil, err
	}
	if err := binary.Read(buf, order, &length); err != nil {
		return 0, nil, err
	}
	dataBuf := make([]byte, length)
	if err := binary.Read(buf, order, &dataBuf); err != nil {
		return 0, nil, err
	}
	return tag, dataBuf, nil
}

//获取对象的尺寸
func Sizeof(obj interface{}) (int, error) {
	return sizeof(reflect.ValueOf(obj))
}

func GetObjSize(obj interface{}) int {
	siz, err := Sizeof(obj)
	if err != nil {
		return 0
	}
	return siz
}

func sizeof(v reflect.Value) (int, error) {
	vst := structBinaryStruct{}
	err := doSerialize(&vst, v)
	if err != nil {
		return -1, err
	}
	return vst.size, nil
}

func (self *structBinaryStruct) serialize(obj *binaryObject) error {
	switch obj.val.Kind() {
	case reflect.Int8, reflect.Uint8:
		self.size++
	case reflect.Int16, reflect.Uint16:
		self.size += 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		self.size += 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		self.size += 8
	case reflect.Array, reflect.Slice:
		if obj.val.Len() > 0 {
			if obj.val.Type().Elem().Kind() == reflect.Struct {
				for i := 0; i < obj.val.Len(); i++ {
					isize, err := sizeof(obj.val.Index(i))
					if err != nil {
						return err
					}
					self.size += isize
				}
			} else {
				isize, err := sizeof(obj.val.Index(0))
				if err != nil {
					return err
				}
				self.size += obj.val.Len() * isize
			}
		}
	case reflect.String:
		if obj.terminatedWithZero {
			self.size += len([]byte(obj.val.String())) + 1
		} else {
			self.size += len([]byte(obj.val.String()))
		}
	default:
		return ErrUnsupportType
	}

	return nil
}

func Pack(w io.Writer, p interface{}) error {
	return pack(w, reflect.ValueOf(p), binary.LittleEndian)
}

func PackWithOrder(w io.Writer, p interface{}, o binary.ByteOrder) error {
	return pack(w, reflect.ValueOf(p), o)
}

func pack(w io.Writer, reflectValue reflect.Value, order binary.ByteOrder) error {
	return doSerialize(&packBinaryStruct{order: order, writer: w}, reflectValue)
}

func (v *packBinaryStruct) serialize(obj *binaryObject) error {
	order := v.order
	if obj.byteorderType != nil {
		order = obj.byteorderType
	}

	dataWord := [2]byte{}
	dataDWord := [4]byte{}
	dataLongLong := [8]byte{}

	switch obj.val.Kind() {

	case reflect.Int8:
		v.writer.Write([]byte{byte(obj.val.Int())})

	case reflect.Uint8:
		v.writer.Write([]byte{byte(obj.val.Uint())})

	case reflect.Int16:
		order.PutUint16(dataWord[:], uint16(obj.val.Int()))
		v.writer.Write(dataWord[:])

	case reflect.Uint16:
		order.PutUint16(dataWord[:], uint16(obj.val.Uint()))
		v.writer.Write(dataWord[:])

	case reflect.Int32:
		order.PutUint32(dataDWord[:], uint32(obj.val.Int()))
		v.writer.Write(dataDWord[:])

	case reflect.Uint32:

		order.PutUint32(dataDWord[:], uint32(obj.val.Uint()))
		v.writer.Write(dataDWord[:])

	case reflect.Int64:
		order.PutUint64(dataLongLong[:], uint64(obj.val.Int()))
		v.writer.Write(dataLongLong[:])

	case reflect.Uint64:
		order.PutUint64(dataLongLong[:], uint64(obj.val.Uint()))
		v.writer.Write(dataLongLong[:])

	case reflect.Float32:
		order.PutUint32(dataDWord[:], math.Float32bits(float32(obj.val.Float())))
		v.writer.Write(dataDWord[:])

	case reflect.Float64:
		order.PutUint64(dataLongLong[:], math.Float64bits(obj.val.Float()))
		v.writer.Write(dataLongLong[:])

	case reflect.Array, reflect.Slice:
		for i := 0; i < obj.val.Len(); i++ {
			err := pack(v.writer, obj.val.Index(i), order)
			if err != nil {
				return err
			}
		}

	case reflect.String:
		strVal := obj.val.String()
		if obj.stringsize > 0 {
			strVal = func(strVal string) string {
				if obj.stringsize < len(strVal) {
					return strVal[:obj.stringsize]
				}
				return strVal[:len(strVal)]
			}(strVal)

		}
		io.WriteString(v.writer, strVal)
		if obj.terminatedWithZero {
			v.writer.Write([]byte{0x00})
		}

	default:
		return ErrUnsupportType
	}

	return nil
}

func UnPack(r io.Reader, v interface{}) error {
	return unpack(r, reflect.ValueOf(v), binary.LittleEndian)
}

func UnPackWithOrder(r io.Reader, v interface{}, o binary.ByteOrder) error {
	return unpack(r, reflect.ValueOf(v), o)
}

func unpack(r io.Reader, v reflect.Value, o binary.ByteOrder) error {
	return doSerialize(&unPackBinaryStruct{order: o, reader: r}, v)
}

func (v *unPackBinaryStruct) serialize(obj *binaryObject) error {
	order := v.order
	if obj.byteorderType != nil {
		order = obj.byteorderType
	}

	var err error
	dataByte := [1]byte{}
	dataWord := [2]byte{}
	dataDWord := [4]byte{}
	dataLongLong := [8]byte{}

	switch obj.val.Kind() {
	case reflect.Int8:
		_, err = v.reader.Read(dataByte[:])
		obj.val.SetInt(int64(dataByte[0]))
	case reflect.Uint8:
		_, err = v.reader.Read(dataByte[:])
		obj.val.SetUint(uint64(dataByte[0]))

	case reflect.Int16:
		_, err = v.reader.Read(dataWord[:])
		obj.val.SetInt(int64(order.Uint16(dataWord[:])))
	case reflect.Uint16:
		_, err = v.reader.Read(dataWord[:])
		obj.val.SetUint(uint64(order.Uint16(dataWord[:])))

	case reflect.Int32:
		_, err = v.reader.Read(dataDWord[:])
		obj.val.SetInt(int64(order.Uint32(dataDWord[:])))
	case reflect.Uint32:
		_, err = v.reader.Read(dataDWord[:])
		obj.val.SetUint(uint64(order.Uint32(dataDWord[:])))

	case reflect.Int64:
		_, err = v.reader.Read(dataLongLong[:])
		obj.val.SetInt(int64(order.Uint64(dataLongLong[:])))
	case reflect.Uint64:
		_, err = v.reader.Read(dataLongLong[:])
		obj.val.SetUint(uint64(order.Uint64(dataLongLong[:])))

	case reflect.Float32:
		_, err = v.reader.Read(dataDWord[:])
		obj.val.SetFloat(float64(math.Float32frombits(order.Uint32(dataDWord[:]))))
	case reflect.Float64:
		_, err = v.reader.Read(dataLongLong[:])
		obj.val.SetFloat(math.Float64frombits(order.Uint64(dataLongLong[:])))

	case reflect.Array: //数组类型
		for i := 0; i < obj.val.Len(); i++ {
			err = unpack(v.reader, obj.val.Index(i), order)
			if err != nil {
				return err
			}
		}

	case reflect.Slice: //切片类型
		for i := 0; i < obj.val.Len(); i++ {
			err = unpack(v.reader, obj.val.Index(i), order)
			if err != nil {
				return err
			}
		}

	case reflect.String: //字符串
		if obj.terminatedWithZero {
			var str string
			str, err = getStringterminateWithZero(v.reader)
			obj.val.SetString(str)
		} else {
			if obj.stringsize > 0 {
				buf := make([]byte, obj.stringsize)
				_, err = v.reader.Read(buf)
				obj.val.SetString(string(buf))
			} else {
				s, _ := getString(v.reader)
				obj.val.SetString(s)
			}
		}

	default:
		return ErrUnsupportType
	}

	return err
}

func getStringterminateWithZero(r io.Reader) (string, error) {
	buf := []byte{}
	single := []byte{0}

	for {
		_, err := r.Read(single)
		if err != nil {
			return "", err
		} else if single[0] == 0 {
			break
		} else {
			buf = append(buf, single[0])
		}
	}

	return string(buf), nil
}

func getString(r io.Reader) (string, error) {
	buf := []byte{}
	single := []byte{0}
	for {
		_, err := r.Read(single)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return "", err
			}
		} else {
			buf = append(buf, single[0])
		}
	}
	return string(buf), nil
}

func doSerialize(v binaryStruct, reflectValue reflect.Value) error {
	return doSerialize0(v, reflectValue, nil, nil)
}

func doSerialize0(bs binaryStruct, reflectValue reflect.Value, bo *binaryObject, sf *reflect.StructField) error {
	if reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}

	//属性必须能进行canset
	if !reflectValue.CanSet() {
		return ErrCannotSet
	}

	obj := &binaryObject{
		val: reflectValue,
	}

	if sf != nil {
		tag, ok := sf.Tag.Lookup(DefaultTagName)
		if ok {
			tagInfos := regexBinary.FindAllStringSubmatch(tag, -1)

			for _, info := range tagInfos {
				byteorder := info[0]
				nt := info[0]
				stringsize := info[1]
				stringsizeValue := info[2]

				if byteorder == "bigEndian" {
					obj.byteorderType = binary.BigEndian
				} else if byteorder == "littleEndian" {
					obj.byteorderType = binary.LittleEndian
				} else if nt == "null-terminated" {
					obj.terminatedWithZero = true
				} else if stringsize == "stringsize" {
					obj.stringsize, _ = strconv.Atoi(stringsizeValue)
				}
			}
		}
	}
	{
		switch reflectValue.Kind() {
		case
			reflect.Bool,
			reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Uintptr,
			reflect.Float32,
			reflect.Float64,
			reflect.Complex64,
			reflect.Complex128,
			reflect.Array,
			reflect.Slice,
			reflect.String:
			return bs.serialize(obj)
		case reflect.Struct: //支持结构嵌套结构
			for i := 0; i < reflectValue.NumField(); i++ {
				typeField := reflectValue.Type().Field(i)
				err := doSerialize0(bs, reflectValue.Field(i), obj, &typeField)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
	return ErrUnsupportType
}

//自定义的格式化打包
func FormatPack(format string, data ...interface{}) ([]byte, error) {

	tagInfos, err := getFormatInfo(format)
	if err != nil {
		return nil, ErrPackFormat
	}

	var byteorder binary.ByteOrder
	switch tagInfos[1] {
	case "<":
		byteorder = binary.LittleEndian
	case ">":
		byteorder = binary.BigEndian
	default:
		byteorder = binary.LittleEndian //默认小端序
	}

	formatstr := tagInfos[2]
	var result []byte

	//处理变长字符串
	if strings.Contains(formatstr, "s") {
		if len(data) == 0 {
			return nil, ErrPackFormat
		}
		n, _ := strconv.Atoi(strings.TrimRight(formatstr, "s"))
		val, _ := ToString(data[0])
		if n < len(val) {
			n = len(val)
		}
		result = append(result, []byte(fmt.Sprintf("%s%s", val, strings.Repeat("\x00", n-len(val))))...)
		return result, nil
	}

	formats := unsafeString2Bytes(formatstr)
	for idx, format := range formats {

		if idx >= len(data) {
			return nil, ErrPackFormatDataLen
		}

		switch format {
		case 'c', 'b', 'B', '?':
			value, _ := ToUint8(data[idx])
			result = append(result, value)
		case 'h', 'H':
			value, _ := ToUint16(data[idx])
			var bmBuf = make([]byte, 2)
			if byteorder == binary.LittleEndian {
				PutUint16L(bmBuf, value)
			} else {
				PutUint16B(bmBuf, value)
			}
			result = append(result, bmBuf...)
		case 'i', 'I', 'l', 'L':
			value, _ := ToUint32(data[idx])
			var bmBuf = make([]byte, 4)
			if byteorder == binary.LittleEndian {
				PutUint32L(bmBuf, value)
			} else {
				PutUint32B(bmBuf, value)
			}
			result = append(result, bmBuf...)
		case 'q', 'Q':
			value, _ := ToUint64(data[idx])
			var bmBuf = make([]byte, 8)
			if byteorder == binary.LittleEndian {
				PutUint64LE(bmBuf, value)
			} else {
				PutUint64B(bmBuf, value)
			}
			result = append(result, bmBuf...)
		case 'f':
			value, _ := ToFloat32(data[idx])
			var bmBuf = make([]byte, 4)
			if byteorder == binary.LittleEndian {
				PutFloat32L(bmBuf, value)
			} else {
				PutFloat32B(bmBuf, value)
			}
			result = append(result, bmBuf...)
		case 'd':
			value, _ := ToFloat64(data[idx])
			var bmBuf = make([]byte, 8)
			if byteorder == binary.LittleEndian {
				PutFloat64LE(bmBuf, value)
			} else {
				PutFloat64B(bmBuf, value)
			}
			result = append(result, bmBuf...)
		default:
			return nil, ErrPackFormat

		}
	}
	return result, nil
}

//自定义的格式化解包
func FormatUnPack(format string, result []byte) ([]interface{}, error) {

	tagInfos, err := getFormatInfo(format)
	if err != nil {
		return nil, ErrPackFormat
	}

	var byteorder binary.ByteOrder
	switch tagInfos[1] {
	case "<":
		byteorder = binary.LittleEndian
	case ">":
		byteorder = binary.BigEndian
	default:
		byteorder = binary.LittleEndian //默认小端序
	}

	formatstr := tagInfos[2]

	data := make([]interface{}, 0)

	if strings.Contains(formatstr, "s") {
		n, _ := strconv.Atoi(strings.TrimRight(formatstr, "s"))
		data = append(data, string(result[:n]))
		return data, nil
	}

	formats := unsafeString2Bytes(formatstr)

	for _, format := range formats {
		switch format {
		case 'c', 'b', 'B', '?':
		case 'h', 'H':
			cost := 2
			val := uint16(0)
			if byteorder == binary.LittleEndian {
				val = GetUint16L(result)
			} else {
				val = GetUint16B(result)
			}
			data = append(data, val)
			result = result[cost:]
		case 'i', 'I', 'l', 'L':
			cost := 4
			val := uint32(0)
			if byteorder == binary.LittleEndian {
				val = GetUint32L(result)
			} else {
				val = GetUint32B(result)
			}
			data = append(data, val)
			result = result[cost:]
		case 'q', 'Q':
			cost := 8
			val := uint64(0)
			if byteorder == binary.LittleEndian {
				val = GetUint64LE(result)
			} else {
				val = GetUint64B(result)
			}
			data = append(data, val)
			result = result[cost:]
		case 'f':
			cost := 4
			val := float32(0)
			if byteorder == binary.LittleEndian {
				val = GetFloat32L(result)
			} else {
				val = GetFloat32B(result)
			}
			data = append(data, val)
			result = result[cost:]
		case 'd':
			cost := 8
			val := float64(0)
			if byteorder == binary.LittleEndian {
				val = GetFloat64L(result)
			} else {
				val = GetFloat64B(result)
			}
			data = append(data, val)
			result = result[cost:]
		default:
			return nil, ErrPackFormat
		}
	}
	return data, nil
}

//计算大小
func FormatCalSize(format string) (int, error) {
	tagInfos, err := getFormatInfo(format)
	if err != nil {
		return 0, ErrPackFormat
	}
	formatStr := tagInfos[2]
	return calSize(formatStr)
}

//获取format信息
func getFormatInfo(format string) ([]string, error) {
	tagInfos := regexFormat.FindAllStringSubmatch(format, -1)
	if len(tagInfos) == 0 {
		return nil, ErrPackFormat
	}
	return tagInfos[0], nil
}

//计算大小
func calSize(formatStr string) (int, error) {
	var size int

	if strings.Contains(formatStr, "s") {
		n, _ := strconv.Atoi(strings.TrimRight(formatStr, "s"))
		return n, nil
	}

	formats := unsafeString2Bytes(formatStr)
	for _, format := range formats {
		switch format {
		case 'c', 'b', 'B', '?':
			size = size + 1
		case 'h', 'H':
			size = size + 2
		case 'i', 'I', 'l', 'L', 'f':
			size = size + 4
		case 'q', 'Q', 'd':
			size = size + 8
		}
	}
	return size, nil
}
