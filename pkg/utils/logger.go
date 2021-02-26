package utils

import "github.com/sirupsen/logrus"

// InitLogger initializes logger.
func InitLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	return logger
}
