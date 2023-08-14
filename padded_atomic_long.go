package uidgenerator

import "sync/atomic"

/*
Represents a padded {@link AtomicLong} to prevent the FalseSharing problem.

The CPU cache line commonly be 64 bytes, here is a sample of cache line after padding.
Baidu's java paddedAtomicLong: 64 bytes = 8 bytes (object reference) + 6 * 8 bytes (padded long) + 8 bytes (a long value)
But in go: 64 bytes = 56 bytes (padded long) + 8 bytes (a long value)

	func main() {
		pal := paddedAtomicLong{}
		size := unsafe.Sizeof(pal)
		fmt.Printf("Size of paddedAtomicLong: %d bytes\n", size)
	}

print:

	Size of paddedAtomicLong: 64 bytes

*/

type paddedAtomicLong struct {
	// Padded 48 bytes
	pad [56]byte
	atomic.Int64
}

func newPaddedAtomicLong(initialValue int64) *paddedAtomicLong {
	p := &paddedAtomicLong{
		pad:   [56]byte{},
		Int64: atomic.Int64{},
	}
	p.Store(initialValue)
	return p
}
