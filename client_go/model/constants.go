package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// 常量定义，与Java版本保持一致

const (
	// MagicNumber 魔数，对应Java的Constant.MAGIC_NUMBER
	MagicNumber = 0x12fcf76

	// Delimiter netty的分隔符，对应Java的Constant.DELIMITER
	Delimiter = "$(* *)$"

	// CountDelimiter 数量统计时，rule+时间 组成key用的分隔符
	CountDelimiter = "#**#"

	// BakDelimiter 备用分隔符
	BakDelimiter = "#\\*\\*#"

	// DefaultDeleteValue 当客户端要删除某个key时，就往etcd里赋值这个value，设置1秒过期，就算删除了
	DefaultDeleteValue = "#[DELETE]#"

	// MaxLength 单次包最大4M
	MaxLength = 4 * 1024 * 1024

	// DefaultThreads 默认线程数
	DefaultThreads = 4

	// PingMessage 心跳ping消息内容
	PingMessage = "ping"

	// PongMessage 心跳pong消息内容
	PongMessage = "pong"
)

// Etcd路径常量，对应Java的ConfigConstant
const (
	// AppsPath 所有的app名字，存这里
	AppsPath = "/jd/apps/"

	// WorkersPath 所有的workers，存这里
	WorkersPath = "/jd/workers/"

	// DashboardPath dashboard的ip存这里
	DashboardPath = "/jd/dashboard/"

	// RulePath 所有的客户端规则（譬如哪个app的哪些前缀的才参与计算）
	RulePath = "/jd/rules/"

	// WhiteListPath 白名单路径，白名单的不参与热key计算
	WhiteListPath = "/jd/whiteList/"

	// ClientCountPath 客户端数量
	ClientCountPath = "/jd/count/"

	// HotKeyPath 每个app的热key放这里。格式如：jd/hotkeys/app1/userA
	HotKeyPath = "/jd/hotkeys/"

	// HotKeyRecordPath 每个app的热key记录放这里，供控制台监听入库用
	HotKeyRecordPath = "/jd/keyRecords/"

	// CaffeineSizePath caffeine的size
	CaffeineSizePath = "/jd/caffeineSize/"

	// TotalReceiveKeyCount totalReceiveKeyCount该worker接收到的key总量，每10秒上报一次
	TotalReceiveKeyCount = "/jd/totalKeyCount/"

	// BufferPoolPath bufferPool直接内存
	BufferPoolPath = "/jd/bufferPool/"

	// KeyHitCountPath 存放客户端hotKey访问次数和总访问次数的path
	KeyHitCountPath = "/jd/keyHitCount/"

	// LogToggle 是否开启日志
	LogToggle = "/jd/logOn"

	// ClearCfgPath 清理历史数据的配置的path
	ClearCfgPath = "/jd/clearCfg/"

	// AppCfgPath app配置
	AppCfgPath = "/jd/appCfg/"

	// DashboardPort 控制台启动的netty端口
	DashboardPort = 11112
)

// 默认配置值
const (
	// DefaultPushPeriod 默认推送间隔（毫秒）
	DefaultPushPeriod = 500

	// DefaultCacheSize 默认缓存大小
	DefaultCacheSize = 200000

	// DefaultCountPeriod 默认计数推送间隔（秒）
	DefaultCountPeriod = 10

	// DefaultWorkerRetryInterval 默认worker重连间隔（秒）
	DefaultWorkerRetryInterval = 30

	// DefaultEtcdFetchInterval 默认etcd拉取间隔（秒）
	DefaultEtcdFetchInterval = 30

	// DefaultRuleFetchInterval 默认规则拉取间隔（秒）
	DefaultRuleFetchInterval = 5

	// DefaultHeartbeatInterval 默认心跳间隔（秒）
	DefaultHeartbeatInterval = 30

	// DataConvertSwitchThreshold 数据转换并行处理阈值
	DataConvertSwitchThreshold = 5000
)

// generateID 生成唯一ID，对应Java的IdGenerater.generateId()
func generateID() string {
	// 使用时间戳 + 随机数生成ID
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	return hex.EncodeToString(append(
		[]byte{byte(timestamp), byte(timestamp >> 8), byte(timestamp >> 16), byte(timestamp >> 24)},
		randomBytes...,
	))
}

// GetHotKeyPath 获取热key路径
func GetHotKeyPath(appName string) string {
	return HotKeyPath + appName
}

// GetHotKeyRecordPath 获取热key记录路径
func GetHotKeyRecordPath(appName string) string {
	return HotKeyRecordPath + appName
}

// GetRulePath 获取规则路径
func GetRulePath(appName string) string {
	return RulePath + appName
}

// GetWorkersPath 获取workers路径
func GetWorkersPath(appName string) string {
	return WorkersPath + appName
}

// GetDefaultWorkersPath 获取默认workers路径
func GetDefaultWorkersPath() string {
	return WorkersPath + "default"
}

// GetKeyPath 获取具体key的路径，对应Java的HotKeyPathTool.keyPath
func GetKeyPath(appName, key string) string {
	return GetHotKeyPath(appName) + "/" + key
}

// GetKeyRecordPath 获取具体key记录的路径，对应Java的HotKeyPathTool.keyRecordPath
func GetKeyRecordPath(appName, key string) string {
	return GetHotKeyRecordPath(appName) + "/" + key
}
