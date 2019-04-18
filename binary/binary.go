package binary

import (
	"encoding/binary"
	"io"
	"math"
)

func GetUint16L(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

func PutUint16L(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}

func GetUint16B(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func PutUint16B(b []byte, v uint16) {
	binary.BigEndian.PutUint16(b, v)
}

func GetUint32L(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

func PutUint32L(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

func GetUint32B(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

func PutUint32B(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}

func GetUint64LE(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}

func PutUint64LE(b []byte, v uint64) {
	binary.LittleEndian.PutUint64(b, v)
}

func GetUint64B(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func PutUint64B(b []byte, v uint64) {
	binary.BigEndian.PutUint64(b, v)
}

func GetFloat32B(b []byte) float32 {
	return math.Float32frombits(GetUint32B(b))
}

func PutFloat32B(b []byte, v float32) {
	PutUint32B(b, math.Float32bits(v))
}

func GetFloat32L(b []byte) float32 {
	return math.Float32frombits(GetUint32L(b))
}

func PutFloat32L(b []byte, v float32) {
	PutUint32L(b, math.Float32bits(v))
}

func GetFloat64B(b []byte) float64 {
	return math.Float64frombits(GetUint64B(b))
}

func PutFloat64B(b []byte, v float64) {
	PutUint64B(b, math.Float64bits(v))
}

func GetFloat64L(b []byte) float64 {
	return math.Float64frombits(GetUint64LE(b))
}

func PutFloat64LE(b []byte, v float64) {
	PutUint64LE(b, math.Float64bits(v))
}

func UvarintSize(x uint64) int {
	i := 0
	for x >= 0x80 {
		x >>= 7
		i++
	}
	return i + 1
}

func VarintSize(x int64) int {
	ux := uint64(x) << 1
	if x < 0 {
		ux = ^ux
	}
	return UvarintSize(ux)
}

func GetUvarint(b []byte) (uint64, int) {
	return binary.Uvarint(b)
}

func PutUvarint(b []byte, v uint64) int {
	return binary.PutUvarint(b, v)
}

func GetVarint(b []byte) (int64, int) {
	return binary.Varint(b)
}

func PutVarint(b []byte, v int64) int {
	return binary.PutVarint(b, v)
}

func ReadUvarint(r io.ByteReader) (uint64, error) {
	return binary.ReadUvarint(r)
}

func ReadVarint(r io.ByteReader) (int64, error) {
	return binary.ReadVarint(r)
}
