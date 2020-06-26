package main

import (
	"Hystrix/common/discover"
	"Hystrix/common/loadbalance"
	"errors"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/hashicorp/consul/api"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
)

var ErrNoInstances = errors.New("query service instance error")

type HystrixHandler struct {
	//map记录hystrix当前注册的hystrix命令
	hystrixs     map[string]bool
	hystrixMutex *sync.Mutex

	disvoceryClient discover.DiscoveryClient
	loadbalance     loadbalance.LoadBalance
	logger          *log.Logger
}

func NewHystrixHandler(discoverClient discover.DiscoveryClient, loadbalance loadbalance.LoadBalance, logger *log.Logger) *HystrixHandler {
	return &HystrixHandler{
		hystrixs:     make(map[string]bool),
		hystrixMutex: &sync.Mutex{},

		disvoceryClient: discoverClient,
		loadbalance:     loadbalance,
		logger:          logger,
	}
}

func (hy *HystrixHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	reqPath := req.URL.Path
	if reqPath == "" {
		return
	}
	//获取服务名称
	pathArray := strings.Split(reqPath, "/")
	//服务名称
	serviceName := pathArray[1]
	if serviceName == "" {
		//路径不存在
		rw.WriteHeader(404)
		return
	}
	//查询hystrix命令
	if _, ok := hy.hystrixs[serviceName]; !ok {
		hy.hystrixMutex.Lock()
		if _, ok := hy.hystrixs[serviceName]; !ok {
			//把serviceName作为 hystrix命令命名
			hystrix.ConfigureCommand(serviceName, hystrix.CommandConfig{
				//进行hystrix命令自定义
			})
			hy.hystrixs[serviceName] = true
		}
		hy.hystrixMutex.Unlock()
	}
	err := hystrix.Do(serviceName, func() error {

		//根据请求路径中提供的服务名从discoveryClient中获取服务列表
		instances := hy.disvoceryClient.DiscoverServices(serviceName, hy.logger)
		instanceList := make([]*api.AgentService, len(instances))
		for i := 0; i < len(instances); i++ {
			instanceList[i] = instances[i].(*api.AgentService)
		}
		//使用负载均衡算法选取实例
		selectedInstance, err := hy.loadbalance.SelectService(instanceList)
		if err != nil {
			return ErrNoInstances
		}

		//创建Director
		director := func(req *http.Request) {
			//重新组织请求路径，去掉服务名称
			destPath := strings.Join(pathArray[2:], "/")

			hy.logger.Println("service id", selectedInstance.ID)

			//设置代理服务地址信息
			req.URL.Scheme = "http"
			req.URL.Host = fmt.Sprintf("%s:%d", selectedInstance.Address, selectedInstance.Port)
			req.URL.Path = "/" + destPath

		}
		var proxyError error
		//返回代理异常，用于记录hystrix.Do执行失败
		errHandler := func(ew http.ResponseWriter, er *http.Request, err error) {
			proxyError = err
		}

		proxy := &httputil.ReverseProxy{
			Director:     director,
			ErrorHandler: errHandler,
		}

		//进行代理转发
		proxy.ServeHTTP(rw, req)

		//将执行异常反馈给hystrix
		return proxyError
	}, func(err error) error {
		hy.logger.Println("proxy error", err)
		return errors.New("fallback excute")
	})

	//返回hystrix.Do执行的异常
	if err != nil {
		/*
			如果未显式调用WriteHeader，则首次调用Write
			将触发一个隐式的WriteHeader（http.StatusOK）。
			因此，对WriteHeader的显式调用主要用于
			发送错误代码。

			提供的代码必须是有效的HTTP 1xx-5xx状态代码。
			只能写入一个头。 当前不支持发送用户定义的1xx信息标题，
			除了100继续响应标头，读取Request.Body时，服务器会自动发送。
		*/
		//如果hystrix.DO中执行额代理转发逻辑出错，向客户端返回500的错误
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
	}
}
