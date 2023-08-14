package uidgenerator

import (
	"errors"
)

/*
Allocate 64 bits for the UID(long)<br>
sign (fixed 1bit) -> deltaSecond -> workerId -> sequence(within the same second)
*/

const (
	// totalBits Total 64 bits
	totalBits = 1 << 6
	// signBits sign
	signBits = 1
)

type bitsAllocator struct {
	// Bits for [sign-> second-> workId-> sequence]
	SignBits      int
	TimestampBits int
	WorkerIdBits  int
	SequenceBits  int
	// Max value for timestamp & workId & sequence
	MaxDeltaSeconds int64
	MaxWorkerId     int64
	MaxSequence     int64
	// Shift for timestamp & workerId
	TimestampShift int
	WorkerIdShift  int
}

/*
newBitsAllocator Constructor with timestampBits, workerIdBits, sequenceBits<br>
The highest bit used for sign, so <code>63</code> bits for timestampBits, workerIdBits, sequenceBits
*/
func newBitsAllocator(timestampBits, workerIdBits, sequenceBits int) (*bitsAllocator, error) {
	// make sure allocated 64 bits
	allocatorTotalBits := signBits + timestampBits + workerIdBits + sequenceBits
	if allocatorTotalBits != totalBits {
		return nil, errors.New("allocate not enough 64 bits")
	}
	/*
		initialize bits
		initialize max value
		initialize shift
	*/
	return &bitsAllocator{
		SignBits:        signBits,
		TimestampBits:   timestampBits,
		WorkerIdBits:    workerIdBits,
		SequenceBits:    sequenceBits,
		MaxDeltaSeconds: ^(-1 << timestampBits),
		MaxWorkerId:     ^(-1 << workerIdBits),
		MaxSequence:     ^(-1 << sequenceBits),
		TimestampShift:  workerIdBits + sequenceBits,
		WorkerIdShift:   sequenceBits,
	}, nil
}

/*
Allocate bits for UID according to delta seconds & workerId & sequence<br>
<b>Note that: </b>The highest bit will always be 0 for sign
*/
func (b *bitsAllocator) allocate(deltaSeconds, workerId, sequence int64) int64 {
	return (deltaSeconds << b.TimestampShift) | (workerId << b.WorkerIdShift) | sequence
}
