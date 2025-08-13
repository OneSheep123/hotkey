# HotKey Client 模块

## 概述

HotKey Client 是一个高性能的热key探测和本地缓存客户端，能够自动识别应用中的热点数据并将其缓存在本地内存中，显著提升数据访问性能。

## 核心功能

### 1. 热Key自动探测
- **智能收集**: 自动收集应用中的key访问信息
- **批量推送**: 定期将key信息批量推送到worker节点进行分析
- **规则过滤**: 只上报符合配置规则的key，避免无效传输
- **频率控制**: 可配置推送间隔，平衡探测及时性和资源消耗

### 2. 本地缓存管理
- **高性能缓存**: 基于Caffeine实现的高性能本地缓存
- **热Key存储**: 自动将探测到的热key存储在本地内存中
- **过期管理**: 支持TTL过期时间，自动清理过期数据
- **容量控制**: 可配置缓存最大容量，防止内存溢出

### 3. 配置中心集成
- **Etcd支持**: 与etcd配置中心无缝集成
- **动态配置**: 实时监听配置变化，支持热更新
- **多节点支持**: 支持连接多个etcd节点，实现高可用
- **规则同步**: 自动同步key探测规则和worker节点信息

### 4. 网络通信
- **Netty客户端**: 基于Netty的高性能网络通信
- **连接管理**: 自动管理worker节点的连接状态
- **负载均衡**: 通过hash算法智能分发key到不同worker
- **断线重连**: 自动检测连接异常并尝试重连

### 5. 事件驱动架构
- **事件总线**: 基于EventBus的事件驱动机制
- **异步处理**: 支持异步事件处理，提升系统响应性
- **事件订阅**: 支持多种事件类型的订阅和处理
- **实时响应**: 能够实时响应配置和状态变化
- **解耦合设计**: 通过事件总线解耦各个组件，便于维护和扩展

## 快速开始

### 1. 添加依赖

```xml
<dependency>
    <groupId>com.jd.platform.hotkey</groupId>
    <artifactId>hotkey-client</artifactId>
    <version>${version}</version>
</dependency>
```

### 2. 初始化客户端

```java
@PostConstruct
public void initHotKey() {
    ClientStarter.Builder builder = new ClientStarter.Builder();
    ClientStarter starter = builder
        .setAppName("your-app-name")
        .setEtcdServer("http://127.0.0.1:2379")
        .setPushPeriod(500L)        // 推送间隔，默认500ms
        .setCaffeineSize(200000)    // 缓存容量，默认20万
        .build();
    starter.startPipeline();
}
```

### 3. 使用热Key功能

```java
// 判断key是否为热key
if (JdHotKeyStore.isHotKey("user:123")) {
    // 从本地缓存获取值
    Object value = JdHotKeyStore.get("user:123");
    // 处理热key逻辑
}

// 智能设置值（仅当key是热key时）
JdHotKeyStore.smartSet("user:123", userInfo);

// 强制设置值
JdHotKeyStore.forceSet("user:123", userInfo);

// 删除key
JdHotKeyStore.remove("user:123");
```

## 配置说明

### 应用配置

```yaml
etcd:
  server: ${etcdServer:http://127.0.0.1:2379}

spring:
  application:
    name: ${name:your-app-name}
```

### 客户端配置参数

| 参数 | 说明 | 默认值 | 建议值 |
|------|------|--------|--------|
| appName | 应用名称 | 必填 | 建议使用有意义的应用标识 |
| etcdServer | etcd服务器地址 | 必填 | 支持多个地址，逗号分隔 |
| pushPeriod | 推送间隔(毫秒) | 500ms | 根据QPS调整，建议100-1000ms |
| caffeineSize | 本地缓存容量 | 200000 | 根据内存情况调整，建议5万-50万 |

## 架构设计

### 核心组件

```
ClientStarter (启动器)
├── EtcdStarter (配置中心管理)
├── PushSchedulerStarter (推送调度器)
├── WorkerRetryConnector (Worker重连器)
└── EventBusCenter (事件总线)
```

### 事件驱动架构

#### 事件类型与订阅者

| 事件类型 | 订阅者 | 触发时机 | 处理逻辑 |
|----------|--------|----------|----------|
| **WorkerInfoChangeEvent** | WorkerChangeSubscriber | Worker节点信息变化 | 连接新的Worker节点，管理连接状态 |
| **ReceiveNewKeyEvent** | ReceiveNewKeySubscribe | 接收到新的热Key | 更新本地缓存，处理热Key的增删 |
| **KeyRuleInfoChangeEvent** | KeyRuleHolder | Key规则配置变化 | 重建本地规则缓存，更新探测策略 |

#### 事件触发流程

```
etcd配置变化 → EtcdStarter监听 → 发布事件 → EventBusCenter → 订阅者处理
    ↓
Worker节点变化 → WorkerInfoChangeEvent → WorkerChangeSubscriber → 连接管理
    ↓
热Key变化 → ReceiveNewKeyEvent → ReceiveNewKeySubscribe → 缓存更新
    ↓
规则变化 → KeyRuleInfoChangeEvent → KeyRuleHolder → 规则重建
```

#### 事件来源详解

1. **Worker信息变化事件**
   - 来源：etcd中Worker节点注册信息变化
   - 处理：自动连接新Worker，清理断开的连接
   - 作用：实现Worker节点的动态发现和故障转移

2. **热Key事件**
   - 来源1：etcd中手工添加/删除的热Key
   - 来源2：Worker节点推送的探测结果
   - 处理：更新本地缓存，同步热Key状态
   - 作用：保持本地缓存与分布式状态的一致性

3. **规则变化事件**
   - 来源：etcd中Key探测规则配置变化
   - 处理：重建本地规则缓存，更新缓存策略
   - 作用：支持运行时动态调整探测规则

### 数据流

```
应用访问Key → 本地缓存查询 → 热Key判断 → 上报Worker → 规则匹配 → 本地缓存更新
```

### 缓存策略

- **L1缓存**: 本地Caffeine缓存，提供纳秒级访问
- **L2缓存**: 分布式Worker缓存，提供全局热Key共享
- **过期策略**: TTL过期 + 访问频率衰减

## 性能特性

### 缓存性能
- **读取性能**: 纳秒级本地缓存访问
- **写入性能**: 异步批量推送，最小化性能影响
- **内存效率**: 智能内存管理，支持大容量缓存

### 网络性能
- **批量传输**: 批量推送减少网络开销
- **连接复用**: 长连接复用，减少连接建立开销
- **负载均衡**: 智能分发，充分利用Worker资源

## 高可用特性

### 故障容错
- **Worker故障转移**: 自动检测Worker故障并切换
- **Etcd高可用**: 支持多Etcd节点，避免单点故障
- **连接重试**: 自动重连机制，保证服务连续性

### 配置热更新
- **规则热更新**: 支持运行时更新探测规则
- **Worker动态发现**: 自动发现新增Worker节点
- **配置同步**: 实时同步配置变更

## 监控与运维

### 关键指标
- **缓存命中率**: 本地缓存命中情况
- **推送延迟**: key上报到Worker的延迟时间
- **连接状态**: Worker节点连接健康状态
- **内存使用**: 本地缓存内存占用情况

### 日志管理
- **结构化日志**: 提供详细的运行日志
- **错误追踪**: 完整的错误堆栈和上下文信息
- **性能日志**: 关键操作的性能统计信息

## 最佳实践

### 1. 应用集成
- 在应用启动时初始化客户端
- 使用Spring的@PostConstruct注解确保初始化顺序
- 配置合适的推送间隔和缓存容量

### 2. 事件处理优化
- 合理配置事件监听器，避免阻塞主流程
- 监控事件处理性能，及时发现性能瓶颈
- 利用事件驱动的异步特性，提升系统响应性

### 3. 性能优化
- 根据应用QPS调整推送间隔
- 合理设置缓存容量，避免内存溢出
- 监控缓存命中率，优化key规则配置

### 4. 高可用部署
- 配置多个Etcd节点
- 部署多个Worker节点
- 监控系统健康状态

### 5. 故障处理
- 配置合适的重连策略
- 监控网络连接状态
- 设置合理的超时时间

## 常见问题

### Q: 如何判断key是否为热key？
A: 使用`JdHotKeyStore.isHotKey(key)`方法，该方法会自动上报key信息并返回是否为热key。

### Q: 推送间隔设置多少合适？
A: 建议根据应用QPS设置，单机QPS 10个时建议500ms，QPS 100个时建议100ms。

### Q: 本地缓存容量如何设置？
A: 根据可用内存和预期热key数量设置，建议5万-50万之间，默认20万。

### Q: 支持哪些配置中心？
A: 目前支持Etcd，可通过扩展IConfigCenter接口支持其他配置中心。

### Q: 事件驱动架构有什么优势？
A: 事件驱动架构提供了松耦合、异步处理、实时响应等优势，使得系统能够灵活地响应各种变化，同时保持代码的清晰和可维护性。

### Q: 如何处理事件处理失败的情况？
A: 系统提供了完善的异常处理机制，事件处理失败不会影响主流程，同时会记录详细的错误日志便于问题排查。

## 版本历史

- **v0.0.4**: 支持动态配置更新、Worker故障转移
- **v0.0.3**: 优化缓存性能、增加批量推送
- **v0.0.2**: 支持多Worker负载均衡
- **v0.0.1**: 基础热key探测功能

## 贡献指南

欢迎提交Issue和Pull Request来改进这个项目。

## 许可证

本项目采用 Apache License 2.0 许可证。 