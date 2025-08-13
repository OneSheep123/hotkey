package main

import (
	"fmt"
	"hotkey-client/callback"
	"time"
)

func main() {
	fmt.Println("HotKey Client Go 版本示例")

	// 创建客户端启动器（这里需要先实现Builder）
	// starter := NewBuilder().
	// 	SetAppName("example-app").
	// 	SetEtcdServers([]string{"127.0.0.1:2379"}).
	// 	SetPushPeriod(500 * time.Millisecond).
	// 	SetCacheSize(100000).
	// 	Build()

	// 暂时跳过启动器，直接演示热Key功能
	fmt.Println("注意: 启动器功能待完善，直接演示热Key功能")

	// 启动管道（暂时跳过）
	// err := starter.StartPipeline()
	// if err != nil {
	// 	log.Fatalf("启动失败: %v", err)
	// }

	// fmt.Println("HotKey Client 启动成功")
	fmt.Println("跳过启动器，直接演示功能")

	// 使用热Key功能
	store := callback.GetInstance()

	// 模拟一些key访问
	keys := []string{"user:123", "product:456", "order:789", "cart:101"}

	for i, key := range keys {
		fmt.Printf("检查Key %d: %s\n", i+1, key)

		// 判断key是否为热key
		if store.IsHotKey(key) {
			fmt.Printf("  ✓ %s 是热Key\n", key)

			// 从本地缓存获取值
			value := store.Get(key)
			if value != nil {
				fmt.Printf("  缓存值: %v\n", value)
			}
		} else {
			fmt.Printf("  ✗ %s 不是热Key\n", key)
		}

		// 模拟一些延迟
		time.Sleep(100 * time.Millisecond)
	}

	// 演示设置值
	fmt.Println("\n演示设置值:")

	// 智能设置值（仅当key是热key时）
	store.SmartSet("user:123", "用户信息")
	fmt.Println("  智能设置 user:123")

	// 强制设置值
	store.ForceSet("product:456", "产品信息")
	fmt.Println("  强制设置 product:456")

	// 再次检查
	fmt.Println("\n再次检查:")
	for _, key := range keys[:2] {
		if store.IsHotKey(key) {
			value := store.Get(key)
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// 演示删除
	fmt.Println("\n演示删除:")
	store.Remove("user:123")
	fmt.Println("  删除 user:123")

	if !store.IsHotKey("user:123") {
		fmt.Println("  ✓ user:123 已成功删除")
	}

	// 保持程序运行一段时间
	fmt.Println("\n程序运行中，按 Ctrl+C 退出...")
	time.Sleep(30 * time.Second)
}
