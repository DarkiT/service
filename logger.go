package service

import (
	"log/slog"
)

// Logger 日志接口
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}

// logAdapter 默认日志记录器
type logAdapter struct {
	logger *slog.Logger
}

// 全局日志实例
var defaultLogger Logger = &logAdapter{logger: slog.Default()}

// SetLogger 设置日志实现
func (bs *BaseService) SetLogger(l Logger) *BaseService {
	if l != nil {
		defaultLogger = l
	}
	return bs
}

// GetLogger 获取日志器
func (bs *BaseService) GetLogger() Logger {
	return defaultLogger
}

func (l *logAdapter) Debug(format string, args ...any) {
	l.logger.Debug(format, args...)
}

func (l *logAdapter) Info(format string, args ...any) {
	l.logger.Info(format, args...)
}

func (l *logAdapter) Warn(format string, args ...any) {
	l.logger.Warn(format, args...)
}

func (l *logAdapter) Error(format string, args ...any) {
	l.logger.Error(format, args...)
}
