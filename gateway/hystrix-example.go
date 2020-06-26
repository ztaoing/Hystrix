package main

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"strconv"
	"time"
)

func main() {
	hystrix.ConfigureCommand("test_command", hystrix.CommandConfig{
		//设置参数
		Timeout: hystrix.DefaultTimeout,
	})
	//同步调用
	err := hystrix.Do("test_command", func() error {
		//执行远程调用或者其他需要在hystrix之后使用的方法
		return nil
	}, func(err error) error {
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	//异步调用
	resultChan := make(chan interface{})
	errChan := hystrix.Go("test_command", func() error {
		//执行远程调用或者其他需要在hystrix之后使用的方法
		resultChan <- "success"
		return nil
	}, func(err error) error {
		//失败回滚方法
		return nil
	})

	select {
	case err := <-errChan:
		//执行失败
		fmt.Println(err)
	case result := <-resultChan:
		//执行成功
		fmt.Println(result)
	case <-time.After(2 * time.Second):
		//超时
		fmt.Println("time out")
		return
	}

	//获取状态
	circuit, _, _ := hystrix.GetCircuit("test_command")
	fmt.Println("command test_command circuit open is" + strconv.FormatBool(circuit.IsOpen()))
}
