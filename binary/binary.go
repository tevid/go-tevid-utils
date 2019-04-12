package trojanGo

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"reflect"
	"regexp"
	"strconv"
)

//默认标签名称
const DEFAULT_TAG_NAME = "binary"

var (
	ErrCannotSet     = errors.New("binary: field can not set")
	ErrUnsupportType = errors.New("binary: unsupported type")
	regexBinary      = regexp.MustCompile("bigEndian|littleEndian|null-terminated|(stringsize)=(\\d+)")
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

//获取对象的尺寸
func Sizeof(obj interface{}) (int, error) {
	return sizeof(reflect.ValueOf(obj))
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
		tag, ok := sf.Tag.Lookup(DEFAULT_TAG_NAME)
		if ok {
			taginfos := regexBinary.FindAllStringSubmatch(tag, -1)

			for _, info := range taginfos {
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
