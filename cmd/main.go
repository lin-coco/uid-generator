package main

import (
	"fmt"
	"log"
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
	//now := time.Now()
	//for i := 0; i < 10000000; i++ {
	//	uid, _ = cachedUidGenerator.GetUID()
	//}
	//fmt.Println(time.Since(now))
	uid, err := cachedUidGenerator.GetUID()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(defaultUidGenerator.ParseUID(uid))
}
