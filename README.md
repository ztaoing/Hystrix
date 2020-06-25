# go-kit
1. transport层主要负责网络传输，处理HTTP、grpc、thrift等相关逻辑
2. endpoint层主要负责request/response格式的转换，以及公用拦截器相关的逻辑。是go-kit的核心，采用洋葱模式，提供了对日志、限流、熔断、链路追踪和服务监控等方面的扩展能力
3. service层主要专注于业务逻辑

# 熔断与负载均衡
* use-string-service 服务作为服务调用方，会通过HTTP的方式调用string-service服务提供的接口。
* use-string-service服务使用go-kit的项目结构进行组织

* use-string-service 的service层 中封装了对string-service服务的http调用，同时提供了服务熔断能力
* use-string-service 的endpoint层 中创建了UseStringEndpoint将UseStringService方法提供出去