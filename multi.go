package writer

import (
	"fmt"
)

// Writer 日志写入器接口（不依赖 go-zero）
type Writer interface {
	Info(content any, fields ...LogField)
	Error(content any, fields ...LogField)
	Debug(content any, fields ...LogField)
	Warn(content any, fields ...LogField)
	Log(level string, content any, fields ...LogField)
	// 格式化输出方法
	Infof(format string, args ...any) FormatLogger
	Errorf(format string, args ...any) FormatLogger
	Debugf(format string, args ...any) FormatLogger
	Warnf(format string, args ...any) FormatLogger
	Logf(level string, format string, args ...any) FormatLogger
	Close() error
}

// MultiWriter 多路复用 Writer，可以同时写入多个 Writer（不依赖 go-zero）
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriter 创建一个多路复用 Writer
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Log 写入日志（核心方法）
func (m *MultiWriter) Log(level string, content any, fields ...LogField) {
	for _, w := range m.writers {
		w.Log(level, content, fields...)
	}
}

// Info 写入 info 级别日志
func (m *MultiWriter) Info(content any, fields ...LogField) {
	for _, w := range m.writers {
		w.Info(content, fields...)
	}
}

// Error 写入 error 级别日志
func (m *MultiWriter) Error(content any, fields ...LogField) {
	for _, w := range m.writers {
		w.Error(content, fields...)
	}
}

// Debug 写入 debug 级别日志
func (m *MultiWriter) Debug(content any, fields ...LogField) {
	for _, w := range m.writers {
		w.Debug(content, fields...)
	}
}

// Warn 写入 warn 级别日志
func (m *MultiWriter) Warn(content any, fields ...LogField) {
	for _, w := range m.writers {
		w.Warn(content, fields...)
	}
}

// Infof 写入 info 级别格式化日志
func (m *MultiWriter) Infof(format string, args ...any) FormatLogger {
	content := fmt.Sprintf(format, args...)
	return &multiFormatLogger{writers: m.writers, level: "info", content: content}
}

// Errorf 写入 error 级别格式化日志
func (m *MultiWriter) Errorf(format string, args ...any) FormatLogger {
	content := fmt.Sprintf(format, args...)
	return &multiFormatLogger{writers: m.writers, level: "error", content: content}
}

// Debugf 写入 debug 级别格式化日志
func (m *MultiWriter) Debugf(format string, args ...any) FormatLogger {
	content := fmt.Sprintf(format, args...)
	return &multiFormatLogger{writers: m.writers, level: "debug", content: content}
}

// Warnf 写入 warn 级别格式化日志
func (m *MultiWriter) Warnf(format string, args ...any) FormatLogger {
	content := fmt.Sprintf(format, args...)
	return &multiFormatLogger{writers: m.writers, level: "warn", content: content}
}

// Logf 写入格式化日志
func (m *MultiWriter) Logf(level string, format string, args ...any) FormatLogger {
	content := fmt.Sprintf(format, args...)
	return &multiFormatLogger{writers: m.writers, level: level, content: content}
}

// multiFormatLogger 用于 MultiWriter 的格式化日志链式调用
type multiFormatLogger struct {
	writers []Writer
	level   string
	content string
}

// Fields 添加字段并写入日志
func (f *multiFormatLogger) Fields(fields ...LogField) {
	for _, w := range f.writers {
		w.Log(f.level, f.content, fields...)
	}
}

// Close 关闭所有 Writer
func (m *MultiWriter) Close() error {
	var errs []error
	for _, w := range m.writers {
		if err := w.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing writers: %v", errs)
	}
	return nil
}
