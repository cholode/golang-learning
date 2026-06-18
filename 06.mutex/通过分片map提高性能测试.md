### 通过分片的方式降低锁冲突从而提高性能



在mutex目录下输入指令

```shell
 go test '-bench=.' '-count=1' .
```

KF
BenchmarkSingleLock_Serial是串行单map

BenchmarkSharded_Serial-20  是串行分片map

BenchmarkSingleLock_Parallel-20 是并行单map

BenchmarkSharded_Parallel-20 是并行分片map

测试结果

```shell
goos: windows
goarch: 386
pkg: example.com/mutex
cpu: 13th Gen Intel(R) Core(TM) i5-13600KF
BenchmarkSingleLock_Serial-20               6079            197252 ns/op
BenchmarkSharded_Serial-20                  6343            183847 ns/op
BenchmarkSingleLock_Parallel-20         16647152                73.76 ns/op
BenchmarkSharded_Parallel-20            36941376                28.54 ns/op
```

可以看到串行的情况下，两者性能差距不明显

并行的情况下，分片map是单map的三倍，合理设置map数量能够提高非常多的性能
