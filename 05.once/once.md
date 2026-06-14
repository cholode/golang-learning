### 源码分析

```go
type Once struct {
	_ noCopy

	done atomic.Bool
	m    Mutex
}

func (o *Once) Do(f func()) {//先进行一次简单的验证
	
	if !o.done.Load() {
		o.doSlow(f)
	}
}

func (o *Once) doSlow(f func()) {
	o.m.Lock()//为防止多个协程同时用一个once，所以先上锁
	defer o.m.Unlock()
	if !o.done.Load() {//避免在短验证的时候就有协程把Once用了，且把锁还了，再次验证
		defer o.done.Store(true)//不管有没有成功都要设为用过
		f()
	}
}


int
```



### Exp1 Once的使用

```go
func Print1() {
	fmt.Println("我是1号")
}

func Print2() {
	fmt.Println("我是2号")
}

func Exp1() {
	var oc sync.Once
	oc.Do(Print1)
	oc.Do(Print2)
}

int
```

        这段代码只有Print1的结果打出来了，同一个once全局的函数只有第一个使用这个once的时候能够生效


