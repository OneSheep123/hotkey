package main

import (
	"fmt"
	"log"
	"time"

	hotkey "github.com/jd/platform/hotkey/client-go"
)

func main() {
	log.Println("🚀 HotKey Go Client - Quick Start Example")
	
	// 创建客户端
	client, err := hotkey.NewClientBuilder().
		SetAppName("quick-start").
		SetEtcdServer("http://127.0.0.1:2379").
		SetPushPeriod(500).
		SetCacheSize(100000).
		Build()
	
	if err != nil {
		log.Fatal("❌ Failed to create client:", err)
	}
	
	// 启动客户端
	if err := client.Start(); err != nil {
		log.Fatal("❌ Failed to start client:", err)
	}
	defer func() {
		log.Println("🛑 Stopping client...")
		client.Stop()
	}()
	
	// 等待客户端就绪 - 简单快速的检查
	log.Println("⏳ Waiting for client ready...")
	if err := quickWaitReady(client); err != nil {
		log.Printf("⚠️  Warning: %v", err)
		log.Println("📝 Note: Client may still work for basic operations")
	}
	
	// 快速演示
	quickDemo()
	
	log.Println("✅ Quick start completed!")
}

// quickWaitReady 快速等待客户端就绪（适合演示和开发）
func quickWaitReady(client *hotkey.Client) error {
	maxChecks := 30 // 最多检查30次（约6秒）
	
	for i := 0; i < maxChecks; i++ {
		if client.IsStarted() {
			stats := client.GetStats()
			if stats != nil {
				log.Printf("✅ Client ready! (check #%d)", i+1)
				return nil
			}
		}
		
		if i%5 == 0 && i > 0 {
			log.Printf("⏳ Still waiting... (%d/%d)", i, maxChecks)
		}
		
		time.Sleep(200 * time.Millisecond)
	}
	
	return fmt.Errorf("client not fully ready after %d checks", maxChecks)
}

// quickDemo 快速演示基本功能
func quickDemo() {
	log.Println("\n🎯 Quick Demo - Basic HotKey Operations")
	
	// 1. 基本操作
	key := "demo:user:123"
	log.Printf("🔍 Checking key: %s", key)
	
	isHot := hotkey.IsHotKey(key)
	log.Printf("   🌡️  Is hot: %v", isHot)
	
	// 2. 设置值
	log.Printf("💾 Setting value for: %s", key)
	hotkey.ForceSet(key, "user_data_123")
	
	// 3. 获取值
	value := hotkey.Get(key)
	log.Printf("📦 Retrieved value: %v", value)
	
	// 4. 再次检查（可能仍然不是热key，这是正常的）
	isHot = hotkey.IsHotKey(key)
	log.Printf("🌡️  Is hot after set: %v", isHot)
	
	// 5. 智能设置（只有在热key时才会设置）
	smartKey := "demo:product:456"
	log.Printf("🧠 Smart setting: %s", smartKey)
	hotkey.SmartSet(smartKey, "product_data_456")
	
	smartValue := hotkey.Get(smartKey)
	if smartValue != nil {
		log.Printf("✅ Smart set successful: %v", smartValue)
	} else {
		log.Printf("ℹ️  Smart set skipped (key not hot)")
	}
	
	// 6. 清理
	log.Println("🧹 Cleaning up...")
	hotkey.Remove(key)
	hotkey.Remove(smartKey)
	
	log.Println("🎉 Demo completed!")
	log.Println("\n💡 Tips:")
	log.Println("   • Keys become 'hot' based on access patterns and rules")
	log.Println("   • Use ForceSet() for guaranteed setting")
	log.Println("   • Use SmartSet() for conditional setting")
	log.Println("   • Check IsHotKey() before expensive operations")
}
