### goroutine+channel的实验

        在普通的函数中<-chan会导致进程阻塞等待直到chan中传入信息

```go
ch := make(chan string)
go func() {
    time.Sleep(time.Second * 2)
    ch <- "hello world"
}()
fmt.Println("waiting for channel...")
msg := <-ch
fmt.Println("now, i get the channel")

fmt.Println(msg)
```

        在上面的代码中，因为sleep而暂停的函数不会往ch中传入信息，随后优先输出waiting...然后再在msg:=<-ch处阻塞等待函数的运行,在sleep结束后就会接着运行后面的部分

        如果将msg:<-ch放在了go func的前面，比如下面这样，就会发生死锁然后直接报错退出

```go
	ch := make(chan string)
	msg := <-ch
	go func() {
	    time.Sleep(time.Second * 2)
		ch <- "hello world"
		
		fmt.Println("goroutine exit successful")
	}()
	
	fmt.Println(msg)
```

        对于项目应用的思考：对于这个组合，可以用于不同协程中的信息传递，可以先进行信息流的设计，然后再进行实现，设计信息流应该要保证先后时序
