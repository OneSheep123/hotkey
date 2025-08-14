package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/jd/platform/hotkey/client-go/model"
)

// ConfigCenter Etcd配置中心，对应Java的IConfigCenter
type ConfigCenter struct {
	client   *clientv3.Client
	timeout  time.Duration
	appName  string
	watchers map[string]context.CancelFunc
}

// NewConfigCenter 创建新的配置中心
func NewConfigCenter(endpoints []string, appName string) (*ConfigCenter, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %v", err)
	}

	return &ConfigCenter{
		client:   client,
		timeout:  5 * time.Second,
		appName:  appName,
		watchers: make(map[string]context.CancelFunc),
	}, nil
}

// Close 关闭配置中心
func (cc *ConfigCenter) Close() error {
	// 取消所有watchers
	for _, cancel := range cc.watchers {
		cancel()
	}
	return cc.client.Close()
}

// Get 获取单个key的值
func (cc *ConfigCenter) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cc.timeout)
	defer cancel()

	resp, err := cc.client.Get(ctx, key)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", nil
	}

	return string(resp.Kvs[0].Value), nil
}

// GetPrefix 获取指定前缀的所有key-value
func (cc *ConfigCenter) GetPrefix(prefix string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cc.timeout)
	defer cancel()

	resp, err := cc.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = string(kv.Value)
	}

	return result, nil
}

// Put 设置key-value
func (cc *ConfigCenter) Put(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cc.timeout)
	defer cancel()

	_, err := cc.client.Put(ctx, key, value)
	return err
}

// PutWithTTL 设置key-value并指定TTL
func (cc *ConfigCenter) PutWithTTL(key, value string, ttlSeconds int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), cc.timeout)
	defer cancel()

	// 创建lease
	lease, err := cc.client.Grant(ctx, ttlSeconds)
	if err != nil {
		return err
	}

	// 设置key-value with lease
	_, err = cc.client.Put(ctx, key, value, clientv3.WithLease(lease.ID))
	return err
}

// Delete 删除key
func (cc *ConfigCenter) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cc.timeout)
	defer cancel()

	_, err := cc.client.Delete(ctx, key)
	return err
}

// Watch 监听单个key的变化
func (cc *ConfigCenter) Watch(key string, callback func(event *WatchEvent)) error {
	ctx, cancel := context.WithCancel(context.Background())
	watchChan := cc.client.Watch(ctx, key)
	cc.watchers[key] = cancel

	go func() {
		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				watchEvent := &WatchEvent{
					Type:  EventType(event.Type),
					Key:   string(event.Kv.Key),
					Value: string(event.Kv.Value),
				}
				callback(watchEvent)
			}
		}
	}()

	return nil
}

// WatchPrefix 监听指定前缀的key变化
func (cc *ConfigCenter) WatchPrefix(prefix string, callback func(event *WatchEvent)) error {
	ctx, cancel := context.WithCancel(context.Background())
	watchChan := cc.client.Watch(ctx, prefix, clientv3.WithPrefix())
	cc.watchers[prefix] = cancel

	go func() {
		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				watchEvent := &WatchEvent{
					Type:  EventType(event.Type),
					Key:   string(event.Kv.Key),
					Value: string(event.Kv.Value),
				}
				callback(watchEvent)
			}
		}
	}()

	return nil
}

// WatchEvent 监听事件
type WatchEvent struct {
	Type  EventType
	Key   string
	Value string
}

// EventType 事件类型
type EventType int32

const (
	EventTypePut    EventType = 0
	EventTypeDelete EventType = 1
)

// GetWorkerAddresses 获取worker地址列表，对应Java的EtcdStarter.fetch
func (cc *ConfigCenter) GetWorkerAddresses() ([]string, error) {
	// 首先尝试获取应用专用的worker
	appWorkerPath := model.GetWorkersPath(cc.appName)
	workers, err := cc.GetPrefix(appWorkerPath)
	if err != nil {
		return nil, err
	}

	// 如果应用专用worker为空，尝试获取默认worker
	if len(workers) == 0 {
		defaultWorkerPath := model.GetDefaultWorkersPath()
		workers, err = cc.GetPrefix(defaultWorkerPath)
		if err != nil {
			return nil, err
		}
	}

	// 提取地址列表
	var addresses []string
	for _, address := range workers {
		if address != "" {
			addresses = append(addresses, address)
		}
	}

	return addresses, nil
}

// GetRules 获取应用的规则列表，对应Java的EtcdStarter.fetchRuleFromEtcd
func (cc *ConfigCenter) GetRules() ([]*model.KeyRule, error) {
	rulePath := model.GetRulePath(cc.appName)
	rulesJson, err := cc.Get(rulePath)
	if err != nil {
		return nil, err
	}

	if rulesJson == "" {
		log.Printf("Warning: rule is empty for app %s", cc.appName)
		return []*model.KeyRule{}, nil
	}

	var rules []*model.KeyRule
	err = json.Unmarshal([]byte(rulesJson), &rules)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rules: %v", err)
	}

	return rules, nil
}

// GetExistingHotKeys 获取现有的热key列表，对应Java的EtcdStarter.fetchExistHotKey
func (cc *ConfigCenter) GetExistingHotKeys() (map[string]int64, error) {
	hotKeyPath := model.GetHotKeyPath(cc.appName)
	hotKeys, err := cc.GetPrefix(hotKeyPath)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	prefix := hotKeyPath + "/"
	for fullKey, value := range hotKeys {
		// 提取实际的key名称
		if strings.HasPrefix(fullKey, prefix) {
			key := strings.TrimPrefix(fullKey, prefix)
			if key != "" {
				// 尝试解析时间戳
				if timestamp, err := parseTimestamp(value); err == nil {
					result[key] = timestamp
				} else {
					// 如果不是时间戳，使用当前时间
					result[key] = time.Now().UnixMilli()
				}
			}
		}
	}

	return result, nil
}

// parseTimestamp 解析时间戳字符串
func parseTimestamp(value string) (int64, error) {
	// 如果是删除标记，跳过
	if value == model.DefaultDeleteValue {
		return 0, fmt.Errorf("delete marker")
	}

	// 尝试解析为时间戳
	if timestamp, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return timestamp.UnixMilli(), nil
	}

	// 尝试解析为Unix毫秒时间戳
	if timestamp, err := time.ParseDuration(value + "ms"); err == nil {
		return int64(timestamp.Milliseconds()), nil
	}

	return time.Now().UnixMilli(), nil
}

// RemoveHotKey 删除热key，对应Java的HotKeyPusher.remove
func (cc *ConfigCenter) RemoveHotKey(key string) error {
	keyPath := model.GetKeyPath(cc.appName, key)
	recordPath := model.GetKeyRecordPath(cc.appName, key)

	// 先设置删除标记（TTL=1秒）
	err := cc.PutWithTTL(keyPath, model.DefaultDeleteValue, 1)
	if err != nil {
		return err
	}

	// 删除key
	err = cc.Delete(keyPath)
	if err != nil {
		return err
	}

	// 删除记录
	return cc.Delete(recordPath)
}
