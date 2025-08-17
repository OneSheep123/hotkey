package model

import (
	"sync/atomic"
	"time"
)

// MessageType 消息类型枚举，与Java版本对应
type MessageType byte

const (
	AppName         MessageType = 1
	RequestNewKey   MessageType = 2
	ResponseNewKey  MessageType = 3
	Ping            MessageType = 4
	Pong            MessageType = 5
	Empty           MessageType = 6
	RequestHitCount MessageType = 7
	RequestHotKey   MessageType = 8
)

// KeyType key类型枚举，与Java版本对应
type KeyType int

const (
	RedisKey KeyType = iota
	RequestPath
	BlackList
	Other
)

// String 返回KeyType的字符串表示
func (kt KeyType) String() string {
	switch kt {
	case RedisKey:
		return "REDIS_KEY"
	case RequestPath:
		return "REQUEST_PATH"
	case BlackList:
		return "BLACK_LIST"
	case Other:
		return "OTHER"
	default:
		return "UNKNOWN"
	}
}

// BaseModel 基础模型，对应Java的BaseModel
type BaseModel struct {
	ID         string `json:"id"`
	CreateTime int64  `json:"createTime"`
	Key        string `json:"key"`
	count      int64  // 使用atomic操作，对应Java的LongAdder
}

// GetCount 获取计数总数
func (bm *BaseModel) GetCount() int64 {
	return atomic.LoadInt64(&bm.count)
}

// SetCount 设置计数
func (bm *BaseModel) SetCount(count int64) {
	atomic.StoreInt64(&bm.count, count)
}

// Add 计数自增指定数量
func (bm *BaseModel) Add(count int64) {
	atomic.AddInt64(&bm.count, count)
}

// HotKeyModel 热key模型，对应Java的HotKeyModel
type HotKeyModel struct {
	BaseModel
	AppName string  `json:"appName"`
	KeyType KeyType `json:"keyType"`
	Remove  bool    `json:"remove"`
}

// NewHotKeyModel 创建新的HotKeyModel
func NewHotKeyModel(key, appName string, keyType KeyType) *HotKeyModel {
	return &HotKeyModel{
		BaseModel: BaseModel{
			ID:         generateID(),
			CreateTime: time.Now().UnixMilli(),
			Key:        key,
		},
		AppName: appName,
		KeyType: keyType,
		Remove:  false,
	}
}

// KeyCountModel 访问计数模型，对应Java的KeyCountModel
type KeyCountModel struct {
	RuleKey       string `json:"ruleKey"`       // 对应的规则名
	TotalHitCount int    `json:"totalHitCount"` // 总访问次数
	HotHitCount   int    `json:"hotHitCount"`   // 热后访问次数
	CreateTime    int64  `json:"createTime"`    // 发送时的时间
}

// HotKeyMsg 网络通信消息，对应Java的HotKeyMsg
type HotKeyMsg struct {
	MagicNumber    int              `json:"magicNumber"`
	AppName        string           `json:"appName"`
	MessageType    MessageType      `json:"messageType"`
	Body           string           `json:"body"`
	HotKeyModels   []*HotKeyModel   `json:"hotKeyModels"`
	KeyCountModels []*KeyCountModel `json:"keyCountModels"`
}

// NewHotKeyMsg 创建新的HotKeyMsg
// 注意：Java版本不设置MagicNumber（默认为0），为了兼容性，Go版本也不设置
func NewHotKeyMsg(msgType MessageType, appName string) *HotKeyMsg {
	return &HotKeyMsg{
		MagicNumber: 0, // 与Java版本保持一致，不设置MagicNumber
		MessageType: msgType,
		AppName:     appName,
	}
}

// KeyRule 规则模型，对应Java的KeyRule
type KeyRule struct {
	Key       string `json:"key"`       // key的前缀，也可以完全和key相同。为"*"时代表通配符
	Prefix    bool   `json:"prefix"`    // 是否是前缀，true是前缀
	Interval  int    `json:"interval"`  // 间隔时间（秒）
	Threshold int    `json:"threshold"` // 累计数量
	Duration  int    `json:"duration"`  // 变热key后，本地、etcd缓存它多久。单位（秒），默认60
	Desc      string `json:"desc"`      // 描述
}

// NewKeyRule 创建新的KeyRule
func NewKeyRule(key string, prefix bool, interval, threshold, duration int) *KeyRule {
	return &KeyRule{
		Key:       key,
		Prefix:    prefix,
		Interval:  interval,
		Threshold: threshold,
		Duration:  duration,
	}
}

// ValueModel 本地缓存值模型，对应Java的ValueModel
type ValueModel struct {
	CreateTime int64       `json:"createTime"` // 该热key创建时间
	Duration   int         `json:"duration"`   // 本地缓存时间，单位毫秒
	Value      interface{} `json:"value"`      // 用户实际存放的value
}

// NewValueModel 创建新的ValueModel
func NewValueModel(duration int) *ValueModel {
	return &ValueModel{
		CreateTime: time.Now().UnixMilli(),
		Duration:   duration * 1000, // 转换为毫秒
		Value:      MagicNumber,     // 默认使用魔数
	}
}

// IsExpired 检查是否过期
func (vm *ValueModel) IsExpired() bool {
	return time.Now().UnixMilli() > vm.CreateTime+int64(vm.Duration)
}

// IsNearExpire 检查是否临近过期（2秒内）
func (vm *ValueModel) IsNearExpire() bool {
	return vm.CreateTime+int64(vm.Duration)-time.Now().UnixMilli() <= 2000
}

// KeyHotModel 用于统计的热key模型
type KeyHotModel struct {
	Key   string `json:"key"`
	IsHot bool   `json:"isHot"`
}

// NewKeyHotModel 创建新的KeyHotModel
func NewKeyHotModel(key string, isHot bool) *KeyHotModel {
	return &KeyHotModel{
		Key:   key,
		IsHot: isHot,
	}
}
