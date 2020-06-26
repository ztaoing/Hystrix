package service

import (
	"Hystrix/common/discover"
	"Hystrix/common/loadbalance"
	"Hystrix/use-string-service/config"
	"encoding/json"
	"errors"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/hashicorp/consul/api"
	"net/http"
	"net/url"
	"strconv"
)

const (
	StringServiceCommandName = "String.string"
	StringService            = "string" //服务名
)

var ErrHystrixFallbackExecute = errors.New("hystrix fall back execute")

type Service interface {

	//远程调用string-service服务
	UseStringService(oprationType, a, b string) (result string, err error)

	//健康检查
	HealthCheck() bool
}

type UseStringService struct {
	//服务发现客户端
	discoverClient discover.DiscoveryClient
	loadbalance    loadbalance.LoadBalance
}

func NewUseStringService(client discover.DiscoveryClient, lb loadbalance.LoadBalance) Service {

	hystrix.ConfigureCommand(StringServiceCommandName, hystrix.CommandConfig{
		/**
		Timeout:                time.Duration(timeout) * time.Millisecond, 超时
		MaxConcurrentRequests:  max, 最大并发请求数
		RequestVolumeThreshold: uint64(volume), 最低请求阀值
		SleepWindow:            time.Duration(sleep) * time.Millisecond, 时间窗口
		ErrorPercentThreshold:  errorPercent 一旦错误的滚动度量超出请求的百分比，断路器就会打开
		*/
		//设置触发阀值
		RequestVolumeThreshold: 5,
	})

	return &UseStringService{
		discoverClient: client,
		loadbalance:    lb,
	}
}

type StringResponse struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

//将服务发现和http调用通过hystrix.do函数包装为一个命令
//对于每一个hystrix命令我们都需要为他们赋予不同的名称，表明了他们属于不同的远程调用
//相同名称的命令会使用相同的熔断器进行熔断保护
func (s UseStringService) UseStringService(oprationType, a, b string) (result string, err error) {
	//hystrix是一种同步调用方式
	hystrix.Do(StringServiceCommandName, func() error {
		//注意：获取服务名为string的服务列表
		instances := s.discoverClient.DiscoverServices(StringService, config.Logger)
		instancesList := make([]*api.AgentService, len(instances))
		for i := 0; i < len(instances); i++ {
			instancesList[i] = instances[i].(*api.AgentService)
		}

		//使用负载均衡算法获取实例
		selectedInstance, err := s.loadbalance.SelectService(instancesList)
		if err == nil {
			//成功获取
			config.Logger.Printf("current string-service ID is %s and address:port is %s:%s\n",
				selectedInstance.ID, selectedInstance.Address, strconv.Itoa(selectedInstance.Port))
			requestUrl := url.URL{
				Scheme: "http",
				Host:   selectedInstance.Address + ":" + strconv.Itoa(selectedInstance.Port),
				Path:   "/op/" + oprationType + "/" + a + "/" + b,
			}
			resp, err := http.Post(requestUrl.String(), "", nil)
			if err == nil {
				res := &StringResponse{}
				/*
					区别
					1、json.NewDecoder是从一个流里面直接进行解码，代码精干
					2、json.Unmarshal是从已存在与内存中的json进行解码
					3、相对于解码，json.NewEncoder进行大JSON的编码比json.marshal性能高，因为内部使用pool

					场景应用
					1、json.NewDecoder用于http连接与socket连接的读取与写入，或者文件读取
					2、json.Unmarshal用于直接是byte的输入
				*/
				err = json.NewDecoder(resp.Body).Decode(result)
				if err == nil && res.Error == nil {
					result = res.Result
				}
			}
		}
		return err
	}, func(err error) error {
		//服务调用失败时进行异常处理和回滚操作
		return ErrHystrixFallbackExecute
	})

	return result, err

}

//使用kit的hystrix
func (s UseStringService) UseStringServiceWithKit(oprationType, a, b string) (result string, err error) {

	//注意：获取服务名为string的服务列表
	instances := s.discoverClient.DiscoverServices(StringService, config.Logger)
	instancesList := make([]*api.AgentService, len(instances))
	for i := 0; i < len(instances); i++ {
		instancesList[i] = instances[i].(*api.AgentService)
	}

	//使用负载均衡算法获取实例
	selectedInstance, err := s.loadbalance.SelectService(instancesList)
	if err == nil {
		//成功获取
		config.Logger.Printf("current string-service ID is %s and address:port is %s:%s\n",
			selectedInstance.ID, selectedInstance.Address, strconv.Itoa(selectedInstance.Port))
		requestUrl := url.URL{
			Scheme: "http",
			Host:   selectedInstance.Address + ":" + strconv.Itoa(selectedInstance.Port),
			Path:   "/op/" + oprationType + "/" + a + "/" + b,
		}
		resp, err := http.Post(requestUrl.String(), "", nil)
		if err == nil {
			res := &StringResponse{}
			/*
				区别
				1、json.NewDecoder是从一个流里面直接进行解码，代码精干
				2、json.Unmarshal是从已存在与内存中的json进行解码
				3、相对于解码，json.NewEncoder进行大JSON的编码比json.marshal性能高，因为内部使用pool

				场景应用
				1、json.NewDecoder用于http连接与socket连接的读取与写入，或者文件读取
				2、json.Unmarshal用于直接是byte的输入
			*/
			err = json.NewDecoder(resp.Body).Decode(result)
			if err == nil && res.Error == nil {
				result = res.Result
			}
		}
	}
	return result, err

}
func (s UseStringService) HealthCheck() bool {
	return true
}

//装饰者模式
type ServiceMiddleware func(Service) Service
