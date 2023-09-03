package uidgenerator

import (
	"testing"
	"time"
)

func TestPadding(t *testing.T) {
	workerIdAssigner, err := NewDisposableWorkerIdAssigner("root:syr1120@xys.com@tcp(127.0.0.1:3306)/uid_generator?charset=utf8mb4&parseTime=true&loc=Local")
	if err != nil {
		t.Error(err)
	}
	defaultUidGenerator, err := NewDefaultUidGenerator(workerIdAssigner)
	if err != nil {
		t.Error(err)
	}
	cachedUidGenerator, err := NewCachedUidGenerator(defaultUidGenerator)
	if err != nil {
		t.Error(err)
	}
	var id int64
	now := time.Now()
	for i := 0; i < 20000000; i++ {
		id, err = cachedUidGenerator.GetUID()
		for id == 0 || err != nil {
			id, err = cachedUidGenerator.GetUID()
		}
	}
	t.Log(time.Since(now))
}
