package main

import (
	"fmt"
	"github.com/hinha/floody/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

func main() {
	cfg := log.NewDevelopmentConfig()
	//cfg.Base.Encoding = "all"
	logger := log.NewLogger(cfg, log.WithAlerter(func(missed int) {
		fmt.Println("missed", missed)
	}))
	defer logger.Sync()

	for i := 0; i < 10; i++ {
		logger.WithOptions(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).Named("test").Named("nested").Info("hello world")
	}
	time.Sleep(2 * time.Second)

}
