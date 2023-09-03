package uidgenerator

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

/*
ringBuffer
Represents a ring buffer based on array.
Using array could improve read element performance due to the CUP cache line. To prevent
the side effect of False Sharing, paddedAtomicLong is using on 'tail' and 'cursor'.

A ring buffer is consisted of:
slots:each element of the array is a slot, which is being set with a UID
flags:flag array corresponding the same index with the slots, indicates whether you can take or put slot
tail:a sequence of the max slot position to produce
cursor:a sequence of the min slot position to consume
*/

const (
	startPoint            = -1
	canPutFlag            = 0
	canTakeFlag           = 1
	defaultPaddingPercent = 50
)

type ringBuffer struct {
	// The size of ringBuffer's slots, each slot hold a UID
	bufferSize int
	indexMask  int64
	slots      []int64
	flags      []paddedAtomicLong
	// Tail: last position sequence to produce
	tail *paddedAtomicLong
	// Cursor: current position sequence to consume
	cursor *paddedAtomicLong
	// Threshold for trigger padding buffer
	paddingThreshold int
	// Reject put/take buffer handle policy
	rejectedPutBufferHandler  RejectedPutBufferHandler
	rejectedTakeBufferHandler RejectedTakeBufferHandler
	// Executor of padding buffer
	bufferPaddingExecutor *bufferPaddingExecutor

	mutex sync.Mutex
}

/*
newRingBuffer
Constructor with buffer size & padding factor
bufferSize must be positive & a power of 2
paddingFactor percent in (0 - 100). When the count of rest available UIDs reach the threshold, it will trigger padding buffer<br>
Sample: paddingFactor=20, bufferSize=1000 -> threshold=1000 * 20 /100,
padding buffer will be triggered when tail-cursor<threshold
*/
func newRingBuffer(bufferSize, paddingFactor int) (*ringBuffer, error) {
	flags := newSlicePaddedAtomicLong(canPutFlag, bufferSize)
	return &ringBuffer{
		bufferSize:                bufferSize,
		indexMask:                 int64(bufferSize) - 1,
		slots:                     make([]int64, bufferSize),
		flags:                     flags,
		paddingThreshold:          bufferSize * paddingFactor / 100,
		tail:                      newPaddedAtomicLong(startPoint),
		cursor:                    newPaddedAtomicLong(startPoint),
		rejectedPutBufferHandler:  &DiscardPutBufferHandler{},
		rejectedTakeBufferHandler: &ErrorTakeBufferHandler{},
	}, nil
}

/*
Put an UID in the ring & tail moved
We use 'mutex' to guarantee the UID fill in slot & publish new tail sequence as atomic operations

It is recommended to put UID in a serialize way, because we once batch generate a series UIDs and put
the one by one into the buffer, so it is unnecessary put in multi-threads

uid
return false means that the buffer is full, apply RejectedPutBufferHandler
*/
func (r *ringBuffer) put(uid int64) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	currentTail := r.tail.Load()
	currentCursor := r.cursor.Load()
	// tail catches the cursor, means that you can't put any cause of ringBuffer is full
	if currentCursor == startPoint {
		currentCursor = 0
	}
	if distance := currentTail - currentCursor; distance == int64(r.bufferSize)-1 {
		r.rejectedPutBufferHandler.RejectPutBuffer(r, uid)
		return false
	}
	// 1. pre-check whether the flag is CAN_PUT_FLAG
	nextTailIndex := r.calSlotIndex(currentTail + 1)
	if r.flags[nextTailIndex].Load() != canPutFlag {
		r.rejectedPutBufferHandler.RejectPutBuffer(r, uid)
		return false
	}
	// 2. put UID in the next slot
	// 3. update next slot's flag to CAN_TAKE_FLAG
	// 4. publish tail with sequence increase by one
	r.slots[nextTailIndex] = uid
	r.flags[nextTailIndex].Store(canTakeFlag)
	r.tail.Add(1)
	// The atomicity of operations above, guarantees by 'mutex'. In another word,
	// the take operation can't consume the UID we just put, until the tail is published(tail.Add(1))
	return true
}

/*
Take an UID of the ring at the next cursor, this is a lock free operation by using atomic cursor

Before getting the UID, we also check whether reach the padding threshold,
the padding buffer operation will be triggered in another thread
If there is no more available UID to be taken, the specified RejectedTakeBufferHandler will be applied

return UID
@throws IllegalStateException if the cursor moved back
*/
func (r *ringBuffer) take() (int64, error) {
	// spin get next available cursor
	currentCursor := r.cursor.Load()
	nextCursor := func() int64 {
		for {
			oldV := r.cursor.Load()
			newV := func(oldV int64) int64 {
				if oldV == r.tail.Load() {
					return oldV
				}
				return oldV + 1
			}(oldV)
			if r.cursor.CompareAndSwap(oldV, newV) {
				return newV
			}
		}
	}()
	// check for safety consideration, it never occurs
	if nextCursor < currentCursor {
		return 0, errors.New("cursor can't move back")
	}
	// trigger padding in an async-mode if reach the threshold
	currentTail := r.tail.Load()
	if currentTail-nextCursor < int64(r.paddingThreshold) {
		//log.Printf("Reach the padding threshold:%d. tail:%d, cursor:%d, rest:%d", r.paddingThreshold, currentTail, nextCursor, currentTail-nextCursor)
		r.bufferPaddingExecutor.asyncPadding()
	}
	// cursor catch the tail, means that there is no more available UID to take
	if nextCursor == currentCursor {
		r.rejectedTakeBufferHandler.RejectTakeBuffer(r)
		return 0, errors.New("too frequent acquisition, no more available UID to take")
		//return 0, nil
	}
	// 1. check next slot flag is CAN_TAKE_FLAG
	nextCursorIndex := r.calSlotIndex(nextCursor)
	if r.flags[nextCursorIndex].Load() != canTakeFlag {
		return 0, errors.New("cursor not in can take status")
	}
	// 2. get UID from next slot
	// 3. set next slot flag as CAN_PUT_FLAG.
	uid := r.slots[nextCursorIndex]
	r.flags[nextCursorIndex].Store(canPutFlag)
	// Note that: Step 2,3 can not swap. If we set flag before get value of slot, the producer may overwrite the
	// slot with a new UID, and this may cause the consumer take the UID twice after walk a round the ring
	return uid, nil
}

// Calculate slot index with the slot sequence (sequence % bufferSize)
func (r *ringBuffer) calSlotIndex(sequence int64) int {
	return int(sequence % r.indexMask)
}

func (r *ringBuffer) string() string {
	bufferSize := r.bufferSize
	tailLoad := r.tail.Load()
	cursorLoad := r.cursor.Load()
	paddingThreshold := r.paddingThreshold
	return fmt.Sprintf("ringBuffer [bufferSize=%v, tail=%v, cursor=%v, paddingThreshold=%v].", bufferSize, tailLoad, cursorLoad, paddingThreshold)
}

// DiscardPutBufferHandler Discard policy for RejectedPutBufferHandler, we just do logging
type DiscardPutBufferHandler struct {
}

func (d *DiscardPutBufferHandler) RejectPutBuffer(ringBuffer *ringBuffer, uid int64) {
	log.Printf("Rejected putting buffer for uid:%d. %s", uid, ringBuffer.string())
}

// ErrorTakeBufferHandler Policy for RejectedTakeBufferHandler,  after logging
type ErrorTakeBufferHandler struct {
}

func (d *ErrorTakeBufferHandler) RejectTakeBuffer(ringBuffer *ringBuffer) {
	log.Printf("Rejected take buffer. %s", ringBuffer.string())
}
