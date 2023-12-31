package log

import "github.com/lovewebshell/minicat/minicat/logger"

var Log logger.Logger = &nopLogger{}

func Errorf(format string, args ...interface{}) {
	Log.Errorf(format, args...)
}

func Error(args ...interface{}) {
	Log.Error(args...)
}

func Warnf(format string, args ...interface{}) {
	Log.Warnf(format, args...)
}

func Warn(args ...interface{}) {
	Log.Warn(args...)
}

func Infof(format string, args ...interface{}) {
	Log.Infof(format, args...)
}

func Info(args ...interface{}) {
	Log.Info(args...)
}

func Debugf(format string, args ...interface{}) {
	Log.Debugf(format, args...)
}

func Debug(args ...interface{}) {
	Log.Debug(args...)
}
