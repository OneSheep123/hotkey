# HotKey Client Startup Package

这个包提供了生产级的HotKey客户端启动管理工具，从examples中提取最有价值的启动方法并作为公共API提供。

## 🚀 功能特性

- **多种等待策略**: 快速、健康检查、优雅等待三种策略
- **Context支持**: 完整的context.Context集成，支持取消操作
- **事件驱动启动**: 详细的阶段跟踪和事件订阅机制
- **健康检查**: 全面的客户端健康状态验证
- **生产就绪**: 线程安全、经过测试、为生产环境优化

## 📦 安装

```go
import "github.com/jd/platform/hotkey/client-go/startup"
```

## 🔧 快速开始

### 基本使用

```go
package main

import (
    "log"
    "time"
    
    hotkey "github.com/jd/platform/hotkey/client-go"
    "github.com/jd/platform/hotkey/client-go/startup"
)

func main() {
    // 创建客户端
    client, err := hotkey.NewClientBuilder().
        SetAppName("my-app").
        SetEtcdServer("http://127.0.0.1:2379").
        Build()
    
    if err != nil {
        log.Fatal(err)
    }
    
    // 启动客户端
    client.Start()
    defer client.Stop()
    
    // 等待客户端就绪 - 使用健康检查策略
    err = startup.WaitForReady(client, startup.HealthCheck, 15*time.Second)
    if err != nil {
        log.Fatal("Client failed to become ready:", err)
    }
    
    log.Println("✅ Client is ready!")
    // 现在可以安全使用hotkey功能
}
```

## 📋 启动策略

### 1. 快速等待 (QuickWait) - 开发环境
```go
// 最多等待约6秒，适合开发和调试
err := startup.WaitForReady(client, startup.QuickWait, 0)
if err != nil {
    log.Printf("⚠️ Warning: %v", err)
    // 可以继续执行，可能仍然可用
}
```

### 2. 健康检查 (HealthCheck) - 测试环境
```go
// 完整的健康检查，可配置超时时间
err := startup.WaitForReady(client, startup.HealthCheck, 15*time.Second)
if err != nil {
    log.Fatal("Client failed to become ready:", err)
}
```

### 3. 优雅等待 (GracefulWait) - 生产环境
```go
// Context管理的优雅等待，支持取消
err := startup.WaitForReady(client, startup.GracefulWait, 30*time.Second)
if err != nil {
    log.Fatal("Client startup failed:", err)
}
```

## 🎯 事件驱动启动

```go
// 创建事件驱动启动管理器
eventStartup := startup.NewEventDrivenStartup(client)
defer eventStartup.Stop()

// 订阅启动事件
events := eventStartup.Subscribe()
go func() {
    for event := range events {
        log.Printf("📡 [%s] %s", event.Phase.String(), event.Message)
    }
}()

// 启动监控
client.Start()
eventStartup.Start()

// 等待特定阶段
err := eventStartup.WaitForPhase(startup.PhaseReady, 30*time.Second)
```

## ⚙️ 高级配置

```go
// 自定义等待选项
opts := startup.WaitOptions{
    Timeout:       30 * time.Second,
    CheckInterval: 500 * time.Millisecond,
    LogProgress:   true,
    OnProgress: func(attempt int, elapsed time.Duration) {
        if attempt%10 == 0 {
            log.Printf("⏳ Still waiting... attempt %d", attempt)
        }
    },
    OnReady: func(elapsed time.Duration) {
        log.Printf("🎉 Client ready in %v!", elapsed)
    },
}

err := startup.WaitWithOptions(client, opts)
```

## 🔍 健康检查

```go
// 手动健康检查
if startup.IsClientHealthy(client) {
    log.Println("✅ Client is healthy")
}

// 获取详细统计信息
stats := startup.GetClientStats(client)
log.Printf("📊 Client stats: %+v", stats)
```

## 📊 启动阶段

事件驱动启动支持以下阶段跟踪：

| 阶段 | 说明 |
|------|------|
| `PhaseInitializing` | 客户端正在初始化 |
| `PhaseConnecting` | 客户端正在连接Worker |
| `PhaseReady` | 客户端已就绪可用 |
| `PhaseFailed` | 客户端启动失败 |

## 🏥 健康检查标准

客户端被认为健康当：

1. ✅ `client.IsStarted()` 返回true
2. ✅ `client.GetStats()` 返回有效统计信息
3. ✅ 连接状态有效：
   - 如果未配置worker：正常
   - 如果配置了worker：至少有一个活跃连接

## 🧪 测试

```bash
# 运行测试
go test ./startup

# 详细输出
go test -v ./startup

# 基准测试
go test -bench=. ./startup
```

## 📈 性能

- **快速等待**: 最多6秒，200ms间隔
- **健康检查**: 可配置超时，200ms间隔
- **优雅等待**: Context管理，100ms间隔
- **事件驱动**: 200ms监控间隔

## 🎯 最佳实践

1. **选择合适的策略**：
   - 开发环境 → `QuickWait`
   - 测试环境 → `HealthCheck`
   - 生产环境 → `GracefulWait`

2. **设置合理的超时**：
   - 开发环境: 6-10秒
   - 测试环境: 15-20秒
   - 生产环境: 30-60秒

3. **使用Context进行取消**：
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

4. **监控启动事件**：
   ```go
   startup := startup.NewEventDrivenStartup(client)
   logger := startup.NewStartupEventLogger(startup)
   ```

5. **优雅处理错误**：
   ```go
   if err := startup.WaitForReady(client, startup.HealthCheck, 15*time.Second); err != nil {
       log.Printf("Warning: %v", err)
       // 实现降级逻辑
   }
   ```

## 🔗 相关文档

- [主项目文档](../README.md) - HotKey客户端完整文档
- [使用示例](../examples/) - 完整的使用示例
- [示例说明](../examples/README.md) - 示例文档
