# uid-generator

百度UidGenerator是Java实现的, 基于Snowflake算法的唯一ID生成器。

本项目使用go语言重构，100%还原所有特性。

uid-generator是在Snowflake基础上优化的算法，目的是解决时钟回拨，重启时生成id重复，提高并发量等

## uid-generator设计原理

官方介绍：[百度UidGenerator](https://github.com/baidu/uid-generator/blob/master/README.md#uidgenerator)

## uid-generator性能测试

官方测试：[百度UidGenerator](https://github.com/baidu/uid-generator/blob/master/README.md#tips)

## uid-generator有哪些优点和缺点？

uid-generator特性和优点：
- 自定义位数划分和初始化策略
- 借用未来时间，避免运行时时钟回拨，并解决天然存在的并发限制
- 使用数据库自增workerId分配器，避免重启时生成相同的id
- 缓存id，应对突发流量，填充对象避免伪共享，性能优越

uid-generator缺点：
- 生成时间会与当前时间不符
- 使用缓存模式可能会得到大量连续的id
- 参数配置复杂，需要充分理解其原理才可自定义配置

## 快速使用

建议使用前，先充分理解uid-generator设计原理和参数配置

- 默认模式生成器
```go
func main() {

	workerIdAssigner, err := uidgenerator.NewDisposableWorkerIdAssigner("root:syr1120@xys.com@tcp(127.0.0.1:3306)/uid_generator?charset=utf8mb4&parseTime=true&loc=Local")
	if err != nil {
		log.Fatal(err)
	}
	defaultUidGenerator, err := uidgenerator.NewDefaultUidGenerator(workerIdAssigner)
	if err != nil {
		log.Fatal(err)
	}
	uid, err := defaultUidGenerator.GetUID()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(defaultUidGenerator.ParseUID(uid))
}
```

- 缓冲模式生成器
```go
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
	uid, err := cachedUidGenerator.GetUID()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(defaultUidGenerator.ParseUID(uid))
}
```

- 性能测试
```go
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
	uid, err := cachedUidGenerator.GetUID()
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now()
	for i := 0; i < 10000000; i++ {
		uid, _ = cachedUidGenerator.GetUID()
	}
	fmt.Println(time.Since(now))
	fmt.Println(defaultUidGenerator.ParseUID(uid))
}
```

## 用go重构uid-generator项目的挑战和优化

首先必须要熟练两种编程语言，Java是一种面向对象的语言，而Go是一种更趋向于结构化编程的语言。
这意味着你在能看懂java源码的同时，需要用go重新设计和实现很多代码段，以适应go的特点。
java代码中充斥着许多面向对象的特点，继承、多态等go语言不支持的特性，需要转化位组合等大量的代替。
我在快速使用章节粘贴的代码就可以看出我是把继承重构成组合了。

java拥有庞大的标准库，这意味着go可能要自己实现一些java库，在此项目中的java线程池，定时任务。
go没有线程池，但幸运的是，go的协程非常轻量，在综合的考虑后，我没有使用三方协程池，而是直接随起随用协程。
定时任务我是使用的`time.NewTicker()`来实现。

在每次生成固定数量的id后需要填充到环形缓冲中，我发现java中每次都是`new ArrayList()`，并且非常的大默认有8192大小，
这意味着内存分配和垃圾回收有更多的性能损耗。但是我在go语言中实现这一块时，我没有每次都是创建一个集合，
而是使用`sync.Pool`对象池缓冲slice，有效的降低了内存和垃圾回收的负担。



