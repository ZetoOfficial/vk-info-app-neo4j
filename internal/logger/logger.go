package logger

import (
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

func Setup(level string, file string) {
	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logrus.Fatalf("Неизвестный уровень логирования: %s", level)
	}
	logrus.SetLevel(logLevel)

	logrus.SetFormatter(&logrus.JSONFormatter{})

	if file != "" {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logrus.Fatalf("Не удалось открыть файл логов: %v", err)
		}
		logrus.SetOutput(f)
	} else {
		logrus.SetOutput(os.Stdout)
	}
}
