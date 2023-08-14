package uidgenerator

type RejectedTakeBufferHandler interface {
	RejectTakeBuffer(ringBuffer *ringBuffer)
}
