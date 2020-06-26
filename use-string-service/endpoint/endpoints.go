package endpoint

import (
	"Hystrix/use-string-service/service"
	"context"
	"github.com/go-kit/kit/endpoint"
)

//endpoint层主要负责request/response格式的转换，以及公用拦截器相关的逻辑。
//是go-kit的核心，采用洋葱模式，提供了对日志、限流、熔断、链路追踪和服务监控等方面的扩展能力

//在endpoint层中，需要创建UseStringEndpoint将UseStringService方法提供出去

type UseStringEndpoint struct {
	UseStringEndpoint   endpoint.Endpoint
	HealthCheckEndpoint endpoint.Endpoint
}

//string request
type UseStringRequest struct {
	RequestType string `json:"request_type"`
	A           string `json:"a"`
	B           string `json:"b"`
}

//string response
type UseStringResponse struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

func MakeUseStringEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(UseStringRequest)

		var (
			a, b, opErrorString string
			opError             error
		)
		a = req.A
		b = req.B
		result, opError := svc.UseStringService(req.RequestType, a, b)
		if opError != nil {
			opErrorString = opError.Error()
		}
		return UseStringResponse{
			Result: result,
			Error:  opErrorString,
		}, nil

	}
}

//以endpoint中返回的error来统计调用失败的次数
func MakeUseStringEndpointWithKit(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(UseStringRequest)

		var (
			a, b, opErrorString string
			opError             error
		)
		a = req.A
		b = req.B
		result, opError := svc.UseStringService(req.RequestType, a, b)
		//注意：直接返回业务异常opError
		//不再将业务逻辑的错误封装到response中返回，而是直接通过endpoint的err返回给transport层
		return UseStringResponse{
			Result: result,
			Error:  opErrorString,
		}, opError

	}
}

//健康检查request
type HealthRequest struct {
}

//健康检查 response
type HealthResponse struct {
	Status bool `json:"status"`
}

func MakeHealthCheckEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		stauts := svc.HealthCheck()
		return HealthResponse{
			Status: stauts,
		}, nil
	}
}
