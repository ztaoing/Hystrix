package main

import (
	"Hystrix/common/discover"
	"Hystrix/common/loadbalance"
	"flag"
	"fmt"
	kitlog "github.com/go-kit/kit/log"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	//创建环境变量
	var (
		consulHost = flag.String("consul.host", "127.0.0.1", "consul server ip address")
		consulPort = flag.Int("conusl.port", 8500, "consul server port")
	)
	flag.Parse()

	/*
		With返回一个新的上下文记录器，其键值在传递给Log调用的键值之前。 如果记录器还是With或创建的上下文记录器
		使用Prefix，会将关键值附加到现有上下文中。

		返回的Logger将每次调用Log方法时，将包含Valuer的所有值元素（奇数索引）替换为其生成的值。
	*/

	//创建日志组件
	var logger kitlog.Logger
	logger = kitlog.NewLogfmtLogger(os.Stderr)
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
	logger = kitlog.With(logger, "caller", kitlog.DefaultCaller)

	consulClient, err := discover.NewKitDiscoverClient(*consulHost, *consulPort)
	if err != nil {
		logger.Log("err", err)
		os.Exit(-1)
	}
	//创建方向代理
	//TODO undefined: NewHystrixHandler
	proxy := NewHystrixHandler(consulClient, new(loadbalance.RandomLoadBalance), log.New(os.Stderr, "", log.LstdFlags))

	errC := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errC <- fmt.Errorf("%s", <-c)
	}()

	//开始监听
	go func() {
		logger.Log("transort", "HTTP", "addr", "9090")
		/*
			ListenAndServe侦听TCP网络地址addr，然后调用带有处理程序的Serve来处理传入连接上的请求。
			接受的连接被配置为启用TCP长连接。处理程序通常为nil，在这种情况下，将使用DefaultServeMux。
			ListenAndServe始终返回非nil错误。
		*/
		errC <- http.ListenAndServe(":9090", proxy)
	}()

	//等待结束
	logger.Log("exit", <-errC)
}
