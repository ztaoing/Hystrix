package transport

import (
	"Hystrix/use-string-service/endpoint"
	"context"
	"encoding/json"
	"errors"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

//在transport层中，将UseStringEndpoint部署在use-string-service服务的/op/{type}/{b}路径下
//这样在调用use-string-service服务的/op/{type}/{b}接口时，会把请求转发给string-service服务进行处理
//以验证负载均衡和服务熔断的效果。
var (
	ErrorBadRequest = errors.New("invalid request paramter")
)

//使用mux创建路由
func MakeHttpHandler(ctx context.Context, endpoint endpoint.UseStringEndpoint, logger log.Logger) http.Handler {
	r := mux.NewRouter()

	options := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kithttp.ServerErrorEncoder(encodeError),
	}

	r.Methods("POST").Path("/op/{type}/{a}/{b}").Handler(kithttp.NewServer(
		endpoint.UseStringEndpoint,
		decodeStringRequest,
		endcodeStringResponse,
		options...,
	))

	r.Path("/metrics").Handler(promhttp.Handler())

	r.Methods("GET").Path("/health").Handler(kithttp.NewServer(
		endpoint.HealthCheckEndpoint,
		decodeHealthCheckRequest,
		endcodeStringResponse,
		options...,
	))

	//添加hystrix
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	r.Handle("/hystrix/stream", hystrixStreamHandler)

	return r
}

//解析stirng请求
func decodeStringRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	requestType, ok := vars["type"]
	if !ok {
		return nil, ErrorBadRequest
	}

	pa, ok := vars["a"]
	if !ok {
		return nil, ErrorBadRequest
	}

	pb, ok := vars["b"]
	if !ok {
		return nil, ErrorBadRequest
	}

	return endpoint.UseStringRequest{
		RequestType: requestType,
		A:           pa,
		B:           pb,
	}, nil

}

//编码string 应答
func endcodeStringResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json;chartset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

//解码healthCheck请求
func decodeHealthCheckRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return endpoint.HealthRequest{}, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-type", "application/json;charset=utf-8")
	switch err {
	default:
		w.WriteHeader(http.StatusInternalServerError)

	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
