package uidgenerator

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

/*
DefaultUidGenerator

# Represents an implementation of UidGenerator

The unique id has 64bits (uint64), default allocated as blow:
sign: The highest bit is 0
delta seconds: The next 28 bits, represents delta seconds since a customer epoch(2016-05-20 00:00:00.000).
Supports about 8.7 years until to 2024-11-20 21:24:16
worker id: The next 22 bits, represents the worker's id which assigns based on database, max id is about 420W
sequence: The next 13 bits, represents a sequence within the same second, max for 8192/s

The DefaultUidGenerator#parseUID(uint64) is a tool method to parse the bits

	+------+----------------------+----------------+-----------+
	| sign |     delta seconds    | worker node id | sequence  |
	+------+----------------------+----------------+-----------+
	  1bit          28bits              22bits         13bits

You can also specify the bits by custom setting.
timeBits: default as 28
workerBits: default as 22
seqBits: default as 13
epochStr: Epoch date string format 'yyyy-MM-dd'. Default as '2016-05-20'

The total bits must be 64 -1
*/
type DefaultUidGenerator struct {
	// Bits allocate
	timeBits   int
	workerBits int
	seqBits    int
	// Customer epoch, unit as second. For example 2016-05-20 (ms: 1463673600000)
	epochStr     string
	epochSeconds int64
	// Stable fields after DefaultUidGenerator initializing
	bitsAllocator *bitsAllocator
	workerId      int64
	// Volatile fields caused by nextId()
	sequence   int64
	lastSecond int64

	workerIdAssigner WorkerIdAssigner
}

type OptionDefault func(defaultUidGenerator *DefaultUidGenerator)

func WithBits(timeBits, workerBits, seqBits int) OptionDefault {
	return func(defaultUidGenerator *DefaultUidGenerator) {
		defaultUidGenerator.timeBits = timeBits
		defaultUidGenerator.workerBits = workerBits
		defaultUidGenerator.seqBits = seqBits
	}
}

func WithEpoch(epochStr string) OptionDefault {
	return func(defaultUidGenerator *DefaultUidGenerator) {
		defaultUidGenerator.epochStr = epochStr
	}
}

func NewDefaultUidGenerator(workerIdAssigner WorkerIdAssigner, opts ...OptionDefault) (*DefaultUidGenerator, error) {
	uidGenerator := DefaultUidGenerator{
		timeBits:   28,
		workerBits: 22,
		seqBits:    13,
		// Customer epoch, unit as second. For example 2023-05-20 (s: 1684540800) util 2031-11-21
		epochStr:         "2023-05-20",
		epochSeconds:     1684540800,
		sequence:         0,
		lastSecond:       -1,
		workerIdAssigner: workerIdAssigner,
	}
	for _, opt := range opts {
		opt(&uidGenerator)
	}

	// initialize bits allocator
	bitsAllocator, err := newBitsAllocator(uidGenerator.timeBits, uidGenerator.workerBits, uidGenerator.seqBits)
	if err != nil {
		return nil, err
	}
	uidGenerator.bitsAllocator = bitsAllocator
	// initialize worker id
	if uidGenerator.workerIdAssigner == nil {
		return nil, errors.New("workerIdAssigner is not allowed nil")
	}
	uidGenerator.workerIdAssigner = workerIdAssigner
	workerId, err := uidGenerator.workerIdAssigner.assignWorkerId()
	if err != nil {
		return nil, err
	}
	if workerId > bitsAllocator.MaxWorkerId {
		return nil, fmt.Errorf("worker id %d exceeds the max %d", workerId, bitsAllocator.MaxWorkerId)
	}
	uidGenerator.workerId = workerId
	return &uidGenerator, nil
}

func (d *DefaultUidGenerator) GetUID() (int64, error) {
	return d.nextId()
}

func (d *DefaultUidGenerator) ParseUID(uid int64) string {
	totalBits := totalBits
	signBits := d.bitsAllocator.SignBits
	timestampBits := d.bitsAllocator.TimestampBits
	workerIdBits := d.bitsAllocator.WorkerIdBits
	sequenceBits := d.bitsAllocator.SequenceBits
	// parse UID
	sequence := uint64(uid<<(totalBits-sequenceBits)) >> (totalBits - sequenceBits)
	workerId := uint64(uid<<(timestampBits+signBits)) >> (totalBits - workerIdBits)
	deltaSeconds := uint64(uid) >> (workerIdBits + sequenceBits)
	thatTime := time.UnixMilli((d.epochSeconds + int64(deltaSeconds)) * 1000)
	thatTimeStr := thatTime.Format("2006-01-02 15:04:05")
	return fmt.Sprintf("{\"uid\":\"%d\",\"binary\":\"%064s\",\"timestamp\":\"%s\",\"workerId\":\"%d\",\"sequence\":\"%d\"}", uid, strconv.FormatInt(uid, 2), thatTimeStr, workerId, sequence)
}

func (d *DefaultUidGenerator) nextId() (int64, error) {
	currentSecond := time.Now().Unix()
	// Clock moved backwards, refuse to generate uid
	if currentSecond < d.lastSecond {
		refusedSeconds := d.lastSecond - currentSecond
		return 0, fmt.Errorf("clock moved backwards. Refusing for %d seconds", refusedSeconds)
	}
	// At the same second, increase sequence
	if currentSecond == d.lastSecond {
		d.sequence = (d.sequence + 1) & d.bitsAllocator.MaxSequence
		// Exceed the max sequence, we wait the next second to generate uid
		if d.sequence == 0 {
			var err error
			currentSecond, err = d.getNextSecond(d.lastSecond)
			if err != nil {
				return 0, err
			}
		}
		// At the different second, sequence restart from zero
	} else {
		d.sequence = 0
	}
	d.lastSecond = currentSecond
	// Allocate bits for UID
	return d.bitsAllocator.allocate(currentSecond-d.epochSeconds, d.workerId, d.sequence), nil
}

func (d *DefaultUidGenerator) getNextSecond(lastTimestamp int64) (int64, error) {
	timestamp, err := d.getCurrentSecond()
	if err != nil {
		return 0, err
	}
	for timestamp <= lastTimestamp {
		timestamp, err = d.getCurrentSecond()
		if err != nil {
			return 0, err
		}
	}
	return timestamp, nil
}

func (d *DefaultUidGenerator) getCurrentSecond() (int64, error) {
	currentSecond := time.Now().Unix()
	if currentSecond-d.epochSeconds > d.bitsAllocator.MaxDeltaSeconds {
		return 0, fmt.Errorf("timestamp bits is exhausted. Refusing UID generate. Now: %d", currentSecond)
	}
	return currentSecond, nil
}
