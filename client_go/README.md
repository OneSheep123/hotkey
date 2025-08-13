# HotKey Client Go 版本

这是HotKey Client的Go语言实现版本，提供了与Java版本相同的功能特性。

## 功能特性

### 1. 热Key自动探测
- 智能收集应用中的key访问信息
- 批量推送key信息到worker节点进行分析
- 规则过滤，只上报符合配置规则的key
- 可配置推送间隔，平衡探测及时性和资源消耗

### 2. 本地缓存管理
- 基于go-cache的高性能本地缓存
- 自动将探测到的热key存储在本地内存中
- 支持TTL过期时间，自动清理过期数据
- 可配置缓存最大容量，防止内存溢出

### 3. 事件驱动架构
- 基于反射的事件总线机制
- 异步事件处理，提升系统响应性
- 支持多种事件类型的订阅和处理
- 实时响应配置和状态变化

### 4. 网络通信
- 基于net包的TCP连接管理
- 自动管理worker节点的连接状态
- 通过hash算法智能分发key到不同worker
- 断线重连机制

## 项目结构

```
client_go/
├── config/           # 配置管理
├── cache/            # 本地缓存
├── eventbus/         # 事件总线
├── worker/           # Worker连接管理
├── key/              # Key收集与推送
├── rule/             # 规则管理
├── callback/         # 回调处理
├── netty/            # 网络通信（待实现）
├── etcd/             # Etcd集成（待实现）
├── log/              # 日志管理（待实现）
├── go.mod            # Go模块依赖
├── client_starter.go # 主启动器
└── README.md         # 项目说明
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 基本使用

```go
package main

import (
    "hotkey-client/config"
    "time"
)

func main() {
    // 创建客户端启动器
    starter := NewBuilder().
        SetAppName("your-app-name").
        SetEtcdServers([]string{"127.0.0.1:2379"}).
        SetPushPeriod(500 * time.Millisecond).
        SetCacheSize(200000).
        Build()

    // 启动管道
    err := starter.StartPipeline()
    if err != nil {
        panic(err)
    }

    // 使用热Key功能
    store := callback.GetInstance()
    
    // 判断key是否为热key
    if store.IsHotKey("user:123") {
        // 从本地缓存获取值
        value := store.Get("user:123")
        // 处理热key逻辑
    }

    // 智能设置值（仅当key是热key时）
    store.SmartSet("user:123", userInfo)

    // 强制设置值
    store.ForceSet("user:123", userInfo)

    // 删除key
    store.Remove("user:123")
}
```

## 核心模块说明

### 配置管理 (config)
- `ClientConfig`: 客户端配置结构
- `Context`: 全局上下文管理

### 本地缓存 (cache)
- `LocalCache`: 缓存接口
- `CaffeineCache`: 基于go-cache的实现
- `CacheFactory`: 缓存工厂

### 事件总线 (eventbus)
- `EventBus`: 事件总线核心
- 支持多种事件类型：Worker变化、新Key、规则变化等

### Worker管理 (worker)
- `WorkerInfoHolder`: Worker信息持有者
- `WorkerRetryConnector`: Worker重连器
- `WorkerChangeSubscriber`: Worker变化订阅者

### Key管理 (key)
- `IKeyCollector`: Key收集器接口
- `TurnKeyCollector`: 轮流Key收集器
- `PushSchedulerStarter`: 推送调度器

### 规则管理 (rule)
- `KeyRuleHolder`: Key规则持有者
- 支持全匹配、前缀匹配、通配符匹配

### 热Key存储 (callback)
- `JdHotKeyStore`: 热Key存储核心
- `ValueModel`: 值模型

## 配置参数

| 参数 | 说明 | 默认值 | 建议值 |
|------|------|--------|--------|
| appName | 应用名称 | 必填 | 建议使用有意义的应用标识 |
| etcdServers | etcd服务器地址 | 必填 | 支持多个地址 |
| pushPeriod | 推送间隔 | 500ms | 根据QPS调整，建议100-1000ms |
| cacheSize | 本地缓存容量 | 200000 | 根据内存情况调整，建议5万-50万 |

## 事件类型

### 1. WorkerInfoChangeEvent
- 触发时机：Worker节点信息变化
- 处理逻辑：连接新的Worker节点，管理连接状态

### 2. ReceiveNewKeyEvent
- 触发时机：接收到新的热Key
- 处理逻辑：更新本地缓存，处理热Key的增删

### 3. KeyRuleInfoChangeEvent
- 触发时机：Key规则配置变化
- 处理逻辑：重建本地规则缓存，更新探测策略

## 性能特性

### 缓存性能
- 读取性能：纳秒级本地缓存访问
- 写入性能：异步批量推送，最小化性能影响
- 内存效率：智能内存管理，支持大容量缓存

### 网络性能
- 批量传输：批量推送减少网络开销
- 连接复用：长连接复用，减少连接建立开销
- 负载均衡：智能分发，充分利用Worker资源

## 高可用特性

### 故障容错
- Worker故障转移：自动检测Worker故障并切换
- 连接重试：自动重连机制，保证服务连续性

### 配置热更新
- 规则热更新：支持运行时更新探测规则
- Worker动态发现：自动发现新增Worker节点

## 待实现模块

### 1. Etcd集成 (etcd)
- etcd客户端连接
- 配置监听和同步
- 规则动态更新

### 2. 网络通信 (netty)
- TCP连接管理
- 消息编解码
- 心跳机制

### 3. 日志管理 (log)
- 结构化日志
- 日志级别控制
- 性能监控

## 最佳实践

### 1. 应用集成
- 在应用启动时初始化客户端
- 配置合适的推送间隔和缓存容量
- 监控系统资源使用情况

### 2. 性能优化
- 根据应用QPS调整推送间隔
- 合理设置缓存容量，避免内存溢出
- 监控缓存命中率，优化key规则配置

### 3. 高可用部署
- 配置多个Etcd节点
- 部署多个Worker节点
- 监控系统健康状态

## 常见问题

### Q: 如何判断key是否为热key？
A: 使用`callback.GetInstance().IsHotKey(key)`方法，该方法会自动上报key信息并返回是否为热key。

### Q: 推送间隔设置多少合适？
A: 建议根据应用QPS设置，单机QPS 10个时建议500ms，QPS 100个时建议100ms。

### Q: 本地缓存容量如何设置？
A: 根据可用内存和预期热key数量设置，建议5万-50万之间，默认20万。

### Q: 事件驱动架构有什么优势？
A: 事件驱动架构提供了松耦合、异步处理、实时响应等优势，使得系统能够灵活地响应各种变化。

## 版本历史

- **v0.0.1**: 基础框架实现，包含核心模块
- **待实现**: Etcd集成、网络通信、日志管理等

## 贡献指南

欢迎提交Issue和Pull Request来改进这个项目。

## 许可证

本项目采用 Apache License 2.0 许可证。 