package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

type Logger struct {
	*logrus.Logger
}

func init() {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)
}

func New(devMode bool) *Logger {
	if devMode {
		log.SetLevel(logrus.DebugLevel)
	}
	return &Logger{log}
}

func Get() *Logger {
	return &Logger{log}
}

func SetLevel(level string) {
	switch level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
}

func SetOutput(filepath string) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	}
}
