package writer

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// ConsoleWriter 控制台 Writer，将日志输出到标准输出（不依赖 go-zero）
type ConsoleWriter struct{}

// NewConsoleWriter 创建一个控制台 Writer
func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{}
}

// getLevelColor 根据日志级别返回对应的颜色函数
func getLevelColor(level string) func(format string, a ...interface{}) string {
	switch level {
	case "info":
		return color.New(color.FgGreen).SprintfFunc()
	case "error":
		return color.New(color.FgRed, color.Bold).SprintfFunc()
	case "warn":
		return color.New(color.FgYellow).SprintfFunc()
	case "debug":
		return color.New(color.FgCyan).SprintfFunc()
	case "alert", "severe", "stack":
		return color.New(color.FgRed, color.Bold, color.BgBlack).SprintfFunc()
	case "slow":
		return color.New(color.FgMagenta).SprintfFunc()
	case "stat":
		return color.New(color.FgBlue).SprintfFunc()
	default:
		return color.New(color.FgWhite).SprintfFunc()
	}
}

// log 内部日志方法，接收 caller 参数
func (c *ConsoleWriter) log(level string, content any, caller string, fields ...LogField) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	contentStr := FormatContent(content)
	levelColor := getLevelColor(level)

	var parts []string
	// 级别使用颜色
	parts = append(parts, levelColor("[%s]", strings.ToUpper(level)))
	// 时间戳使用灰色
	timestampColor := color.New(color.FgHiBlack)
	parts = append(parts, timestampColor.Sprint(timestamp))
	if caller != "" {
		// caller 使用灰色
		parts = append(parts, timestampColor.Sprint(caller))
	}
	parts = append(parts, contentStr)

	trace, span, duration, logType, userID := extractFields(fields)
	// 字段使用青色
	fieldColor := color.New(color.FgCyan)
	if trace != "" {
		parts = append(parts, fieldColor.Sprint(fmt.Sprintf("trace=%s", trace)))
	}
	if span != "" {
		parts = append(parts, fieldColor.Sprint(fmt.Sprintf("span=%s", span)))
	}
	if duration != "" {
		parts = append(parts, fieldColor.Sprint(fmt.Sprintf("duration=%s", duration)))
	}
	if logType != "" {
		parts = append(parts, fieldColor.Sprint(fmt.Sprintf("log_type=%s", logType)))
	}
	if userID != nil {
		parts = append(parts, fieldColor.Sprint(fmt.Sprintf("user_id=%d", *userID)))
	}

	for _, field := range fields {
		if field.Key != "trace" && field.Key != "span" && field.Key != "duration" && field.Key != "log_type" && field.Key != "logType" && field.Key != "user_id" && field.Key != "userId" {
			parts = append(parts, fieldColor.Sprint(fmt.Sprintf("%s=%v", field.Key, field.Value)))
		}
	}

	output := strings.Join(parts, " ")
	if level == "error" || level == "warn" || level == "alert" || level == "severe" || level == "stack" {
		fmt.Fprintf(os.Stderr, "%s\n", output)
	} else {
		fmt.Fprintf(os.Stdout, "%s\n", output)
	}
}

// Log 写入日志（公开方法，供外部直接调用）
func (c *ConsoleWriter) Log(level string, content any, fields ...LogField) {
	c.log(level, content, GetCaller(2), fields...)
}

// Info 写入 info 级别日志
func (c *ConsoleWriter) Info(content any, fields ...LogField) {
	c.log("info", content, GetCaller(2), fields...)
}

// Error 写入 error 级别日志
func (c *ConsoleWriter) Error(content any, fields ...LogField) {
	c.log("error", content, GetCaller(2), fields...)
}

// Debug 写入 debug 级别日志
func (c *ConsoleWriter) Debug(content any, fields ...LogField) {
	c.log("debug", content, GetCaller(2), fields...)
}

// Warn 写入 warn 级别日志
func (c *ConsoleWriter) Warn(content any, fields ...LogField) {
	c.log("warn", content, GetCaller(2), fields...)
}

// Close 关闭写入器（控制台 Writer 不需要关闭）
func (c *ConsoleWriter) Close() error {
	return nil
}
