# go-kit
1. transport层主要负责网络传输，处理HTTP、grpc、thrift等相关逻辑
2. endpoint层主要负责request/response格式的转换，以及公用拦截器相关的逻辑。是go-kit的核心，采用洋葱模式，提供了对日志、限流、熔断、链路追踪和服务监控等方面的扩展能力
3. service层主要专注于业务逻辑

# 熔断与负载均衡
* use-string-service 服务作为服务调用方，会通过HTTP的方式调用string-service服务提供的接口。
* use-string-service服务使用go-kit的项目结构进行组织

* use-string-service 的service层 中封装了对string-service服务的http调用，同时提供了服务熔断能力
* use-string-service 的endpoint层 中创建了UseStringEndpoint将UseStringService方法提供出去

#hystrix运行流程
* 除了进行服务熔断，hystrix在执行过程中还为不同命名的远程调用提供goroutine隔离的能力。
* goroutine隔离使得不同的远程调用方法在固定数量的goroutine下执行，控制了每种远程调用的并发数量，从而进行流量控制
* 在某个hystrix命令调用出现大量超时阻塞时，仅仅会影响与自己命名相同的hystrix命令，并不会影响到其他hystrix命令以及系统其他请求的执行
* 在hystrix命令配置的goroutine执行数量被占用时，该hystrix命令的执行将会直接进入到失败回滚逻辑中，进行服务降级，保护服务调用者的资源稳定

1. 每一个被hystrix包装的远程调用逻辑都会封装为一个hystrix命令，其内包含用户预置远程调用逻辑和失败回滚逻辑，根据hystrix命名唯一确认一个hystrix命令
2. 根据hystrix命令的命名获取到对应的断路器，判断断路器是否打开。如果断路器已打开，将直接执行失败回滚逻辑，不执行真正的远程调用逻辑，此时服务调用已经被熔断了
3. hystrix中每一种命令都限制了并发数量，当hystrix命令的并发数量超过了执行池中设定的最大执行数量时，额外的请求就会被直接拒绝，进入失败回滚逻辑中，以避免服务过载
4. 在执行远程调用时，执行出现异常或者下游服务超时，那么hystrix命令将会向metrics控制器上传执行结果，并进入到失败回滚逻辑中
5. metrics控制器使用滑动窗口的方式统计一段时间的调用次数、失败次数、超时次数、被拒绝次数。如果在时间窗口内的错误率超过了熔断器错误率阀值，那么断路器将会打开。

#hystrix常用参数:
* type CommandConfig struct{
* Timeout :命令执行的超时时间，远程调用逻辑执行超过该时间将被强制执行超时，进入失败回滚逻辑
* MaxConcurrentRequests :最大并发请求数，代表每个hystrix命令最大执行的并发goroutine数，用于进行流量控制和资源隔离。当同种hystrix执行的并发的数量超过了该值，请求将会直接进入到失败回滚逻辑中，并被标记为拒绝请求上报
* RequestVolumeThreshold :最小请求阀值，只有滑动窗口时间内的请求数量超过该值，断路器才会执行对应的判断逻辑。在低请求量时断路器不会发生效应，即使这些请求全部失败
* SleepWindow :超时窗口时间，是指断路器打开后多久时长进入半开状态，重新允许远程调用的发生，试探下游服务是否恢复正常。如果接下来的请求都成功，断路器将关闭，否则重新打开
* }
* 在hystrix.setting.go文件中有hystrix命令的默认参数设置，如果不需要调整hystrix执行配置，可以直接使用默认设置执行