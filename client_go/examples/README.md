# HotKey Go Client 启动优化示例

本目录包含了几种优雅的客户端启动方式，替代简单的 `time.Sleep()` 等待。

## 🚀 优化方案对比

### 1. 原始方案（不推荐）
```go
// ❌ 不够优雅的方式
client.Start()
time.Sleep(2 * time.Second)  // 硬编码等待时间 - basic_usage.go 原版
time.Sleep(3 * time.Second)  // 硬编码等待时间 - advanced_usage.go 原版
startServer()
```

**问题：**
- 硬编码等待时间，可能过长或过短
- 无法感知实际的初始化状态
- 在不同环境下表现不一致
- 浪费时间或初始化不完整

### 2. 健康检查轮询（推荐）
```go
// ✅ 使用健康检查轮询
client.Start()
if err := waitForClientReady(client, 30*time.Second); err != nil {
    log.Fatal("Client failed to become ready:", err)
}
startServer()
```

**优点：**
- 基于实际状态检查
- 可配置超时时间
- 实现简单，易于理解

**文件：** `advanced_usage.go`, `basic_usage.go`

### 2.5. 快速入门版本（开发推荐）
```go
// ✅ 快速简单的健康检查
client.Start()
if err := quickWaitReady(client); err != nil {
    log.Printf("Warning: %v", err)
    // 继续执行，可能仍然可用
}
startDemo()
```

**优点：**
- 实现极简，适合学习和开发
- 快速失败，不会长时间阻塞
- 有警告但不会完全失败

**文件：** `quick_start.go`

### 3. Context + Channel 优雅等待（推荐）
```go
// ✅ 使用 Context 和 Channel
notifier := NewClientReadyNotifier(client)
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

notifier.Start(ctx)
if err := notifier.WaitReady(ctx); err != nil {
    log.Fatal("Client failed to become ready:", err)
}
startServer()
```

**优点：**
- 支持 Context 取消
- 线程安全
- 可复用的通知器
- 更好的资源管理

**文件：** `graceful_startup.go`

### 4. 事件驱动启动（高级）
```go
// ✅ 事件驱动的启动管理
startup := NewEventDrivenStartup(client)
startup.Start()

if err := startup.WaitForPhase(PhaseReady, 30*time.Second); err != nil {
    log.Fatal("Startup failed:", err)
}
startServer()
```

**优点：**
- 详细的启动阶段跟踪
- 事件订阅机制
- 丰富的状态信息
- 适合复杂的启动流程

**文件：** `event_driven_startup.go`

## 📊 启动阶段说明

### 健康检查指标
- **IsStarted()** - 客户端是否已启动
- **GetStats()** - 获取统计信息
- **activeConnections** - 活跃连接数
- **totalConnections** - 总连接数

### 事件驱动阶段
1. **Initializing** - 初始化阶段
2. **Connecting** - 连接Worker阶段  
3. **Ready** - 就绪阶段
4. **Failed** - 失败阶段

## 📁 文件说明

| 文件 | 复杂度 | 适用场景 | 特点 |
|------|--------|----------|------|
| `quick_start.go` | ⭐ | 学习/开发 | 最简单，快速失败 |
| `basic_usage.go` | ⭐⭐ | 基础应用 | 详细演示，健康检查 |
| `advanced_usage.go` | ⭐⭐⭐ | Web服务 | HTTP接口，完整功能 |
| `graceful_startup.go` | ⭐⭐⭐⭐ | 生产环境 | Context管理，优雅处理 |
| `event_driven_startup.go` | ⭐⭐⭐⭐⭐ | 复杂系统 | 事件驱动，详细监控 |

## 🛠️ 使用建议

### 学习和开发
使用 **快速入门** 方案：
```go
func quickWaitReady(client *hotkey.Client) error {
    maxChecks := 30 // 约6秒
    for i := 0; i < maxChecks; i++ {
        if client.IsStarted() && client.GetStats() != nil {
            return nil
        }
        time.Sleep(200 * time.Millisecond)
    }
    return fmt.Errorf("not ready after checks")
}
```

### 简单应用
使用 **健康检查轮询** 方案：
```go
func waitForClientReady(client *hotkey.Client, timeout time.Duration) error {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    timeoutChan := time.After(timeout)
    
    for {
        select {
        case <-ticker.C:
            if isClientReady(client) {
                return nil
            }
        case <-timeoutChan:
            return fmt.Errorf("timeout")
        }
    }
}
```

### 中等复杂度应用
使用 **Context + Channel** 方案：
```go
notifier := NewClientReadyNotifier(client)
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

notifier.Start(ctx)
err := notifier.WaitReady(ctx)
```

### 复杂应用
使用 **事件驱动** 方案：
```go
startup := NewEventDrivenStartup(client)
events := startup.Subscribe()

go func() {
    for event := range events {
        log.Printf("Phase: %s - %s", event.Phase, event.Message)
    }
}()

startup.Start()
err := startup.WaitForPhase(PhaseReady, 30*time.Second)
```

## 🔧 配置建议

### 超时时间
- **开发环境**: 10-15秒
- **测试环境**: 20-30秒  
- **生产环境**: 30-60秒

### 检查间隔
- **轮询间隔**: 100-200毫秒
- **心跳间隔**: 1-5秒

### 连接要求
- **最小活跃连接**: 1个
- **推荐活跃连接**: 总连接数的50%以上

## 🚦 健康检查端点

所有示例都提供了标准的健康检查端点：

- `GET /health` - 详细健康状态
- `GET /ready` - 简单就绪检查
- `GET /status` - 客户端统计信息

## 🧪 测试方法

### 1. 启动etcd（如果需要）
```bash
docker run -d --name etcd \
  -p 12379:2379 \
  quay.io/coreos/etcd:latest \
  etcd --listen-client-urls http://0.0.0.0:2379 \
  --advertise-client-urls http://0.0.0.0:2379
```

### 2. 运行示例
```bash
# 健康检查轮询
go run advanced_usage.go

# Context + Channel
go run graceful_startup.go

# 事件驱动
go run event_driven_startup.go
```

### 3. 测试健康检查
```bash
# 检查健康状态
curl http://localhost:8080/health

# 检查就绪状态  
curl http://localhost:8080/ready

# 测试热key检测
curl "http://localhost:8080/hotkey?key=test:123"
```

## 📈 性能对比

| 方案 | 启动时间 | 内存占用 | CPU占用 | 复杂度 |
|------|----------|----------|---------|--------|
| Sleep | 固定3秒 | 最低 | 最低 | 最简单 |
| 健康检查 | 100ms-30s | 低 | 低 | 简单 |
| Context+Channel | 100ms-30s | 中 | 低 | 中等 |
| 事件驱动 | 100ms-30s | 中 | 中 | 复杂 |

## 🎯 最佳实践

1. **生产环境推荐**: Context + Channel 方案
2. **开发环境推荐**: 健康检查轮询方案  
3. **监控系统推荐**: 事件驱动方案
4. **始终设置合理的超时时间**
5. **提供健康检查端点**
6. **记录详细的启动日志**
7. **优雅处理启动失败**
