package gismanager

import "github.com/sirupsen/logrus"

// GetLogger returns the project's default logrus logger.
//
// TODO(PR 4): replace with *slog.Logger per CLAUDE.md's logging rule.
func GetLogger() (logger *logrus.Logger) {
	logger = logrus.New()
	Formatter := new(logrus.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	logger.Formatter = Formatter
	return
}
