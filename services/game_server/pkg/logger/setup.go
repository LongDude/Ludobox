package logger

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func LoggerSetup(level string) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	logger.SetLevel(parseLevel(level))

	return logger
}

func ShouldLogRequests(level string) bool {
	return parseLevel(level) >= logrus.DebugLevel
}

func parseLevel(level string) logrus.Level {
	parsedLevel, err := logrus.ParseLevel(strings.TrimSpace(strings.ToLower(level)))
	if err != nil {
		return logrus.InfoLevel
	}

	return parsedLevel
}
