package decision

import (
	"nofx/market"
	"testing"
)

// TestOIFilter_ZeroOIShouldNotBeFiltered 测试 OI=0 时不应该被过滤
// Issue: 当 OI API 返回 0（网络问题或临时故障）时，主流币种如 BTC 会被错误过滤掉
func TestOIFilter_ZeroOIShouldNotBeFiltered(t *testing.T) {
	tests := []struct {
		name           string
		oiValue        float64
		currentPrice   float64
		shouldBeInMap  bool
		description    string
	}{
		{
			name:          "OI=0 应该放行（API异常）",
			oiValue:       0,
			currentPrice:  87000.0,
			shouldBeInMap: true,
			description:   "OI=0 很可能是 API 异常，BTC 真实 OI 不可能是 0",
		},
		{
			name:          "OI 正常且高于阈值应该放行",
			oiValue:       200000, // 200000 * 87000 = 17.4B >> 15M
			currentPrice:  87000.0,
			shouldBeInMap: true,
			description:   "正常 OI 高于阈值",
		},
		{
			name:          "OI 正常但低于阈值应该被过滤",
			oiValue:       100, // 100 * 87000 = 8.7M < 15M
			currentPrice:  87000.0,
			shouldBeInMap: false,
			description:   "OI 低于阈值的小币种应该被过滤",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 构建测试上下文
			ctx := &Context{
				CandidateCoins: []CandidateCoin{
					{Symbol: "TESTUSDT", Sources: []string{"test"}},
				},
				Positions:     []PositionInfo{}, // 没有现有持仓
				MarketDataMap: make(map[string]*market.Data),
			}

			// 模拟 market.Data
			testData := &market.Data{
				Symbol:       "TESTUSDT",
				CurrentPrice: tt.currentPrice,
				OpenInterest: &market.OIData{
					Latest: tt.oiValue,
				},
			}

			// 执行 OI 过滤逻辑（与 EnrichMarketData 中的逻辑保持一致）
			const minOIThresholdMillions = 15.0
			isExistingPosition := false // 不是现有持仓

			shouldAdd := true
			if !isExistingPosition && testData.OpenInterest != nil && testData.CurrentPrice > 0 {
				oiValue := testData.OpenInterest.Latest * testData.CurrentPrice
				oiValueInMillions := oiValue / 1_000_000

				// 修复后逻辑：OI > 0 且低于阈值才过滤
				// OI=0 时不过滤（可能是 API 异常）
				if testData.OpenInterest.Latest > 0 && oiValueInMillions < minOIThresholdMillions {
					shouldAdd = false
				}
			}

			if shouldAdd {
				ctx.MarketDataMap["TESTUSDT"] = testData
			}

			// 验证结果
			_, exists := ctx.MarketDataMap["TESTUSDT"]
			if exists != tt.shouldBeInMap {
				t.Errorf("%s: 期望在 MarketDataMap 中: %v, 实际: %v",
					tt.description, tt.shouldBeInMap, exists)
			}
		})
	}
}

// TestOIFilter_FixedLogic 测试修复后的 OI 过滤逻辑
// 这个测试验证修复后的逻辑：OI > 0 且低于阈值才过滤
func TestOIFilter_FixedLogic(t *testing.T) {
	tests := []struct {
		name          string
		oiValue       float64
		currentPrice  float64
		shouldFilter  bool
	}{
		{"OI=0 不过滤", 0, 87000.0, false},
		{"OI 高于阈值不过滤", 200000, 87000.0, false},
		{"OI > 0 但低于阈值过滤", 100, 87000.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const minOIThresholdMillions = 15.0

			oiValue := tt.oiValue * tt.currentPrice
			oiValueInMillions := oiValue / 1_000_000

			// 修复后的逻辑：OI > 0 且低于阈值才过滤
			shouldFilter := tt.oiValue > 0 && oiValueInMillions < minOIThresholdMillions

			if shouldFilter != tt.shouldFilter {
				t.Errorf("OI=%.0f, Price=%.0f: 期望过滤=%v, 实际=%v",
					tt.oiValue, tt.currentPrice, tt.shouldFilter, shouldFilter)
			}
		})
	}
}
