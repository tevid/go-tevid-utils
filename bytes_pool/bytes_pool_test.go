package bytes_pool

import (
	. "github.com/tevid/gohamcrest"
	"testing"
)

func TestBytesPool_AllocAndRelease(t *testing.T) {
	bp := NewBytesPool(512, 64*1024, 3*1024*1024)

	for i := 0; i < len(bp.entityList); i++ {
		tempData := make([][]byte, len(bp.entityList[i].chunks))

		for j := 0; j < len(tempData); j++ {
			data := bp.Alloc(bp.entityList[i].esize)
			Assert(t, cap(data), Equal(bp.entityList[i].esize))
			tempData[j] = data
		}

		Assert(t, bp.entityList[i].pos, Equal(uint64(0)))

		for j := 0; j < len(bp.entityList); j++ {
			bp.Release(tempData[j])
		}
		Assert(t, bp.entityList[i].pos, Not(Equal(uint64(0))))
	}
}

func TestBytesPool_Alloc_1(t *testing.T) {
	bp := NewBytesPool(32, 1024, 1024)
	data := bp.Alloc(5)
	Assert(t, len(data), Equal(5))
	Assert(t, cap(data), Equal(32))
	bp.Release(data)
}

func TestBytesPool_Alloc_2(t *testing.T) {
	bp := NewBytesPool(32, 1024, 1024)
	data := bp.Alloc(2046)
	Assert(t, len(data), Equal(2046))
	Assert(t, cap(data), Equal(2046))
	bp.Release(data)
}

func TestBytesPool_Release_1(t *testing.T) {
	bp := NewBytesPool(128, 1024, 1024)
	mem := bp.Alloc(64)
	go func() {
		defer func() {
			Assert(t, recover(), NilVal())
		}()
		bp.Release(mem)
		bp.Release(mem)
	}()
}

func BenchmarkBytesPool_AllocAndRelease(b *testing.B) {
	bp := NewBytesPool(128, 1024, 64*1024)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bp.Release(bp.Alloc(1024))
		}
	})
}

func Benchmark_AllocAndRelease(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var x []byte
		for pb.Next() {
			x = make([]byte, 1024)
		}
		x = x[:0]
	})
}
