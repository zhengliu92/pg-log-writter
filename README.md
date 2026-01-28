# pg-log-writer

一个支持 PostgreSQL 和控制台的 Golang 日志写入工具库。

## 功能特性

- ✅ **轻量级设计**，核心库仅依赖 `fatih/color`（控制台着色）
- ✅ **数据库驱动无关**，通过 `DBExecutor` 接口支持任意 PostgreSQL 驱动（pgx, pq 等）
- ✅ **支持 PostgreSQL**：自动创建表结构和索引
- ✅ 支持批量写入，提高性能
- ✅ 支持缓冲区刷新机制，可配置刷新间隔和缓冲区大小
- ✅ 支持 trace/span/duration/user_id 字段自动提取和存储
- ✅ 线程安全，支持并发写入
- ✅ 异步写入，不阻塞业务代码
- ✅ 优雅关闭，确保所有日志都被写入
- ✅ 提供 `MultiWriter`，支持同时输出到多个目标（控制台 + PostgreSQL）
- ✅ 提供 `ConsoleWriter`，支持控制台输出（支持彩色输出，error/warn 输出到 stderr）

## 安装

```bash
go get github.com/zhengliu92/pg-log-writter
```

## 使用方式

### 1. 实现 DBExecutor 接口

首先需要实现 `DBExecutor` 接口，以下是使用 `pgx` 的示例：

```go
import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

// PgxExecutor 使用 pgx 实现 DBExecutor 接口
type PgxExecutor struct {
    pool *pgxpool.Pool
}

func NewPgxExecutor(dsn string) (*PgxExecutor, error) {
    pool, err := pgxpool.New(context.Background(), dsn)
    if err != nil {
        return nil, err
    }
    return &PgxExecutor{pool: pool}, nil
}

func (e *PgxExecutor) Exec(ctx context.Context, sql string, args ...any) error {
    _, err := e.pool.Exec(ctx, sql, args...)
    return err
}

func (e *PgxExecutor) Ping(ctx context.Context) error {
    return e.pool.Ping(ctx)
}

func (e *PgxExecutor) Close() error {
    e.pool.Close()
    return nil
}
```

### 2. 使用 PostgreSQL Writer

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    writer "github.com/zhengliu92/pg-log-writter"
)

func main() {
    // 创建数据库执行器
    dsn := "postgres://postgres:password@localhost:5432/logs?sslmode=disable"
    db, err := NewPgxExecutor(dsn)
    if err != nil {
        panic(err)
    }

    // 配置 PostgreSQL Writer
    config := &writer.PostgresConfig{
        TableName:     "app_logs",
        BufferSize:    100,
        FlushInterval: 5 * time.Second,
    }

    // 创建 PostgreSQL 写入器
    w, err := writer.NewPostgresqlWriter(db, config)
    if err != nil {
        panic(err)
    }
    defer w.Close()

    // 直接使用
    w.Info("用户登录成功")
    w.Error("数据库连接失败")
    w.Info("请求处理完成",
        writer.Field("duration", "50ms"),
        writer.Field("trace", "abc123"),
        writer.Field("user_id", 12345),
    )
}
```

### 3. 使用 MultiWriter 同时输出到控制台和 PostgreSQL

```go
package main

import (
    "time"
    writer "github.com/zhengliu92/pg-log-writter"
)

func main() {
    // 创建数据库执行器（参考上面的实现）
    db, _ := NewPgxExecutor("postgres://...")

    // 创建 PostgreSQL Writer
    pgConfig := &writer.PostgresConfig{
        TableName:     "app_logs",
        BufferSize:    100,
        FlushInterval: 5 * time.Second,
    }
    pgWriter, _ := writer.NewPostgresqlWriter(db, pgConfig)
    defer pgWriter.Close()

    // 创建控制台 Writer
    consoleWriter := writer.NewConsoleWriter()

    // 创建多路复用 Writer，同时输出到控制台和 PostgreSQL
    w := writer.NewMultiWriter(consoleWriter, pgWriter)
    defer w.Close()

    // 日志会同时输出到控制台和 PostgreSQL
    w.Info("用户登录成功")
    w.Error("数据库连接失败")
    w.Info("请求处理完成",
        writer.Field("duration", "50ms"),
        writer.Field("trace", "abc123"),
        writer.Field("user_id", 12345),
    )
}
```

### 4. 仅使用 Console Writer

```go
package main

import (
    writer "github.com/zhengliu92/pg-log-writter"
)

func main() {
    // 创建控制台 Writer
    w := writer.NewConsoleWriter()

    // 使用控制台输出（支持彩色显示）
    w.Info("这是一条 info 日志")
    w.Error("这是一条 error 日志")  // 输出到 stderr
    w.Warn("这是一条 warn 日志")    // 输出到 stderr
    w.Debug("这是一条 debug 日志")
}
```

## 包结构

```
github.com/zhengliu92/pg-log-writter
├── types.go      # 类型定义和接口（DBExecutor, LogField, LogEntry, PostgresConfig, Writer）
├── postgres.go   # PostgresqlWriter 核心实现
├── console.go    # ConsoleWriter 核心实现
├── multi.go      # MultiWriter 核心实现
└── utils.go      # 工具函数（FormatContent, GetCaller, 字段转换/提取）
```

## 接口定义

### DBExecutor 接口

```go
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
```

### Writer 接口

```go
type Writer interface {
    Info(content any, fields ...LogField)
    Error(content any, fields ...LogField)
    Debug(content any, fields ...LogField)
    Warn(content any, fields ...LogField)
    Log(level string, content any, fields ...LogField)
    Close() error
}
```

## 配置说明

### PostgreSQL Config 结构体

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `TableName` | `string` | 表名 | `"logs"` |
| `BufferSize` | `int` | 缓冲区大小，达到此大小后立即批量写入 | `100` |
| `FlushInterval` | `time.Duration` | 刷新间隔，定期刷新缓冲区（即使未达到 BufferSize） | `5 * time.Second` |

### 配置建议

- **BufferSize**: 根据日志量调整，建议 50-500。值越大，批量写入效率越高，但内存占用也越大。
- **FlushInterval**: 建议 3-10 秒。间隔越短，日志实时性越高，但会增加写入频率。
- **TableName**: 建议使用应用名称，如 `app_logs`，便于区分不同应用的日志。

## API 说明

### 创建 Writer

```go
// 创建 PostgreSQL 写入器（需要传入 DBExecutor 实现）
pgWriter, err := writer.NewPostgresqlWriter(db, config)

// 创建控制台写入器
consoleWriter := writer.NewConsoleWriter()

// 创建多路复用写入器（可组合多个 Writer）
multiWriter := writer.NewMultiWriter(consoleWriter, pgWriter)
```

### 写入日志

```go
// 基本日志方法
w.Info(content, fields...)
w.Error(content, fields...)
w.Debug(content, fields...)
w.Warn(content, fields...)

// 通用日志方法（可指定任意级别）
w.Log("custom", content, fields...)
```

### 创建字段

```go
// 创建字段
writer.Field("key", value)

// 特殊字段（会自动提取到对应列）
writer.Field("trace", "trace-id")       // 提取到 LogEntry.Trace
writer.Field("span", "span-id")         // 提取到 LogEntry.Span
writer.Field("duration", "20ms")        // 提取到 LogEntry.Duration
writer.Field("user_id", 12345)          // 提取到 LogEntry.UserID
writer.Field("log_type", "system")      // 提取到 LogEntry.LogType
```

### 其他方法

```go
// 检查连接（PostgresqlWriter 支持）
err := pgWriter.Ping(ctx)

// 关闭 Writer（会刷新所有缓冲的日志）
err := w.Close()
```

## 存储格式

### 日志条目结构

```json
{
  "@timestamp": "2025-12-17T10:30:00Z",
  "level": "info",
  "content": "[HTTP] 200 - GET /api/users",
  "log_type": "system",
  "duration": "20ms",
  "trace": "5a98a59d88786b63d4605481b542dd83",
  "span": "4df29a5b1c46695d",
  "user_id": 12345,
  "fields": {
    "status": 200,
    "method": "GET"
  }
}
```

### 字段说明

| 字段 | 类型 | 说明 | 来源 |
|------|------|------|------|
| `@timestamp` | `string` | 日志时间戳（RFC3339 格式） | 自动生成 |
| `level` | `string` | 日志级别（info/error/debug/warn） | 方法参数 |
| `content` | `string` | 日志内容 | 方法参数 |
| `log_type` | `string` | 日志类型（user/system 等，可选） | 从字段中提取 |
| `duration` | `string` | 持续时间（如 "20ms"） | 从字段中提取 |
| `trace` | `string` | 追踪 ID | 从字段中提取 |
| `span` | `string` | Span ID | 从字段中提取 |
| `user_id` | `int64` | 用户 ID（可选） | 从字段中提取 |
| `fields` | `object` | 其他自定义字段 | 剩余字段 |

### PostgreSQL 表结构

PostgreSQL Writer 会自动创建表结构和索引（如果不存在）：

```sql
CREATE TABLE app_logs (
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
);

-- 自动创建的索引
CREATE INDEX idx_app_logs_timestamp ON app_logs(timestamp);
CREATE INDEX idx_app_logs_level ON app_logs(level);
CREATE INDEX idx_app_logs_trace ON app_logs(trace);
CREATE INDEX idx_app_logs_user_id ON app_logs(user_id);
CREATE INDEX idx_app_logs_log_type ON app_logs(log_type);
```

## 查询示例

```sql
-- 查看最近的日志
SELECT * FROM app_logs ORDER BY timestamp DESC LIMIT 100;

-- 按级别统计
SELECT level, COUNT(*) FROM app_logs GROUP BY level;

-- 查看特定 trace 的日志
SELECT * FROM app_logs WHERE trace = 'your-trace-id' ORDER BY timestamp;

-- 查看特定用户的日志
SELECT * FROM app_logs WHERE user_id = 12345 ORDER BY timestamp DESC;
```

## 注意事项

### 错误处理

- `NewPostgresqlWriter` 会立即尝试连接后端，如果连接失败会返回错误
- 写入日志时如果后端不可用，错误会被静默处理（不会阻塞业务代码）
- 建议在生产环境中监控后端连接状态，定期调用 `Ping()` 方法

### 性能优化

- 根据日志量调整 `BufferSize` 和 `FlushInterval`
- 批量写入可以提高性能，但会增加内存占用
- 在高并发场景下，建议使用较大的 `BufferSize`（如 200-500）

### 优雅关闭

- 调用 `Close()` 方法会：
  1. 停止后台刷新 goroutine
  2. 等待所有缓冲的日志写入完成
  3. 关闭数据库连接
- 建议在应用退出时调用 `defer w.Close()` 确保所有日志都被写入

### 字段提取规则

- `trace`、`span`、`duration`、`user_id`、`log_type` 字段会被自动提取到对应列
- 其他字段存储在 `fields` JSONB 列中

## License

MIT
