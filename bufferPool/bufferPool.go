package bufferPool

import "sync"

type BufferPool struct {
	bufPool *sync.Pool
}

func NewBufferPool() *BufferPool {
	return &BufferPool{bufPool: &sync.Pool{
		New: func() interface{} {
			return make([]byte, 16*1024)
		},
	}}
}

func (bp BufferPool) Get() []byte  { return bp.bufPool.Get().([]byte) }
func (bp BufferPool) Put(v []byte) { bp.bufPool.Put(v) }
