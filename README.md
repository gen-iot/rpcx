# rpcx

Easy to use and developer friendly **RPC library**

![go report](https://goreportcard.com/badge/github.com/gen-iot/rpcx)
![license](https://img.shields.io/badge/license-MIT-brightgreen.svg)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://github.com/gen-iot/rpcx)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](https://github.com/gen-iot/rpcx/pulls)
[![Ask Me Anything !](https://img.shields.io/badge/Ask%20me-anything-1abc9c.svg)](https://github.com/gen-iot/rpcx/issues)

## Difference Between Other RPC Libraries

- Once **Callable** established, local and remote are both in **Full duplex** mode.
- No more **client** or **server** roles, **Callables** can **call/called** each others
- Lots of **Middlewares** and developer can **Custom** their own middleware!
- It works without any proto files , just define functions as usual.
- Based on asynchronous I/O [**(liblpc)**](https://github.com/gen-iot/liblpc) and provides synchronous call semantics
- **High Performace**: event drivend, msgpack(for serialize),reuse context,goroutine pool...


## Usage

```go
import "github.com/gen-iot/rpcx"
```

## Overview

- **Developer friendly**
- **Middlewares**: dump,proxy,recover,validate...

## Getting Started


### RPC Functions Formal

 **Remember :param `ctx`,`err` always required**

1. Both have `in`&`out`
```go
func Function(ctx rpcx.Context, in InType)(out OutType, err error)
```
2. Only `err` 

```go
func Function(ctx rpcx.Context)(err error)
```

3.  `out`  and `err`
```go
func Function(ctx rpcx.Context) (out OutType,err error)
```

4.  `in` and `err`
```go
func Function(ctx rpcx.Context, in InType)(err error)
```

### Create RPC

```go
rpc, err := rpcx.New()
```

### Close RPC

```go
err := rpc.Close()
```

### Register RPC Function

```go
rpc, err := rpcx.New()
std.AssertError(err, "new rpc")
rpc.RegFuncWithName(
    "hello", // manual specified func name
    func(ctx rpcx.Context, msg string) (string, error) {
        return "hello from server", nil
    })
```

### Connect To RPC

```go
sockAddr, err := liblpc.ResolveTcpAddr("127.0.0.1")
std.AssertError(err,"resolve addr")
callable := rpc.NewClientCallable(sockAddr,nil)
callable.Start()
```

### Add Exist Conn To RPC

```go
// `fd` must be a valid file descriptor
callable := rpc.NewConnCallable(fd, nil)
callable.Start()
```

### Invoke RPC Functions

**Suppose there is a remote function:**

```go
func hello(ctx rpcx.Context)(string,error)
```

**Now call it**

```go
out := new(string)
err := callable.Call3(time.Second*5, "hello", out)
std.AssertError(err, "call 'hello'")
fmt.Println("result:",*out)
```


## Callable Calls

**All Call[0-6] support Timeout And Middlewares**

|Function|Header|In|Out|
|:--:|:--:|:--:|:--:|
|Call    |     |   |  |
|Call0 | ✅ | |  |
|Call1 |  |  ✅ |  |
|Call2 |✅  | ✅ |  |
|Call3 |  |  |  ✅ |
|Call4 |  ✅ |  |  ✅ |
|Call5 |  | ✅ |  ✅|
|Call6 |  ✅|  ✅| ✅ |

## Middleware

middlewares works like `AOP` as we know in `java`.

📌**Import middlewares before use**
```go
import "github.com/gen-iot/rpcx/middleware"
```
### RPC Core

📌**middlewares apply on rpc core will affect whole rpc context**

```go
rpc, err := rpcx.New()
std.AssertError(err, "new rpc")
rpc.Use(
    middleware.Recover(true), // recover 
    middleware.ValidateStruct( // validator
        middleware.ValidateInOut,
        std.NewValidator(std.LANG_EN)),
)
```

### RPC Functions

```go
rpc, err := rpcx.New()
std.AssertError(err, "new rpc")
rpc.RegFuncWithName("hello",
    func(ctx rpcx.Context, msg string) (string, error) {
        return "hello from server", nil
    },
    middleware.LoginRequred(),  // require logined
    middleware.MustAdmin(),       // require admin role
)

```

### Callable

```go
err := callable.Call("hello",
    middleware.Recover(true), // recover 
)
```

## More Example

try [examples](https://github.com/gen-iot/rpcx/tree/master/examples)


## License

Released under the [MIT License](https://github.com/gen-iot/rpcx/blob/master/License)