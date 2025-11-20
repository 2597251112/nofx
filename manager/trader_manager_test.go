package manager

import (
	"fmt"
	"nofx/config"
	"nofx/trader"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

// TestRemoveTrader 测试从内存中移除trader
func TestRemoveTrader(t *testing.T) {
	tm := NewTraderManager()

	// 创建一个模拟的 trader 并添加到 map
	traderID := "test-trader-123"
	tm.traders[traderID] = nil // 使用 nil 作为占位符，实际测试中只需验证删除逻辑

	// 验证 trader 存在
	if _, exists := tm.traders[traderID]; !exists {
		t.Fatal("trader 应该存在于 map 中")
	}

	// 调用 RemoveTrader
	tm.RemoveTrader(traderID)

	// 验证 trader 已被移除
	if _, exists := tm.traders[traderID]; exists {
		t.Error("trader 应该已从 map 中移除")
	}
}

// TestRemoveTrader_NonExistent 测试移除不存在的trader不会报错
func TestRemoveTrader_NonExistent(t *testing.T) {
	tm := NewTraderManager()

	// 尝试移除不存在的 trader，不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("移除不存在的 trader 不应该 panic: %v", r)
		}
	}()

	tm.RemoveTrader("non-existent-trader")
}

// TestRemoveTrader_Concurrent 测试并发移除trader的安全性
func TestRemoveTrader_Concurrent(t *testing.T) {
	tm := NewTraderManager()
	traderID := "test-trader-concurrent"

	// 添加 trader
	tm.traders[traderID] = nil

	// 并发调用 RemoveTrader
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			tm.RemoveTrader(traderID)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证 trader 已被移除
	if _, exists := tm.traders[traderID]; exists {
		t.Error("trader 应该已从 map 中移除")
	}
}

// TestGetTrader_AfterRemove 测试移除后获取trader返回错误
func TestGetTrader_AfterRemove(t *testing.T) {
	tm := NewTraderManager()
	traderID := "test-trader-get"

	// 添加 trader
	tm.traders[traderID] = nil

	// 移除 trader
	tm.RemoveTrader(traderID)

	// 尝试获取已移除的 trader
	_, err := tm.GetTrader(traderID)
	if err == nil {
		t.Error("获取已移除的 trader 应该返回错误")
	}
}

// TestAddTraderFromDB_LoadAPIKey_TableDriven 使用表驱动测试验证不同 Provider 的 API Key 加载
func TestAddTraderFromDB_LoadAPIKey_TableDriven(t *testing.T) {
	testCases := []struct {
		name           string
		provider       string
		apiKey         string
		expectedAPIKey string
	}{
		{
			name:           "Custom Provider",
			provider:       "custom",
			apiKey:         "sk-custom-test-key",
			expectedAPIKey: "sk-custom-test-key",
		},
		{
			name:           "OpenAI Provider",
			provider:       "openai",
			apiKey:         "sk-openai-test-key",
			expectedAPIKey: "sk-openai-test-key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tm := NewTraderManager()

			// 1. 准备测试数据
			traderCfg := &config.TraderRecord{
				ID:   fmt.Sprintf("trader-%s-1", tc.provider),
				Name: fmt.Sprintf("%s Trader", tc.name),
			}
			// 模拟 AI 模型
			aiModelCfg := &config.AIModelConfig{
				Provider: tc.provider,
				APIKey:   tc.apiKey,
			}
			exchangeCfg := &config.ExchangeConfig{
				ID: "binance",
			}

			// 2. Mock trader.NewAutoTrader
			var capturedConfig trader.AutoTraderConfig
			// 注意：gomonkey 是全局 patch，需要在每个子测试中应用或重置
			// 由于 t.Run 是顺序执行的，我们可以这样用
			patches := gomonkey.ApplyFunc(trader.NewAutoTrader, func(cfg trader.AutoTraderConfig, db interface{}, uid string) (*trader.AutoTrader, error) {
				capturedConfig = cfg
				return &trader.AutoTrader{}, nil
			})
			defer patches.Reset()

			// 3. 执行 AddTraderFromDB
			err := tm.AddTraderFromDB(traderCfg, aiModelCfg, exchangeCfg, "", "", 10, 20, 60, []string{}, nil, "user1")

			// 4. 验证
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedAPIKey, capturedConfig.CustomAPIKey, "%s 的 API Key 应该被正确加载到 CustomAPIKey 字段中", tc.name)
		})
	}
}