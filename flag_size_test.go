package uidgenerator

import (
	"testing"
	"unsafe"
)

func TestFlagSize(t *testing.T) {
	flag := *newPaddedAtomicLong(0)
	size := unsafe.Sizeof(flag)
	t.Log(size)
}
