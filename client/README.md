# HotKey Client 客户端

## 项目概述

HotKey Client 是京东平台热点数据探测系统的客户端组件，用于在应用程序中集成热点Key的探测、上报和本地缓存功能。该客户端能够自动识别访问频率较高的Key，并将其上报到Worker节点进行集群级别的热点分析，同时提供高性能的本地缓存机制来优化热点数据的访问性能。

## 主要功能

### 🔥 热点Key探测
- **自动探测**: 根据配置的规则自动识别热点Key
- **手动上报**: 支持业务代码主动上报可能的热点Key
- **实时统计**: 实时统计Key的访问频次和热度

### 📊 本地缓存
- **Caffeine缓存**: 基于高性能的Caffeine缓存库
- **智能缓存**: 只缓存被识别为热点的Key
- **过期管理**: 支持TTL过期和容量限制

### 🌐 集群通信
- **Netty通信**: 使用Netty与Worker节点进行高效通信
- **Etcd配置**: 通过Etcd进行配置管理和服务发现
- **事件驱动**: 基于EventBus的事件驱动架构

### 📋 规则管理
- **动态规则**: 支持运行时动态更新探测规则
- **前缀匹配**: 支持Key前缀匹配和通配符
- **分级缓存**: 不同规则对应不同的缓存时长

## 技术架构

### 核心技术栈
- **Java 8**: 基础运行环境
- **Netty 4.1.42**: 网络通信框架
- **Caffeine 2.8.0**: 高性能本地缓存
- **Etcd**: 配置中心和服务发现
- **Guava EventBus**: 事件总线
- **FastJSON**: JSON序列化
- **Protostuff**: 高效序列化协议

### 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    HotKey Client                            │
├─────────────────────────────────────────────────────────────┤
│  Application Layer                                          │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │   HotKeyPusher  │  │  JdHotKeyStore  │                  │
│  │   (API入口)      │  │   (缓存操作)     │                  │
│  └─────────────────┘  └─────────────────┘                  │
├─────────────────────────────────────────────────────────────┤
│  Core Layer                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │   Key Handler   │  │   Rule Engine   │  │ Event Bus   │ │
│  │   (Key处理)      │  │   (规则引擎)     │  │ (事件总线)   │ │
│  └─────────────────┘  └─────────────────┘  └─────────────┘ │
├─────────────────────────────────────────────────────────────┤
│  Infrastructure Layer                                       │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │  Netty Client   │  │  Etcd Client    │  │ Local Cache │ │
│  │  (网络通信)      │  │  (配置管理)      │  │ (本地缓存)   │ │
│  └─────────────────┘  └─────────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 核心组件介绍

### 1. ClientStarter (客户端启动器)
- **职责**: 客户端的主入口，负责初始化和启动各个组件
- **配置**: 支持Builder模式进行灵活配置
- **功能**: 
  - 设置应用名称和Etcd地址
  - 配置推送周期和缓存大小
  - 启动监听管道和事件总线

### 2. HotKeyPusher (热点Key推送器)
- **职责**: 提供热点Key上报的API接口
- **功能**:
  - `push(key)`: 上报普通Key
  - `push(key, keyType, count)`: 上报指定类型和次数的Key
  - `remove(key)`: 删除热点Key
- **特性**: 支持批量聚合推送，减少网络开销

### 3. JdHotKeyStore (热点Key存储)
- **职责**: 热点Key的本地缓存管理和智能判断
- **核心方法**:
  - `isHotKey(key)`: 判断是否为热点Key
  - `get(key)`: 获取缓存值
  - `smartSet(key, value)`: 智能设置缓存
  - `getValue(key)`: 获取值并自动上报
- **特性**: 临近过期自动续期，智能缓存管理

### 4. KeyRuleHolder (规则管理器)
- **职责**: 管理热点Key的探测规则
- **规则类型**:
  - 精确匹配: `user:123`
  - 前缀匹配: `user:*`
  - 通配符: `*`
- **功能**: 动态规则更新，多级匹配策略

### 5. NettyClient (网络客户端)
- **职责**: 与Worker节点的网络通信
- **特性**:
  - 连接池管理
  - 心跳保活
  - 自动重连
  - 消息编解码

### 6. EtcdStarter (Etcd启动器)
- **职责**: Etcd配置管理和监听
- **功能**:
  - Worker节点发现
  - 规则配置监听
  - 热点Key事件监听
  - 配置变更通知

## 使用方法

### 1. 基本配置

```java
// 创建客户端实例
ClientStarter client = new ClientStarter.Builder()
    .setAppName("your-app-name")           // 设置应用名称
    .setEtcdServer("127.0.0.1:2379")      // 设置Etcd地址
    .setPushPeriod(500L)                   // 设置推送周期(毫秒)
    .setCaffeineSize(200000)               // 设置缓存大小
    .build();

// 启动客户端
client.startPipeline();
```

### 2. 热点Key操作

```java
// 上报热点Key
HotKeyPusher.push("user:12345");
HotKeyPusher.push("product:67890", KeyType.REDIS_KEY, 10);

// 判断是否为热点Key
boolean isHot = JdHotKeyStore.isHotKey("user:12345");

// 获取热点Key的值
Object value = JdHotKeyStore.get("user:12345");

// 智能设置缓存(仅对热点Key生效)
JdHotKeyStore.smartSet("user:12345", userData);

// 强制设置缓存
JdHotKeyStore.forceSet("user:12345", userData);

// 删除热点Key
JdHotKeyStore.remove("user:12345");
```

### 3. 规则配置示例

在Etcd中配置探测规则:
```json
[
  {
    "key": "user:*",
    "duration": 300,
    "prefix": true
  },
  {
    "key": "product:hot",
    "duration": 600,
    "prefix": false
  },
  {
    "key": "*",
    "duration": 120,
    "prefix": false
  }
]
```

## API 接口文档

### HotKeyPusher API

| 方法 | 参数 | 说明 |
|------|------|------|
| `push(String key)` | key: 要上报的Key | 上报普通Key，默认Redis类型，计数1 |
| `push(String key, KeyType keyType)` | key: Key名称<br>keyType: Key类型 | 上报指定类型的Key |
| `push(String key, KeyType keyType, int count)` | key: Key名称<br>keyType: Key类型<br>count: 访问次数 | 上报指定类型和次数的Key |
| `remove(String key)` | key: 要删除的Key | 删除热点Key |

### JdHotKeyStore API

| 方法 | 参数 | 返回值 | 说明 |
|------|------|--------|------|
| `isHotKey(String key)` | key: Key名称 | boolean | 判断是否为热点Key |
| `get(String key)` | key: Key名称 | Object | 获取缓存值 |
| `getValue(String key)` | key: Key名称 | Object | 获取值并自动上报 |
| `smartSet(String key, Object value)` | key: Key名称<br>value: 缓存值 | void | 智能设置缓存 |
| `forceSet(String key, Object value)` | key: Key名称<br>value: 缓存值 | void | 强制设置缓存 |
| `remove(String key)` | key: Key名称 | void | 删除缓存并通知集群 |

## 依赖项说明

### 核心依赖
- `common`: 公共组件模块，包含通信协议和工具类

### 第三方依赖
- `netty-all 4.1.42.Final`: 网络通信框架
- `caffeine 2.8.0`: 高性能本地缓存
- `etcd-java 0.0.16`: Etcd客户端
- `fastjson 1.2.83`: JSON序列化
- `hutool-all 5.1.0`: Java工具库
- `protostuff 1.7.4`: 序列化框架
- `snappy-java 1.1.7.3`: 压缩库

## 配置说明

### 客户端配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `appName` | String | 必填 | 应用名称，用于隔离不同应用的配置 |
| `etcdServer` | String | 必填 | Etcd服务器地址，格式: host:port |
| `pushPeriod` | Long | 500 | Key推送周期，单位毫秒 |
| `caffeineSize` | int | 200000 | 本地缓存最大容量 |

### Etcd配置路径

| 路径 | 说明 |
|------|------|
| `/jd/workers/{appName}/` | Worker节点信息 |
| `/jd/rules/{appName}` | 探测规则配置 |
| `/jd/hotkey/{appName}/` | 手动添加的热点Key |

## 快速开始指南

### 1. 环境准备
- Java 8+
- Etcd 3.x
- Maven 3.x

### 2. 添加依赖
```xml
<dependency>
    <groupId>com.jd.platform.hotkey</groupId>
    <artifactId>hotkey-client</artifactId>
    <version>0.0.4-SNAPSHOT</version>
</dependency>
```

### 3. 启动Etcd
```bash
etcd --listen-client-urls http://0.0.0.0:2379 \
     --advertise-client-urls http://127.0.0.1:2379
```

### 4. 配置规则
在Etcd中设置规则:
```bash
etcdctl put /jd/rules/your-app-name '[{"key":"*","duration":300,"prefix":false}]'
```

### 5. 启动客户端
```java
public class Application {
    public static void main(String[] args) {
        ClientStarter client = new ClientStarter.Builder()
            .setAppName("your-app-name")
            .setEtcdServer("127.0.0.1:2379")
            .build();

        client.startPipeline();

        // 使用示例
        HotKeyPusher.push("test:key");
        boolean isHot = JdHotKeyStore.isHotKey("test:key");
        System.out.println("Is hot key: " + isHot);
    }
}
```

### 6. 监控和调试
- 查看日志输出确认连接状态
- 使用Etcd客户端查看配置和热点Key
- 监控缓存命中率和推送频率

## 工作流程详解

### 1. 初始化流程
```
ClientStarter.startPipeline()
    ├── 设置Caffeine缓存大小
    ├── 初始化Etcd配置中心
    ├── 启动定时推送器 (Key推送 + 统计推送)
    ├── 启动Worker重连器
    ├── 注册事件总线订阅者
    └── 启动Etcd监听器
```

### 2. Key探测流程
```
业务调用 HotKeyPusher.push(key)
    ├── 检查Key是否在规则范围内
    ├── 如果在规则内，添加到收集器
    ├── 定时推送器批量发送到Worker
    ├── Worker分析后推送热点Key到Etcd
    ├── 客户端监听到热点Key事件
    └── 更新本地缓存
```

### 3. 缓存访问流程
```
业务调用 JdHotKeyStore.isHotKey(key)
    ├── 检查本地缓存是否存在
    ├── 如果不存在且在规则内，上报Key
    ├── 如果存在但临近过期，续期上报
    ├── 统计访问次数
    └── 返回是否为热点Key
```

## 设计模式和架构原则

### 1. 设计模式
- **Builder模式**: ClientStarter使用Builder模式进行配置
- **工厂模式**: CacheFactory、KeyHandlerFactory等工厂类
- **观察者模式**: 基于EventBus的事件驱动架构
- **策略模式**: 不同类型的Key收集器和推送器
- **单例模式**: NettyClient、EventBusCenter等核心组件

### 2. 架构原则
- **单一职责**: 每个类都有明确的职责边界
- **开闭原则**: 支持扩展新的缓存实现和推送策略
- **依赖倒置**: 面向接口编程，降低耦合度
- **异步处理**: 网络通信和事件处理都采用异步方式
- **容错设计**: 网络断线重连、配置变更容错等

## 注意事项

1. **应用名称**: 必须设置唯一的应用名称，避免不同应用间的配置冲突
2. **Etcd连接**: 确保Etcd服务可用，客户端会定期重试连接
3. **规则配置**: 规则变更会实时生效，无需重启应用
4. **缓存容量**: 根据实际内存情况调整缓存大小
5. **网络延迟**: 推送周期建议不要设置过小，避免网络压力
6. **线程安全**: 所有API都是线程安全的，可以在多线程环境中使用

## 性能优化建议

1. **批量推送**: 客户端会自动聚合Key进行批量推送，减少网络开销
2. **缓存预热**: 对于已知的热点Key，可以提前调用`forceSet`进行预热
3. **规则优化**: 合理设置规则的缓存时长，平衡内存使用和性能
4. **监控指标**: 关注缓存命中率、推送频率等关键指标
5. **资源限制**: 根据应用规模调整线程池和缓存大小

## 故障排查

### 常见问题

1. **连接Etcd失败**
   - 检查Etcd服务是否启动
   - 验证网络连通性
   - 确认Etcd地址配置正确

2. **热点Key不生效**
   - 检查规则配置是否正确
   - 确认Key是否匹配规则
   - 查看Worker节点是否正常

3. **缓存不命中**
   - 确认Key已被识别为热点
   - 检查缓存是否已过期
   - 验证缓存容量设置

### 日志级别
- `INFO`: 正常运行信息
- `WARN`: 警告信息，如Worker连接失败
- `ERROR`: 错误信息，如Etcd连接异常

---

更多详细信息请参考项目源码和相关文档。如有问题，请提交Issue或联系开发团队。
