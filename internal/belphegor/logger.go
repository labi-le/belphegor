package belphegor

import "github.com/sirupsen/logrus"

var logger = logrus.New()

func Logger() *logrus.Logger {
	return logger
}
