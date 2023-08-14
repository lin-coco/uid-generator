package uidgenerator

type RejectedPutBufferHandler interface {
	RejectPutBuffer(ringBuffer *ringBuffer, uid int64)
}
