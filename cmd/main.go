package main

import (
	"fmt"
	"log"
	"time"
	"uidgenerator"
)

func main() {

	workerIdAssigner, err := uidgenerator.NewDisposableWorkerIdAssigner("root:syr1120@xys.com@tcp(127.0.0.1:3306)/uid_generator?charset=utf8mb4&parseTime=true&loc=Local")
	if err != nil {
		log.Fatal(err)
	}
	defaultUidGenerator, err := uidgenerator.NewDefaultUidGenerator(workerIdAssigner)
	if err != nil {
		log.Fatal(err)
	}
	cachedUidGenerator, err := uidgenerator.NewCachedUidGenerator(defaultUidGenerator)
	if err != nil {
		log.Fatal(err)
	}
	//uid, err := cachedUidGenerator.GetUID()
	//if err != nil {
	//	log.Fatal(err)
	//}
	var id int64
	now := time.Now()
	for i := 0; i < 20000000; i++ {
		id, err = cachedUidGenerator.GetUID()
		for id == 0 || err != nil {
			id, err = cachedUidGenerator.GetUID()
		}
	}
	fmt.Println(time.Since(now))
	//uid, err := cachedUidGenerator.GetUID()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Println(defaultUidGenerator.ParseUID(uid))
}
