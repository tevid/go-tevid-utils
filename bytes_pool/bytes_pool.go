package bytes_pool

import (
	"reflect"
	"runtime"
	"sync/atomic"
	"unsafe"
)

// 基于字节数组的对象池
// 目的：分配可复用的对象，降低小对象频繁分配回收增加系统压力

const (
	DEFAULT_ENTITY_LIST_LEN = 20
	DEFAULT_INCREATE_FACTOR = 2
	INIT_POS                = 1
	POS_SHIFT               = 32
)

//bytes池
type (
	BytesPool struct {
		initEntitySize int
		maxEntitySize  int
		entityList     []entity
	}

	//池实体
	entity struct {
		esize    int     //实体所属大小
		memory   []byte  //公共内存空间
		chunks   []chunk //数据块链表
		beginPtr uintptr //开始指针
		endPtr   uintptr //结束指针
		pos      uint64  //当前内存地址位置
	}

	//数据块
	chunk struct {
		data    []byte
		flag    uint64
		nextPos uint64
	}
)

//构建对象池
func NewBytesPool(initSize, maxSize, memorySize int) *BytesPool {
	pool := &BytesPool{
		initEntitySize: initSize,
		maxEntitySize:  maxSize,
		entityList:     make([]entity, 0, DEFAULT_ENTITY_LIST_LEN),
	}

	//构造若干大小的连续分配区
	for bytesSize := initSize; bytesSize <= maxSize && bytesSize <= memorySize; bytesSize *= DEFAULT_INCREATE_FACTOR {

		//块长度
		chunkLen := memorySize / bytesSize
		//构造bytes实体
		entity := entity{
			esize:  bytesSize,
			memory: make([]byte, memorySize),
			chunks: make([]chunk, chunkLen),
			pos:    INIT_POS << POS_SHIFT, //从 INIT_POS 开始
		}
		for i := 0; i < chunkLen; i++ {
			posIdx := i + INIT_POS
			chk := &entity.chunks[i]

			//从entiry的内存区域中连续划分空间指向内部的chunk块
			chk.data = entity.memory[i*bytesSize : (i+1)*bytesSize : (i+1)*bytesSize]

			//处理最后一个内存块
			if i < chunkLen-1 {
				//左移，构成连续的内存块
				chk.nextPos = uint64(posIdx+1) << POS_SHIFT
			} else {
				entity.beginPtr = uintptr(unsafe.Pointer(&entity.memory[0]))
				entity.endPtr = uintptr(unsafe.Pointer(&chk.data[0]))
			}
		}
		pool.entityList = append(pool.entityList, entity)
	}
	return pool
}

//从对象池中分配字节数为size大小的可复用字节数值
func (p *BytesPool) Alloc(size int) []byte {
	if size > p.maxEntitySize {
		return make([]byte, size)
	}
	//遍历存储实体链表，查找可满足分配的实体
	for i := 0; i < len(p.entityList); i++ {
		//满足可分配的条件
		if size <= p.entityList[i].esize {
			data := p.entityList[i].pop()
			if data != nil {
				return data[:size]
			}
			break
		}
	}
	return make([]byte, size)
}

//把字节数值归还回池中
func (p *BytesPool) Release(data []byte) {
	//因为Alloc阶段可能出现data[:size]的情况， 所以这里计算的长度不能使用len
	size := cap(data)
	//遍历存储实体链表，查找可满足归还的实体
	for i := 0; i < len(p.entityList); i++ {
		//满足可归还的条件
		if p.entityList[i].esize == size {
			p.entityList[i].push(data)
			break
		}
	}
}

func (e *entity) pop() []byte {
	for {
		currentPos := atomic.LoadUint64(&e.pos)
		if currentPos == 0 { //超出了实体的存储范围了
			return nil
		}
		//根据内存空间地址，获取当前内存块
		posIndex := currentPos>>32 - INIT_POS
		chk := &e.chunks[posIndex]
		//下一个连续空间的地址位置
		nextPos := atomic.LoadUint64(&chk.nextPos)
		if atomic.CompareAndSwapUint64(&e.pos, currentPos, nextPos) {
			//移出chk
			atomic.StoreUint64(&chk.nextPos, 0)
			return chk.data
		}
		runtime.Gosched()
	}
}

func (e *entity) push(data []byte) {
	//归还的时候，获取data的指针
	ptr := (*reflect.SliceHeader)(unsafe.Pointer(&data)).Data
	//检查指针范围的有效性
	if e.beginPtr <= ptr && ptr <= e.endPtr {
		posIndex := (ptr - e.beginPtr) / uintptr(e.esize)
		chk := e.chunks[posIndex]
		if chk.nextPos != 0 {
			panic("chunk had been release")
		}
		chk.flag++
		updatePos := uint64(posIndex+1)<<32 + chk.flag
		for {
			currentPos := atomic.LoadUint64(&e.pos)
			//归还的块，指向当前实体分配位置
			atomic.StoreUint64(&chk.nextPos, currentPos)
			if atomic.CompareAndSwapUint64(&e.pos, currentPos, updatePos) {
				break
			}
			runtime.Gosched()
		}
	}

}
