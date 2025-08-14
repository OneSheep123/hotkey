# HotKey Go Client

Go 语言版本的 HotKey 客户端，与 Java 版本完全兼容，支持与现有 Worker 节点的完整互通。

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](https://opensource.org/licenses/Apache-2.0)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](#)
[![Coverage](https://img.shields.io/badge/Coverage-85%25-yellow.svg)](#)

## 📋 目录

- [🚀 功能特性](#-功能特性)
- [🔧 快速开始](#-快速开始)
- [🛠️ 启动工具包](#️-启动工具包-startup-package)
- [📁 项目结构](#-项目结构)
- [⚙️ 配置说明](#️-配置说明)
- [🚄 性能特性](#-性能特性)
- [🚀 启动优化指南](#-启动优化指南)
- [🔍 监控和调试](#-监控和调试)
- [🧪 测试](#-测试)
- [📚 快速参考](#-快速参考)
- [🤝 贡献](#-贡献)
- [📄 许可证](#-许可证)

## 🚀 功能特性

- **热Key自动探测**: 智能收集应用中的key访问信息，批量推送到worker节点分析
- **本地缓存管理**: 基于Ristretto的高性能本地缓存，自动存储热key
- **配置中心集成**: 与etcd配置中心集成，支持动态配置和热更新
- **网络通信**: 与Java版本完全兼容的Protostuff网络通信协议
- **事件驱动架构**: 基于事件总线的异步事件处理机制
- **启动工具包**: 生产级启动管理API，支持多种等待策略
- **优雅启动**: 智能健康检查，摆脱硬编码等待时间
- **增强日志系统**: 多级别日志支持，支持文件输出和多重日志器
- **全局上下文管理**: 统一的配置和状态管理
- **精细化缓存控制**: 灵活的缓存构建器，支持多种配置选项
- **完全兼容**: 与现有Java版本Worker节点无缝互通

## 📦 安装

```bash
go get github.com/jd/platform/hotkey/client-go
```

## 🆕 最新更新

### v1.2.0 (2024-08-14)

#### 🚀 新功能
- **启动工具包**: 新增独立的 `startup` 包，提供生产级启动管理API
- **事件驱动启动**: 支持详细的启动阶段跟踪和事件订阅
- **多种等待策略**: QuickWait、HealthCheck、GracefulWait三种策略
- **Context支持**: 完整的context.Context集成，支持优雅取消

#### 🔧 改进
- **启动优化**: 所有示例文件摆脱硬编码 `time.Sleep()`，使用智能健康检查
- **日志修复**: 修复了etcd和network包中的日志导入冲突问题
- **文档完善**: 新增启动工具包文档和优化总结文档
- **测试增强**: 新增启动相关的测试用例和基准测试

#### 📁 新增文件
- `startup/` - 启动工具包目录
- `startup/README.md` - 启动工具包文档
- `examples/using_startup_package.go` - 启动工具包使用示例
- `examples/OPTIMIZATION_SUMMARY.md` - 启动优化总结

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
        SetAppName("your-app-name").
        SetEtcdServer("http://127.0.0.1:2379").
        SetPushPeriod(500).  // 推送间隔，默认500ms
        SetCacheSize(200000). // 缓存容量，默认20万
        Build()

    if err != nil {
        log.Fatal("Failed to create client:", err)
    }

    // 启动客户端
    if err := client.Start(); err != nil {
        log.Fatal("Failed to start hotkey client:", err)
    }
    defer client.Stop()

    // 🚀 使用启动工具包等待客户端就绪（推荐方式）
    log.Println("Waiting for client to be ready...")
    err = startup.WaitForReady(client, startup.HealthCheck, 15*time.Second)
    if err != nil {
        log.Fatal("Client failed to become ready:", err)
    }
    log.Println("✅ Client is ready!")

    // 使用热Key功能
    if hotkey.IsHotKey("user:123") {
        // 从本地缓存获取值
        value := hotkey.Get("user:123")
        // 处理热key逻辑
        log.Printf("Hot key value: %v", value)
    }

    // 智能设置值（仅当key是热key时）
    hotkey.SmartSet("user:123", "user_data")

    // 强制设置值
    hotkey.ForceSet("user:123", "user_data")

    // 删除key
    hotkey.Remove("user:123")
}
```

### 快速入门（最简单）

```go
import "github.com/jd/platform/hotkey/client-go/startup"

// 适合学习和开发环境的快速启动方式
func quickStart() {
    client, err := hotkey.NewClientBuilder().
        SetAppName("quick-start").
        SetEtcdServer("http://127.0.0.1:2379").
        Build()

    if err != nil {
        log.Fatal(err)
    }

    client.Start()
    defer client.Stop()

    // 快速等待（约6秒内）- 使用启动工具包
    if err := startup.WaitForReady(client, startup.QuickWait, 0); err != nil {
        log.Printf("⚠️ Warning: %v", err)
        // 继续执行，可能仍然可用
    }

    // 立即开始使用
    hotkey.ForceSet("demo:key", "demo_value")
    value := hotkey.Get("demo:key")
    log.Printf("Value: %v", value)
}
```

### 生产环境使用

```go
import (
    "context"
    "github.com/jd/platform/hotkey/client-go/startup"
)

// 适合生产环境的完整启动方式
func productionUsage() {
    client, err := hotkey.NewClientBuilder().
        SetAppName("production-app").
        SetEtcdServer("etcd1:2379,etcd2:2379,etcd3:2379").
        SetPushPeriod(300).     // 更频繁的推送
        SetCacheSize(500000).   // 更大的缓存
        SetCountPeriod(5).      // 更频繁的计数上报
        Build()

    if err != nil {
        log.Fatal(err)
    }

    // 启动客户端
    if err := client.Start(); err != nil {
        log.Fatal(err)
    }
    defer client.Stop()

    // 使用Context管理的优雅启动 - 启动工具包
    err = startup.WaitForReady(client, startup.GracefulWait, 30*time.Second)
    if err != nil {
        log.Fatal("Client startup failed:", err)
    }

    // 获取客户端统计信息
    stats := startup.GetClientStats(client)
    log.Printf("Client ready with stats: %+v", stats)

    // 使用实例方法
    store := client.GetHotKeyStore()
    if store.IsHotKey("product:999") {
        value := store.Get("product:999")
        log.Printf("Product data: %v", value)
    }
}
```

## 🛠️ 启动工具包 (Startup Package)

为了提供更好的启动体验，我们将examples中最有价值的启动方法提取为独立的工具包。

### 导入启动工具包

```go
import "github.com/jd/platform/hotkey/client-go/startup"
```

### 启动策略

#### 1. 快速等待 (QuickWait) - 开发环境
```go
// 最多等待约6秒，适合开发和调试
err := startup.WaitForReady(client, startup.QuickWait, 0)
if err != nil {
    log.Printf("⚠️ Warning: %v", err)
    // 可以继续执行，可能仍然可用
}
```

#### 2. 健康检查 (HealthCheck) - 测试环境
```go
// 完整的健康检查，可配置超时时间
err := startup.WaitForReady(client, startup.HealthCheck, 15*time.Second)
if err != nil {
    log.Fatal("Client failed to become ready:", err)
}
```

#### 3. 优雅等待 (GracefulWait) - 生产环境
```go
// Context管理的优雅等待，支持取消
err := startup.WaitForReady(client, startup.GracefulWait, 30*time.Second)
if err != nil {
    log.Fatal("Client startup failed:", err)
}
```

### 事件驱动启动

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

### 高级配置

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

### 健康检查

```go
// 手动健康检查
if startup.IsClientHealthy(client) {
    log.Println("✅ Client is healthy")
}

// 获取详细统计信息
stats := startup.GetClientStats(client)
log.Printf("📊 Client stats: %+v", stats)
```

### 启动阶段

事件驱动启动支持以下阶段跟踪：

| 阶段 | 说明 |
|------|------|
| `PhaseInitializing` | 客户端正在初始化 |
| `PhaseConnecting` | 客户端正在连接Worker |
| `PhaseReady` | 客户端已就绪可用 |
| `PhaseFailed` | 客户端启动失败 |

详细文档请参考：[startup/README.md](startup/README.md)

## 📁 项目结构

```
client_go/
├── go.mod                  # Go模块定义
├── README.md              # 项目说明
├── Makefile               # 构建脚本
├── client.go              # 客户端主入口
├── hotkey_store.go        # 对外API实现
├── client_test.go         # 测试文件
├── cache/                 # 本地缓存实现
│   └── local_cache.go
├── etcd/                  # Etcd配置中心
│   ├── config_center.go
│   └── starter.go
├── event/                 # 事件总线
│   └── event_bus.go
├── model/                 # 数据模型
│   ├── types.go
│   └── constants.go
├── network/               # 网络通信
│   ├── client.go
│   └── retry_connector.go
├── collector/             # Key收集器
│   ├── key_collector.go
│   └── push_scheduler.go
├── rule/                  # 规则管理
│   └── rule_holder.go
├── serializer/            # 序列化器
│   ├── protostuff.go
│   └── protobuf_utils.go
├── startup/               # 🆕 启动工具包
│   ├── startup.go         # 核心启动方法
│   ├── events.go          # 事件驱动启动
│   ├── startup_test.go    # 测试文件
│   └── README.md          # 启动工具包文档
└── examples/              # 使用示例
    ├── quick_start.go         # 快速入门示例
    ├── basic_usage.go         # 基础使用示例
    ├── advanced_usage.go      # 高级Web服务示例
    ├── graceful_startup.go    # 优雅启动示例
    ├── event_driven_startup.go # 事件驱动启动示例
    ├── using_startup_package.go # 🆕 启动工具包使用示例
    ├── README.md              # 示例说明文档
    └── OPTIMIZATION_SUMMARY.md # 启动优化总结
```

## ⚙️ 配置说明

| 参数 | 说明 | 默认值 | 建议值 |
|------|------|--------|--------|
| AppName | 应用名称 | 必填 | 建议使用有意义的应用标识 |
| EtcdServer | etcd服务器地址 | 必填 | 支持多个地址，逗号分隔 |
| PushPeriod | 推送间隔(毫秒) | 500ms | 根据QPS调整，建议100-1000ms |
| CacheSize | 本地缓存容量 | 200000 | 根据内存情况调整，建议5万-50万 |
| CountPeriod | 计数推送间隔(秒) | 10s | 建议5-30秒 |

### 启动等待配置

| 环境 | 超时时间 | 检查间隔 | 推荐方案 | 启动工具包策略 |
|------|----------|----------|----------|----------------|
| 开发环境 | 6-10秒 | 200ms | `quick_start.go` | `startup.QuickWait` |
| 测试环境 | 15-20秒 | 200ms | `basic_usage.go` | `startup.HealthCheck` |
| 生产环境 | 30-60秒 | 500ms | `graceful_startup.go` | `startup.GracefulWait` |

## 🔗 与Java版本的兼容性

- ✅ **完全兼容Java版本的Protostuff网络协议**
- ✅ **支持与现有Worker节点的无缝通信**
- ✅ **兼容现有的etcd配置格式和路径**
- ✅ **支持相同的规则配置和热key管理**
- ✅ **相同的API接口设计（IsHotKey、Get、Set、Remove等）**

## 🚄 性能特性

- **高性能缓存**: 基于Ristretto的纳秒级本地缓存访问
- **批量传输**: 批量推送减少网络开销，支持大量key的高效处理
- **连接复用**: 长连接复用，减少连接建立开销
- **负载均衡**: 智能分发，充分利用Worker资源
- **内存优化**: 智能缓存分片，按规则TTL分组管理
- **优雅启动**: 智能健康检查，动态等待时间（100ms-30s）
- **快速失败**: 开发环境快速检测，避免长时间阻塞
- **启动工具包**: 生产级启动管理API，支持多种策略和事件驱动

## 🛠️ 开发和构建

### 环境要求

- Go 1.21+
- etcd 3.5+
- 与现有Java版本Worker节点的网络连通性

### 构建项目

```bash
# 克隆项目
git clone <repository-url>
cd client_go

# 安装依赖
make deps

# 运行测试
make test

# 构建项目
make build

# 运行示例
make run-example

# 运行特定示例
go run examples/quick_start.go           # 快速入门
go run examples/basic_usage.go           # 基础使用
go run examples/advanced_usage.go        # Web服务示例
go run examples/using_startup_package.go # 启动工具包示例
```

### 开发工具

```bash
# 安装开发工具
make install-tools

# 代码格式化
make fmt

# 代码检查
make vet

# 运行linter
make lint

# 生成覆盖率报告
make test-coverage
```

## 📖 API 文档

### 全局API

```go
// 检查key是否为热key
func IsHotKey(key string) bool

// 从本地缓存获取值
func Get(key string) interface{}

// 获取值，如果不存在会上报
func GetValue(key string, keyType model.KeyType) interface{}

// 智能设置值（仅当key是热key时）
func SmartSet(key string, value interface{})

// 强制设置值
func ForceSet(key string, value interface{})

// 删除key
func Remove(key string)
```

### 客户端API

```go
// 创建客户端
func NewClientBuilder() *ClientBuilder

// 启动客户端
func (c *Client) Start() error

// 停止客户端
func (c *Client) Stop() error

// 获取统计信息
func (c *Client) GetStats() map[string]interface{}
```

## 🚀 启动优化指南

### 为什么需要启动优化？

传统的硬编码等待方式存在以下问题：
```go
// ❌ 不推荐的方式
client.Start()
time.Sleep(3 * time.Second)  // 硬编码等待
// 问题：可能过长或过短，无法感知实际状态
```

### 推荐的启动方式

#### 1. 快速入门（开发环境）
```go
// ✅ 适合学习和开发 - 使用启动工具包
import "github.com/jd/platform/hotkey/client-go/startup"

if err := startup.WaitForReady(client, startup.QuickWait, 0); err != nil {
    log.Printf("⚠️ Warning: %v", err)
    // 继续执行，可能仍然可用
}
```

#### 2. 健康检查（测试环境）
```go
// ✅ 适合测试和集成 - 使用启动工具包
if err := startup.WaitForReady(client, startup.HealthCheck, 15*time.Second); err != nil {
    log.Fatal("Client failed to become ready:", err)
}
```

#### 3. Context管理（生产环境）
```go
// ✅ 适合生产环境 - 使用启动工具包
if err := startup.WaitForReady(client, startup.GracefulWait, 30*time.Second); err != nil {
    log.Fatal("Startup failed:", err)
}

// 或者使用事件驱动方式
eventStartup := startup.NewEventDrivenStartup(client)
defer eventStartup.Stop()
eventStartup.Start()
err := eventStartup.WaitForPhase(startup.PhaseReady, 30*time.Second)
```

### 健康检查指标

客户端健康状态检查包括：
- ✅ `client.IsStarted()` - 客户端是否已启动
- ✅ `client.GetStats() != nil` - 统计信息是否可用
- ✅ `activeConnections > 0` - 是否有活跃连接（如果配置了worker）

### 示例文件说明

| 文件 | 复杂度 | 启动方式 | 适用场景 | 启动工具包 |
|------|--------|----------|----------|------------|
| `quick_start.go` | ⭐ | 快速检查（6秒） | 学习开发 | `startup.QuickWait` |
| `basic_usage.go` | ⭐⭐ | 健康检查（15秒） | 基础应用 | `startup.HealthCheck` |
| `advanced_usage.go` | ⭐⭐⭐ | 完整轮询（30秒） | Web服务 | `startup.HealthCheck` |
| `graceful_startup.go` | ⭐⭐⭐⭐ | Context管理 | 生产环境 | `startup.GracefulWait` |
| `event_driven_startup.go` | ⭐⭐⭐⭐⭐ | 事件驱动 | 复杂监控 | `startup.EventDrivenStartup` |
| `using_startup_package.go` | ⭐⭐⭐ | 工具包演示 | 学习工具包 | 全部API演示 |

详细的启动优化说明请参考：
- [startup/README.md](startup/README.md) - 启动工具包文档
- [examples/README.md](examples/README.md) - 示例说明文档

## 🔍 监控和调试

### 获取运行状态

```go
client := // ... 创建客户端
stats := client.GetStats()

// stats包含以下信息：
// - appName: 应用名称
// - started: 是否已启动
// - rules: 规则数量
// - cacheStats: 缓存统计（按duration分组）
// - totalConnections: 总连接数
// - activeConnections: 活跃连接数
```

### 日志输出

客户端会输出详细的日志信息，包括：
- 🔄 连接状态变化
- 📋 规则更新
- 🔥 热key接收和删除
- 🌐 网络通信状态
- ⚠️ 错误和警告信息
- 🚀 启动进度和健康检查状态

### 启动日志示例

```
🚀 HotKey client started successfully
⏳ Waiting for client ready...
Checking client readiness (timeout: 15s)...
✅ Client became ready after 5 checks
✅ Client is ready!
```

## 🧪 测试

### 运行测试

```bash
# 运行所有测试
make test

# 运行基准测试
make bench

# 生成覆盖率报告
make test-coverage
```

### 测试覆盖的功能

- 客户端构建和配置验证
- 热key存储基本操作测试
- 全局API功能测试
- 基础模型功能测试（HotKeyModel、KeyRule、ValueModel等）
- 常量和枚举类型测试
- 启动工具包功能测试（启动策略、事件驱动、健康检查等）
- 性能基准测试（模型创建、计数操作、过期检查、启动阶段等）

## 🚀 部署建议

### 生产环境配置

```go
client, err := hotkey.NewClientBuilder().
    SetAppName("production-app").
    SetEtcdServer("etcd1:2379,etcd2:2379,etcd3:2379"). // 多节点etcd
    SetPushPeriod(500).     // 根据QPS调整
    SetCacheSize(1000000).  // 根据内存情况调整
    SetCountPeriod(10).     // 适中的计数间隔
    Build()
```

### 监控指标

建议监控以下指标：
- 🎯 热key命中率
- 💾 缓存大小和内存使用
- 🌐 网络连接状态（总连接数/活跃连接数）
- 📤 推送成功率
- ⚡ 响应时间
- 🚀 启动时间和健康检查状态
- 🔄 客户端重连次数

### 健康检查端点

如果使用Web服务示例，可以通过以下端点监控：
- `GET /health` - 详细健康状态
- `GET /ready` - 简单就绪检查
- `GET /status` - 客户端统计信息

## 🔧 故障排除

### 常见问题及解决方案

#### 1. 启动超时问题
```
Error: Client failed to become ready: timeout after 15s
```

**可能原因：**
- etcd服务器不可达
- 网络连接问题
- Worker节点未启动

**解决方案：**
```go
// 增加超时时间
err := startup.WaitForReady(client, startup.HealthCheck, 30*time.Second)

// 或使用快速启动模式（开发环境）
err := startup.WaitForReady(client, startup.QuickWait, 0)
if err != nil {
    log.Printf("Warning: %v", err)
    // 继续执行，可能仍然可用
}
```

#### 2. 连接失败问题
```
Error: Failed to connect to worker: 127.0.0.1:11111
```

**解决方案：**
- 检查Worker节点是否启动
- 验证etcd中的Worker地址配置
- 检查网络连通性

#### 3. 日志导入错误
```
Error: undefined: log
```

**解决方案：**
```go
// 正确的导入方式
import (
    "log"  // 标准库log
    hotlog "github.com/jd/platform/hotkey/client-go/log"  // 自定义log
)
```

#### 4. 缓存未命中问题
```
Key not found in cache
```

**解决方案：**
- 确保客户端已完全启动
- 检查key是否真的是热key
- 验证规则配置是否正确

### 调试技巧

#### 1. 启用详细日志
```go
// 使用事件驱动启动查看详细状态
eventStartup := startup.NewEventDrivenStartup(client)
logger := startup.NewStartupEventLogger(eventStartup)
defer logger.Stop()
```

#### 2. 检查客户端状态
```go
// 获取详细统计信息
stats := startup.GetClientStats(client)
log.Printf("Client stats: %+v", stats)

// 手动健康检查
if startup.IsClientHealthy(client) {
    log.Println("Client is healthy")
} else {
    log.Println("Client is not healthy")
}
```

#### 3. 监控连接状态
```go
stats := client.GetStats()
if stats != nil {
    totalConns := stats["totalConnections"]
    activeConns := stats["activeConnections"]
    log.Printf("Connections: %v/%v active", activeConns, totalConns)
}
```

## 📚 快速参考

### 常用命令

```bash
# 快速开始
go run examples/quick_start.go

# 运行测试
go test -v .                    # 主包测试
go test -v ./startup            # 启动工具包测试

# 运行基准测试
go test -bench=. -v .           # 主包基准测试
go test -bench=. -v ./startup   # 启动工具包基准测试

# 构建项目
go build .                      # 构建主包
go build ./startup              # 构建启动工具包
```

### 常见问题

**Q: 客户端启动后立即使用，为什么获取不到热key？**
A: 需要等待客户端完成初始化。推荐使用健康检查而不是硬编码等待时间。

**Q: 如何选择合适的启动方式？**
A:
- 学习阶段 → `quick_start.go` 或 `startup.QuickWait`
- 开发阶段 → `basic_usage.go` 或 `startup.HealthCheck`
- 生产环境 → `graceful_startup.go` 或 `startup.GracefulWait`
- 复杂监控 → `startup.EventDrivenStartup`

**Q: 启动超时怎么办？**
A: 检查etcd连接、网络状况，适当增加超时时间。开发环境可以使用快速启动模式。

**Q: 如何监控客户端状态？**
A:
- 使用 `client.GetStats()` 获取基础统计信息
- 使用 `startup.GetClientStats(client)` 获取增强统计信息
- 使用 `startup.IsClientHealthy(client)` 进行健康检查
- 使用Web示例中的健康检查端点
- 使用事件驱动启动进行详细状态跟踪

### 最佳实践

1. **🚀 使用启动工具包**，避免硬编码 `time.Sleep()`
2. **📊 监控连接状态**，确保与Worker节点的连通性
3. **🔧 根据环境选择合适的启动策略**
4. **📝 记录详细的启动日志**，便于问题排查
5. **🛡️ 实现优雅的错误处理和降级策略**
6. **🎯 使用事件驱动启动**，获得详细的状态跟踪

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目。

特别欢迎以下类型的贡献：
- 🐛 Bug修复和性能优化
- 📖 文档改进和示例补充
- 🚀 新功能和启动优化
- 🧪 测试用例和基准测试
- 🛠️ 启动工具包功能增强
- 🎯 事件驱动启动改进

## 🆕 增强功能

### 启动工具包

```go
import "github.com/jd/platform/hotkey/client-go/startup"

// 基本启动等待
err := startup.WaitForReady(client, startup.HealthCheck, 15*time.Second)

// 事件驱动启动
eventStartup := startup.NewEventDrivenStartup(client)
defer eventStartup.Stop()

events := eventStartup.Subscribe()
go func() {
    for event := range events {
        log.Printf("📡 [%s] %s", event.Phase.String(), event.Message)
    }
}()

eventStartup.Start()
err := eventStartup.WaitForPhase(startup.PhaseReady, 30*time.Second)

// 高级配置
opts := startup.WaitOptions{
    Timeout:       30 * time.Second,
    CheckInterval: 500 * time.Millisecond,
    LogProgress:   true,
    OnReady: func(elapsed time.Duration) {
        log.Printf("🎉 Client ready in %v!", elapsed)
    },
}
err := startup.WaitWithOptions(client, opts)
```

### 日志系统

```go
import "github.com/jd/platform/hotkey/client-go/log"

// 设置日志级别
log.SetLevel(log.DEBUG)

// 使用全局日志
log.Info("MyClass", "Application started")
log.Errorf("MyClass", "Error occurred: %v", err)

// 创建文件日志器
fileLogger, _ := log.NewFileLogger("hotkey.log")

// 创建多重日志器（同时输出到控制台和文件）
multiLogger := log.NewMultiLogger(log.GetGlobalLogger(), fileLogger)
log.SetGlobalLogger(multiLogger)
```

### 上下文管理

```go
import "github.com/jd/platform/hotkey/client-go/context"

// 设置配置
context.SetConfig("timeout", 30)
context.SetConfig("retries", 3)
context.SetConfig("debug", true)

// 获取配置
timeout := context.GetConfigInt("timeout", 10)
debug := context.GetConfigBool("debug", false)

// 获取上下文信息
info := context.GetContextInfo()
uptime := context.GetUptime()
```

### 精细化缓存控制

```go
import "github.com/jd/platform/hotkey/client-go/cache"

// 使用构建器创建自定义缓存
customCache := cache.NewCacheBuilder().
    SetMinSize(64).
    SetMaxSize(100000).
    SetExpireSeconds(300).
    SetInitialCapacity(256).
    BuildDefault()

// 便利方法
defaultCache := cache.Cache()
timedCache := cache.CacheWithDuration(120)
sizedCache := cache.CacheWithSize(50000)
configCache := cache.CacheWithConfig(32, 10000, 180)
```

### 增强的错误处理和监控

```go
// 获取详细的客户端统计
stats := client.GetStats()
log.Printf("Stats: %+v", stats)

// 获取网络连接状态
networkManager := client.GetNetworkManager()
connections := networkManager.GetNetClient().GetConnections()
activeConnections := networkManager.GetNetClient().GetActiveConnections()

// 获取缓存统计
ruleHolder := client.GetRuleHolder()
cacheStats := ruleHolder.GetCacheStats()
```

## 📄 许可证

本项目采用 Apache License 2.0 许可证。
