package uidgenerator

import (
	"log"
)

/*
CachedUidGenerator
Represents a cached implementation of UidGenerator combines
from DefaultUidGenerator, based on a lock free ringBuffer

The properties you can specify as below:
boostPower: ringBuffer size boost for a power of 2, Sample: boostPower is 3, it means the buffer size
			will be (bitsAllocator.MaxSequence + 1) << 3, Default as defaultBoostPower
paddingFactor: Represents a percent value of (0 - 100). When the count of rest available UIDs reach the
				threshold, it will trigger padding buffer. Default as defaultPaddingPercent
	  			Sample: paddingFactor=20, bufferSize=1000 -> threshold=1000 * 20 /100,
				padding buffer will be triggered when tail-cursor<threshold
scheduleInterval: Padding buffer in a schedule, specify padding buffer interval, Unit as second
RejectedPutBufferHandler: Policy for rejected put buffer. Default as discard put request, just do logging
RejectedTakeBufferHandler: Policy for rejected take buffer. Default as return false, just do logging
*/

const (
	defaultBoostPower = 3
)

type CachedUidGenerator struct {
	DefaultUidGenerator
	// ringBuffer size grow arg
	boostPower int
	// padding
	paddingFactor int
	// schedule interval
	scheduleInterval int64

	rejectedPutBufferHandler  RejectedPutBufferHandler
	rejectedTakeBufferHandler RejectedTakeBufferHandler
	/* ringBuffer */
	ringBuffer            *ringBuffer
	bufferPaddingExecutor *bufferPaddingExecutor
}

type OptionCached func(cachedUidGenerator *CachedUidGenerator)

func WithBoostPower(boostPower int) OptionCached {
	return func(cachedUidGenerator *CachedUidGenerator) {
		cachedUidGenerator.boostPower = boostPower
	}
}

func WithPaddingFactor(paddingFactor int) OptionCached {
	return func(cachedUidGenerator *CachedUidGenerator) {
		cachedUidGenerator.paddingFactor = paddingFactor
	}
}

func WithScheduleInterval(scheduleInterval int64) OptionCached {
	return func(cachedUidGenerator *CachedUidGenerator) {
		cachedUidGenerator.scheduleInterval = scheduleInterval
	}
}

func WithRejectedPutBufferHandler(rejectedPutBufferHandler RejectedPutBufferHandler) OptionCached {
	return func(cachedUidGenerator *CachedUidGenerator) {
		cachedUidGenerator.rejectedPutBufferHandler = rejectedPutBufferHandler
	}
}

func WithRejectedTakeBufferHandler(rejectedTakeBufferHandler RejectedTakeBufferHandler) OptionCached {
	return func(cachedUidGenerator *CachedUidGenerator) {
		cachedUidGenerator.rejectedTakeBufferHandler = rejectedTakeBufferHandler
	}
}

func NewCachedUidGenerator(defaultUidGenerator *DefaultUidGenerator, opts ...OptionCached) (*CachedUidGenerator, error) {
	uidGenerator := CachedUidGenerator{
		DefaultUidGenerator: *defaultUidGenerator,
		boostPower:          defaultBoostPower,
		paddingFactor:       defaultPaddingPercent,
		scheduleInterval:    0,
	}
	for _, opt := range opts {
		opt(&uidGenerator)
	}
	// initialize ringBuffer
	bufferSize := int(defaultUidGenerator.bitsAllocator.MaxSequence+1) << uidGenerator.boostPower
	ringBuffer, err := newRingBuffer(bufferSize, uidGenerator.paddingFactor)
	if err != nil {
		return nil, err
	}
	uidGenerator.ringBuffer = ringBuffer
	log.Printf("initialized ring buffer size:%d, paddingFactor:%d", bufferSize, uidGenerator.paddingFactor)
	// initialize RingBufferPaddingExecutor
	usingSchedule := uidGenerator.scheduleInterval != 0
	bufferPaddingExecutor := newBufferPaddingExecutor(ringBuffer, newDefaultBufferPidProvider(int(uidGenerator.bitsAllocator.MaxSequence+1), uidGenerator.bitsAllocator, uidGenerator.epochSeconds, uidGenerator.workerId), usingSchedule)
	if usingSchedule {
		err := bufferPaddingExecutor.setScheduleInterval(uidGenerator.scheduleInterval)
		if err != nil {
			return nil, err
		}
	}
	log.Printf("initialized bufferPaddingExecutor. Using schdule:%v, interval:%d", usingSchedule, uidGenerator.scheduleInterval)
	uidGenerator.bufferPaddingExecutor = bufferPaddingExecutor
	uidGenerator.ringBuffer.bufferPaddingExecutor = bufferPaddingExecutor
	// set rejected put/take handle policy
	if uidGenerator.rejectedPutBufferHandler != nil {
		uidGenerator.bufferPaddingExecutor.ringBuffer.rejectedPutBufferHandler = uidGenerator.rejectedPutBufferHandler
	}
	if uidGenerator.rejectedTakeBufferHandler != nil {
		uidGenerator.bufferPaddingExecutor.ringBuffer.rejectedTakeBufferHandler = uidGenerator.rejectedTakeBufferHandler
	}
	// fill in all slots of the ringBuffer
	uidGenerator.bufferPaddingExecutor.paddingBuffer()
	// start buffer padding threads
	uidGenerator.bufferPaddingExecutor.start()
	return &uidGenerator, nil
}

func (c *CachedUidGenerator) GetUID() (int64, error) {
	return c.ringBuffer.take()
}

func (c *CachedUidGenerator) ParseUID(uid int64) string {
	return c.DefaultUidGenerator.ParseUID(uid)
}
