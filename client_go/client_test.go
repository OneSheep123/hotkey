package hotkey

import (
	"github.com/jd/platform/hotkey/client-go/model"
	"testing"
)

func TestClientBuilder(t *testing.T) {
	// 测试构建器
	client, err := NewClientBuilder().
		SetAppName("test-app").
		SetEtcdServer("http://127.0.0.1:2379").
		SetPushPeriod(1000).
		SetCacheSize(100000).
		Build()

	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	if client.appName != "test-app" {
		t.Errorf("Expected appName 'test-app', got '%s'", client.appName)
	}

	if client.pushPeriod != 1000 {
		t.Errorf("Expected pushPeriod 1000, got %d", client.pushPeriod)
	}
}

func TestClientBuilderValidation(t *testing.T) {
	// 测试缺少必需参数
	_, err := NewClientBuilder().
		SetEtcdServer("http://127.0.0.1:2379").
		Build()

	if err == nil {
		t.Error("Expected error when appName is missing")
	}

	_, err = NewClientBuilder().
		SetAppName("test-app").
		Build()

	if err == nil {
		t.Error("Expected error when etcdServer is missing")
	}
}

func TestHotKeyStoreBasicOperations(t *testing.T) {
	// 跳过这个测试，因为需要真实的依赖
	t.Skip("Skipping test that requires real dependencies")

}

func TestGlobalAPI(t *testing.T) {
	// 跳过这个测试，因为需要真实的依赖
	t.Skip("Skipping test that requires real dependencies")
}

func TestNewKeyListener(t *testing.T) {
	// 跳过这个测试，因为需要真实的依赖
	t.Skip("Skipping test that requires real dependencies")
}

// 测试基本模型功能
func TestHotKeyModel(t *testing.T) {
	// 测试创建HotKeyModel
	hotKeyModel := model.NewHotKeyModel("test:key", "test-app", model.RedisKey)

	if hotKeyModel.Key != "test:key" {
		t.Errorf("Expected key 'test:key', got '%s'", hotKeyModel.Key)
	}

	if hotKeyModel.AppName != "test-app" {
		t.Errorf("Expected appName 'test-app', got '%s'", hotKeyModel.AppName)
	}

	if hotKeyModel.KeyType != model.RedisKey {
		t.Errorf("Expected keyType RedisKey, got %v", hotKeyModel.KeyType)
	}

	// 测试计数功能
	hotKeyModel.Add(10)
	if hotKeyModel.GetCount() != 10 {
		t.Errorf("Expected count 10, got %d", hotKeyModel.GetCount())
	}

	hotKeyModel.SetCount(20)
	if hotKeyModel.GetCount() != 20 {
		t.Errorf("Expected count 20, got %d", hotKeyModel.GetCount())
	}
}

func TestKeyRule(t *testing.T) {
	// 测试创建KeyRule
	rule := model.NewKeyRule("user:*", true, 10, 100, 60)

	if rule.Key != "user:*" {
		t.Errorf("Expected key 'user:*', got '%s'", rule.Key)
	}

	if !rule.Prefix {
		t.Error("Expected prefix to be true")
	}

	if rule.Interval != 10 {
		t.Errorf("Expected interval 10, got %d", rule.Interval)
	}

	if rule.Threshold != 100 {
		t.Errorf("Expected threshold 100, got %d", rule.Threshold)
	}

	if rule.Duration != 60 {
		t.Errorf("Expected duration 60, got %d", rule.Duration)
	}
}

func TestValueModel(t *testing.T) {
	// 测试创建ValueModel
	valueModel := model.NewValueModel(60) // 60秒

	if valueModel.Duration != 60000 { // 应该转换为毫秒
		t.Errorf("Expected duration 60000ms, got %d", valueModel.Duration)
	}

	// 测试过期检查
	if valueModel.IsExpired() {
		t.Error("ValueModel should not be expired immediately after creation")
	}

	// 测试临近过期检查
	if valueModel.IsNearExpire() {
		t.Error("ValueModel should not be near expiry immediately after creation")
	}
}

func TestConstants(t *testing.T) {
	// 测试常量值
	if model.MagicNumber != 0x12fcf76 {
		t.Errorf("Expected MagicNumber 0x12fcf76, got %d", model.MagicNumber)
	}

	if model.Delimiter != "$(* *)$" {
		t.Errorf("Expected Delimiter '$(* *)$', got '%s'", model.Delimiter)
	}

	if model.DefaultDeleteValue != "#[DELETE]#" {
		t.Errorf("Expected DefaultDeleteValue '#[DELETE]#', got '%s'", model.DefaultDeleteValue)
	}
}

func TestKeyType(t *testing.T) {
	// 测试KeyType字符串表示
	if model.RedisKey.String() != "REDIS_KEY" {
		t.Errorf("Expected 'REDIS_KEY', got '%s'", model.RedisKey.String())
	}

	if model.RequestPath.String() != "REQUEST_PATH" {
		t.Errorf("Expected 'REQUEST_PATH', got '%s'", model.RequestPath.String())
	}

	if model.BlackList.String() != "BLACK_LIST" {
		t.Errorf("Expected 'BLACK_LIST', got '%s'", model.BlackList.String())
	}

	if model.Other.String() != "OTHER" {
		t.Errorf("Expected 'OTHER', got '%s'", model.Other.String())
	}
}

func TestHotKeyMsg(t *testing.T) {
	// 测试创建HotKeyMsg
	msg := model.NewHotKeyMsg(model.RequestNewKey, "test-app")

	if msg.MessageType != model.RequestNewKey {
		t.Errorf("Expected MessageType RequestNewKey, got %v", msg.MessageType)
	}

	if msg.AppName != "test-app" {
		t.Errorf("Expected AppName 'test-app', got '%s'", msg.AppName)
	}
}

// 基准测试

func BenchmarkIsHotKey(b *testing.B) {
	// 跳过这个基准测试，因为需要真实的依赖
	b.Skip("Skipping benchmark that requires real dependencies")
}

func BenchmarkGet(b *testing.B) {
	// 跳过这个基准测试，因为需要真实的依赖
	b.Skip("Skipping benchmark that requires real dependencies")
}

func BenchmarkForceSet(b *testing.B) {
	// 跳过这个基准测试，因为需要真实的依赖
	b.Skip("Skipping benchmark that requires real dependencies")
}

// 基本模型性能测试
func BenchmarkHotKeyModelCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.NewHotKeyModel("test:key", "test-app", model.RedisKey)
	}
}

func BenchmarkHotKeyModelCount(b *testing.B) {
	hotKeyModel := model.NewHotKeyModel("test:key", "test-app", model.RedisKey)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hotKeyModel.Add(1)
	}
}

func BenchmarkKeyRuleCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.NewKeyRule("user:*", true, 10, 100, 60)
	}
}

func BenchmarkValueModelCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.NewValueModel(60)
	}
}

func BenchmarkValueModelExpiry(b *testing.B) {
	valueModel := model.NewValueModel(60)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valueModel.IsExpired()
	}
}
