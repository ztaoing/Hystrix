package main

import (
	"Hystrix/common/discover"
	"Hystrix/common/loadbalance"
	"Hystrix/use-string-service/config"
	"Hystrix/use-string-service/endpoint"
	"Hystrix/use-string-service/service"
	"Hystrix/use-string-service/transport"
	"context"
	"flag"
	"fmt"
	"github.com/go-kit/kit/circuitbreaker"
	uuid "github.com/satori/go.uuid"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

//main完成服务注册并依次构建service层，endpoint层，transport层，
//然后将transport层的http服务部署在配置的端口下

func main() {
	var (
		servicePort = flag.Int("service.port", 10086, "service port")
		serviceHost = flag.String("service.host", "127.0.0.1", "service host")
		serviceName = flag.String("service.name", "use-string", "service name")
		consulPort  = flag.Int("consul.port", 8500, "consul port")
		consulHost  = flag.String("consul.host", "127.0.0.1", "consul host")
	)

	flag.Parse()

	ctx := context.Background()
	errChan := make(chan error)

	//服务发现
	var discoverClient discover.DiscoveryClient
	discoverClient, err := discover.NewKitDiscoverClient(*consulHost, *consulPort)
	if err != nil {
		config.Logger.Println("get consul client failed")
		os.Exit(-1)
	}

	//【service层】
	var svc service.Service
	svc = service.NewUseStringService(discoverClient, &loadbalance.RandomLoadBalance{})

	//【endpoint层】
	useStringEndpoint := endpoint.MakeUseStringEndpoint(svc)
	//(使用装饰者模式)
	useStringEndpoint = circuitbreaker.Hystrix(service.StringServiceCommandName)(useStringEndpoint)
	healthEndpoint := endpoint.MakeHealthCheckEndpoint(svc)
	//封装
	endpts := endpoint.UseStringEndpoint{
		UseStringEndpoint:   useStringEndpoint,
		HealthCheckEndpoint: healthEndpoint,
	}

	//【transport层】
	//创建http.handler
	r := transport.MakeHttpHandler(ctx, endpts, config.KitLogger)
	instanceID := *serviceName + "-" + uuid.NewV4().String()

	//http server
	go func() {
		config.Logger.Println("http server start at port:" + strconv.Itoa(*servicePort))
		//启动前执行注册
		if !discoverClient.Register(*serviceName, instanceID, "/health", *serviceHost, *servicePort, nil, config.Logger) {
			//注册失败
			config.Logger.Println("use-string-service for service %s failed", serviceName)
			os.Exit(-1)
		}
		handler := r
		errChan <- http.ListenAndServe(":"+strconv.Itoa(*servicePort), handler)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	error := <-errChan

	//服务退出注销服务
	discoverClient.Deregister(instanceID, config.Logger)
	config.Logger.Println(error)

}
