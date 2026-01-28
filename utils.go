package writer

import (
	"fmt"
	"runtime"
	"strings"
)

// FormatContent 将任意类型转换为字符串
func FormatContent(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetCaller 获取调用者信息
func GetCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}

	// 只保留文件名和行号
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// ConvertFields 将 FieldAccessor 切片转换为 map
func ConvertFields(fields []FieldAccessor) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}

	result := make(map[string]interface{})
	for _, field := range fields {
		key := field.GetKey()
		// 跳过特殊字段
		if key == "trace" || key == "span" || key == "duration" || key == "log_type" || key == "logType" || key == "user_id" || key == "userId" {
			continue
		}
		result[key] = field.GetValue()
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// ExtractFields 从 FieldAccessor 中提取特殊字段
func ExtractFields(fields []FieldAccessor) (trace, span, duration, logType string) {
	for _, field := range fields {
		key := field.GetKey()
		value := fmt.Sprintf("%v", field.GetValue())
		switch key {
		case "trace":
			trace = value
		case "span":
			span = value
		case "duration":
			duration = value
		case "log_type", "logType":
			logType = value
		}
	}
	return
}

// extractFields 从 LogField 切片中提取特殊字段
func extractFields(fields []LogField) (trace, span, duration, logType string, userID *int64) {
	for _, field := range fields {
		value := fmt.Sprintf("%v", field.Value)
		switch field.Key {
		case "trace":
			trace = value
		case "span":
			span = value
		case "duration":
			duration = value
		case "log_type", "logType":
			logType = value
		case "user_id", "userId":
			if id, ok := toInt64(field.Value); ok {
				userID = &id
			}
		}
	}
	return
}

// toInt64 尝试将值转换为 int64
func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case uint:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		return int64(val), true
	case float32:
		return int64(val), true
	case float64:
		return int64(val), true
	default:
		return 0, false
	}
}

// convertLogFields 将 LogField 切片转换为 map
func convertLogFields(fields []LogField) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}

	result := make(map[string]interface{})
	for _, field := range fields {
		// 跳过特殊字段
		if field.Key == "trace" || field.Key == "span" || field.Key == "duration" || field.Key == "log_type" || field.Key == "logType" || field.Key == "user_id" || field.Key == "userId" {
			continue
		}
		result[field.Key] = field.Value
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
