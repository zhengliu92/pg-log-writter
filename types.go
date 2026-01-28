package writer

import (
	"context"
	"time"
)

// DBExecutor 数据库执行器接口，用于抽象数据库操作
// 用户可以使用任意 PostgreSQL 驱动（pgx, pq 等）实现此接口
type DBExecutor interface {
	// Exec 执行 SQL 语句
	Exec(ctx context.Context, sql string, args ...any) error
	// Ping 检查数据库连接
	Ping(ctx context.Context) error
	// Close 关闭数据库连接
	Close() error
}

// FieldAccessor 字段访问接口，用于统一处理不同类型的字段
type FieldAccessor interface {
	GetKey() string
	GetValue() interface{}
}

// LogField 日志字段（自定义类型，不依赖 go-zero）
type LogField struct {
	Key   string
	Value any
}

// GetKey 实现 FieldAccessor 接口
func (f LogField) GetKey() string {
	return f.Key
}

// GetValue 实现 FieldAccessor 接口
func (f LogField) GetValue() interface{} {
	return f.Value
}

// Field 创建一个日志字段
func Field(key string, value any) LogField {
	return LogField{Key: key, Value: value}
}

// LogEntry 表示一条日志条目
type LogEntry struct {
	Timestamp string                 `json:"@timestamp"`
	Level     string                 `json:"level"`
	Content   string                 `json:"content"`
	LogType   string                 `json:"log_type,omitempty"` // 日志类型：user（用户）、system（系统）等
	Duration  string                 `json:"duration,omitempty"`
	Trace     string                 `json:"trace,omitempty"`
	Span      string                 `json:"span,omitempty"`
	UserID    *int64                 `json:"user_id,omitempty"` // 用户ID（可选）
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// PostgresConfig Postgresql Writer 配置
type PostgresConfig struct {
	TableName     string        `json:"table_name"`     // 表名
	BufferSize    int           `json:"buffer_size"`    // 缓冲区大小
	FlushInterval time.Duration `json:"flush_interval"` // 刷新间隔
}

// DefaultPostgresConfig 返回默认 Postgresql 配置
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		TableName:     "logs",
		BufferSize:    100,
		FlushInterval: 5 * time.Second,
	}
}
