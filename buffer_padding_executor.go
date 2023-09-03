package uidgenerator

import (
	"errors"
	"sync/atomic"
	"time"
)

/*
bufferPaddingExecutor
Represents an executor for padding ringBuffer.
There are two kinds of executors: one for scheduled padding, the other for padding immediately.
*/

type bufferPaddingExecutor struct {
	// Whether buffer padding is running
	running atomic.Bool
	// We can borrow UIDs from the future, here store the last second we have consumed.
	lastSecond *paddedAtomicLong
	// ringBuffer
	ringBuffer *ringBuffer
	// bufferedUidProvider
	uidProvider bufferedUidProvider
	// Schedule interval Unit as seconds
	scheduleInterval int64

	stop chan struct{}
}

/*
newBufferPaddingExecutor
Constructor with ringBuffer, bufferedUidProvider, and whether you use schedule padding

ringBuffer ringBuffer
uidProvider bufferedUidProvider
usingSchedule bool
*/
func newBufferPaddingExecutor(ringBuffer *ringBuffer, uidProvider bufferedUidProvider, usingSchedule bool) *bufferPaddingExecutor {
	bufferPaddingExecutor := bufferPaddingExecutor{
		running:     atomic.Bool{},
		lastSecond:  newPaddedAtomicLong(time.Now().Unix()),
		ringBuffer:  ringBuffer,
		uidProvider: uidProvider,
		stop:        make(chan struct{}),
	}
	return &bufferPaddingExecutor
}

// Padding buffer fill the slots until to catch the cursor
func (b *bufferPaddingExecutor) paddingBuffer() {
	//log.Printf("Ready to padding buffer lastSecond:%d. %s", b.lastSecond.Load(), b.ringBuffer.string())
	// is still running
	if !b.running.CompareAndSwap(false, true) {
		//log.Printf("Padding buffer is still running. %s", b.ringBuffer.string())
		return
	}

	// fill the rest slots until to catch the cursor
	isFullRingBuffer := false
	for !isFullRingBuffer {
		provideIds := b.uidProvider.provide(b.lastSecond.Add(1))
		for i := 0; i < len(provideIds); i++ {
			if isFullRingBuffer = !b.ringBuffer.put(provideIds[i]); isFullRingBuffer {
				break
			}
		}
		b.uidProvider.recycle(provideIds)
	}

	// not running now
	b.running.CompareAndSwap(true, false)
	//log.Printf("end to padding buffer lastSecond:%d. %s", b.lastSecond.Load(), b.ringBuffer.string())
}

func (b *bufferPaddingExecutor) asyncPadding() {
	go b.paddingBuffer()
}

// Start executors such as schedule
func (b *bufferPaddingExecutor) start() {
	if b.scheduleInterval != 0 {
		go func() {
			ticker := time.NewTicker(time.Second * time.Duration(b.scheduleInterval))
		LOOP:
			for {
				select {
				case <-ticker.C:
					b.paddingBuffer()
				case <-b.stop:
					break LOOP
				}
			}
		}()
	}
}

// Shutdown executors
func (b *bufferPaddingExecutor) shutdown() {
	b.stop <- struct{}{}
}

func (b *bufferPaddingExecutor) setScheduleInterval(scheduleInterval int64) error {
	if scheduleInterval <= 0 {
		return errors.New("schedule interval must positive")
	}
	b.scheduleInterval = scheduleInterval
	return nil
}
