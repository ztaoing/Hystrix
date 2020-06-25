package plugins

import (
	"Hystrix/use-string-service/service"
	"github.com/go-kit/kit/log"
	"time"
)

type loggingMidleware struct {
	service.Service
	loger log.Logger
}

func (mw loggingMidleware) UseStringService(oprationType, a, b string) (result string, err error) {
	defer func(begin time.Time) {
		mw.loger.Log(
			"function", "UseStringService",
			"a", a,
			"b", b,
			"result", result,
			"took", time.Since(begin),
		)
	}(time.Now())
	result, err = mw.Service.UseStringService(oprationType, a, b)
	return
}

func (mw loggingMidleware) HealthCheck() (result bool) {
	defer func(begin time.Time) {
		mw.loger.Log(
			"function", "UseStringService",
			"result", result,
			"took", time.Since(begin),
		)

	}(time.Now())
	result = mw.Service.HealthCheck()
	return result
}

//logging中间件
func LoggingMiddleware(logger log.Logger) service.ServiceMiddleware {
	return func(s service.Service) service.Service {
		return loggingMidleware{s, logger}
	}
}
