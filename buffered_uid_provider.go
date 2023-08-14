package uidgenerator

type bufferedUidProvider interface {
	// Provide Provides UID in one second
	provide(momentInSecond int64) []int64
	recycle(list []int64)
}
