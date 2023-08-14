package uidgenerator

import "sync"

type defaultBufferPidProvider struct {
	// slice pool for nextIdsForOneSecond, size is (uidGenerator.bitsAllocator.MaxSequence + 1)
	sliceCap      int
	slicePool     sync.Pool
	bitsAllocator *bitsAllocator
	epochSeconds  int64
	workerId      int64
}

func newDefaultBufferPidProvider(sliceCap int, bitsAllocator *bitsAllocator, epochSeconds, workerId int64) *defaultBufferPidProvider {
	return &defaultBufferPidProvider{
		sliceCap: sliceCap,
		slicePool: sync.Pool{New: func() any {
			return make([]int64, sliceCap)
		}},
		bitsAllocator: bitsAllocator,
		epochSeconds:  epochSeconds,
		workerId:      workerId,
	}

}

// Get the UIDs in the same specified second under the max sequence
func (d *defaultBufferPidProvider) provide(momentInSecond int64) []int64 {
	// get result list size of (max sequence + 1)
	uidList := d.slicePool.Get().([]int64)
	// Allocate the first sequence of the second, the others can be calculated with the offset
	firstSeqUid := d.bitsAllocator.allocate(momentInSecond-d.epochSeconds, d.workerId, 0)
	for offset := int64(0); offset < int64(d.sliceCap); offset++ {
		uidList[offset] = firstSeqUid + offset
	}
	return uidList
}

// recycle
func (d *defaultBufferPidProvider) recycle(list []int64) {
	d.slicePool.Put(list)
}
