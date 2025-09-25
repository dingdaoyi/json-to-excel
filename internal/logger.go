package internal

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func init() {
	Logger = logrus.New()
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// 从环境变量设置日志级别
	logLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch logLevel {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}
}
