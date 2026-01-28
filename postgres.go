package writer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// PostgresqlWriter 将日志写入 PostgreSQL 数据库
type PostgresqlWriter struct {
	db            DBExecutor
	tableName     string
	bufferSize    int
	flushInterval time.Duration

	buffer    []LogEntry
	bufferMux sync.Mutex
	done      chan struct{}
	wg        sync.WaitGroup
}

// NewPostgresqlWriter 创建一个 PostgreSQL 日志写入器
// db: 实现 DBExecutor 接口的数据库执行器
// config: 配置项（可选，传 nil 使用默认配置）
func NewPostgresqlWriter(db DBExecutor, config *PostgresConfig) (*PostgresqlWriter, error) {
	if db == nil {
		return nil, fmt.Errorf("db executor is required")
	}

	if config == nil {
		config = DefaultPostgresConfig()
	}

	// 测试连接
	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	w := &PostgresqlWriter{
		db:            db,
		tableName:     config.TableName,
		bufferSize:    config.BufferSize,
		flushInterval: config.FlushInterval,
		buffer:        make([]LogEntry, 0, config.BufferSize),
		done:          make(chan struct{}),
	}

	// 确保表存在
	if err := w.ensureTable(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure table: %w", err)
	}

	// 启动后台刷新协程
	w.wg.Add(1)
	go w.flushLoop()

	return w, nil
}

// ensureTable 确保日志表存在
func (w *PostgresqlWriter) ensureTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGSERIAL PRIMARY KEY,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			level VARCHAR(20) NOT NULL,
			content TEXT,
			log_type VARCHAR(20),
			duration VARCHAR(50),
			trace VARCHAR(100),
			span VARCHAR(100),
			user_id BIGINT,
			fields JSONB
		)
	`, w.tableName)

	if err := w.db.Exec(ctx, query); err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s(timestamp)`, w.tableName, w.tableName),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_level ON %s(level)`, w.tableName, w.tableName),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_trace ON %s(trace)`, w.tableName, w.tableName),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_user_id ON %s(user_id)`, w.tableName, w.tableName),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_log_type ON %s(log_type)`, w.tableName, w.tableName),
	}

	for _, idx := range indexes {
		if err := w.db.Exec(ctx, idx); err != nil {
			return err
		}
	}

	return nil
}

// AddEntry 添加一条日志到缓冲区
func (w *PostgresqlWriter) AddEntry(entry LogEntry) {
	w.bufferMux.Lock()
	defer w.bufferMux.Unlock()

	w.buffer = append(w.buffer, entry)

	if len(w.buffer) >= w.bufferSize {
		w.flushLocked()
	}
}

// Log 写入日志（核心方法）
func (w *PostgresqlWriter) Log(level string, content any, fields ...LogField) {
	trace, span, duration, logType, userID := extractFields(fields)
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Content:   FormatContent(content),
		LogType:   logType,
		Duration:  duration,
		Trace:     trace,
		Span:      span,
		UserID:    userID,
		Fields:    convertLogFields(fields),
	}
	w.AddEntry(entry)
}

// Info 写入 info 级别日志
func (w *PostgresqlWriter) Info(content any, fields ...LogField) {
	w.Log("info", content, fields...)
}

// Error 写入 error 级别日志
func (w *PostgresqlWriter) Error(content any, fields ...LogField) {
	w.Log("error", content, fields...)
}

// Debug 写入 debug 级别日志
func (w *PostgresqlWriter) Debug(content any, fields ...LogField) {
	w.Log("debug", content, fields...)
}

// Warn 写入 warn 级别日志
func (w *PostgresqlWriter) Warn(content any, fields ...LogField) {
	w.Log("warn", content, fields...)
}

// Infof 写入 info 级别格式化日志
func (w *PostgresqlWriter) Infof(format string, args ...any) FormatLogger {
	return &postgresFormatLogger{writer: w, level: "info", content: fmt.Sprintf(format, args...)}
}

// Errorf 写入 error 级别格式化日志
func (w *PostgresqlWriter) Errorf(format string, args ...any) FormatLogger {
	return &postgresFormatLogger{writer: w, level: "error", content: fmt.Sprintf(format, args...)}
}

// Debugf 写入 debug 级别格式化日志
func (w *PostgresqlWriter) Debugf(format string, args ...any) FormatLogger {
	return &postgresFormatLogger{writer: w, level: "debug", content: fmt.Sprintf(format, args...)}
}

// Warnf 写入 warn 级别格式化日志
func (w *PostgresqlWriter) Warnf(format string, args ...any) FormatLogger {
	return &postgresFormatLogger{writer: w, level: "warn", content: fmt.Sprintf(format, args...)}
}

// Logf 写入格式化日志
func (w *PostgresqlWriter) Logf(level string, format string, args ...any) FormatLogger {
	return &postgresFormatLogger{writer: w, level: level, content: fmt.Sprintf(format, args...)}
}

// postgresFormatLogger 用于 PostgresqlWriter 的格式化日志链式调用
type postgresFormatLogger struct {
	writer  *PostgresqlWriter
	level   string
	content string
}

// Fields 添加字段并写入日志
func (f *postgresFormatLogger) Fields(fields ...LogField) {
	f.writer.Log(f.level, f.content, fields...)
}

// flushLoop 后台定时刷新协程
func (w *PostgresqlWriter) flushLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.Flush()
		case <-w.done:
			w.Flush()
			return
		}
	}
}

// Flush 刷新缓冲区到数据库
func (w *PostgresqlWriter) Flush() {
	w.bufferMux.Lock()
	defer w.bufferMux.Unlock()
	w.flushLocked()
}

// flushLocked 在已持有锁的情况下刷新缓冲区
func (w *PostgresqlWriter) flushLocked() {
	if len(w.buffer) == 0 {
		return
	}

	entries := make([]LogEntry, len(w.buffer))
	copy(entries, w.buffer)
	w.buffer = w.buffer[:0]

	// 异步写入数据库
	go w.writeEntries(entries)
}

// writeEntries 批量写入日志条目
func (w *PostgresqlWriter) writeEntries(entries []LogEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, entry := range entries {
		fieldsJSON, _ := json.Marshal(entry.Fields)
		query := fmt.Sprintf(`
			INSERT INTO %s (timestamp, level, content, log_type, duration, trace, span, user_id, fields)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, w.tableName)

		ts, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			ts = time.Now()
		}

		_ = w.db.Exec(ctx, query,
			ts,
			entry.Level,
			entry.Content,
			entry.LogType,
			entry.Duration,
			entry.Trace,
			entry.Span,
			entry.UserID,
			fieldsJSON,
		)
	}
}

// Close 关闭写入器
func (w *PostgresqlWriter) Close() error {
	close(w.done)
	w.wg.Wait()
	return w.db.Close()
}

// Ping 检查数据库连接
func (w *PostgresqlWriter) Ping(ctx context.Context) error {
	return w.db.Ping(ctx)
}
