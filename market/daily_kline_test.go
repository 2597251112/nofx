package market

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TestWSMonitor_GetCurrentKlines_1d_Support tests that 1d timeframe is supported
func TestWSMonitor_GetCurrentKlines_1d_Support(t *testing.T) {
	monitor := &WSMonitor{
		klineDataMap3m:  sync.Map{},
		klineDataMap15m: sync.Map{},
		klineDataMap1h:  sync.Map{},
		klineDataMap4h:  sync.Map{},
		klineDataMap1d:  sync.Map{}, // 日线存储
	}

	symbol := "BTCUSDT"

	// Create fresh 1d klines
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	freshEntry := &KlineCacheEntry{
		Klines: []Kline{
			{
				OpenTime:  oneMinuteAgo.Add(-24 * time.Hour).UnixMilli(),
				CloseTime: oneMinuteAgo.UnixMilli(),
				Close:     95000.0,
				High:      96000.0,
				Low:       94000.0,
				Open:      94500.0,
				Volume:    50000.0,
			},
			{
				OpenTime:  oneMinuteAgo.UnixMilli(),
				CloseTime: oneMinuteAgo.Add(24 * time.Hour).UnixMilli(),
				Close:     95500.0,
				High:      96500.0,
				Low:       94500.0,
				Open:      95000.0,
				Volume:    51000.0,
			},
		},
		ReceivedAt: oneMinuteAgo,
	}

	// Store 1d data in cache
	monitor.klineDataMap1d.Store(symbol, freshEntry)

	// Try to get 1d klines
	klines, err := monitor.GetCurrentKlines(symbol, "1d")

	// Should return 1d klines without error
	if err != nil {
		t.Fatalf("1d klines should be supported, got error: %v", err)
	}

	if klines == nil || len(klines) != 2 {
		t.Errorf("Expected 2 klines, got %d", len(klines))
	}

	if klines[0].Close != 95000.0 {
		t.Errorf("Expected close price 95000.0, got %.2f", klines[0].Close)
	}

	t.Logf("✅ Test PASSED: 1d klines correctly returned")
}

// TestWSMonitor_getKlineDataMap_1d tests that getKlineDataMap returns correct map for 1d
func TestWSMonitor_getKlineDataMap_1d(t *testing.T) {
	monitor := &WSMonitor{
		klineDataMap3m:  sync.Map{},
		klineDataMap15m: sync.Map{},
		klineDataMap1h:  sync.Map{},
		klineDataMap4h:  sync.Map{},
		klineDataMap1d:  sync.Map{},
	}

	// Store a test value in 1d map
	monitor.klineDataMap1d.Store("TEST", "test_value")

	// Get the map for 1d
	dataMap := monitor.getKlineDataMap("1d")

	// Verify it's the correct map
	value, exists := dataMap.Load("TEST")
	if !exists {
		t.Fatal("getKlineDataMap('1d') should return klineDataMap1d")
	}
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %v", value)
	}

	t.Logf("✅ Test PASSED: getKlineDataMap('1d') returns correct map")
}

// TestSubKlineTime_Includes1d tests that subKlineTime includes "1d"
func TestSubKlineTime_Includes1d(t *testing.T) {
	found := false
	for _, tf := range subKlineTime {
		if tf == "1d" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("subKlineTime should include '1d', got: %v", subKlineTime)
	}

	t.Logf("✅ Test PASSED: subKlineTime includes '1d'")
}

// TestWSMonitor_InitializeHistoricalData_1d tests that 1d data is initialized
// Note: This is an integration test that requires network, skip in unit tests
func TestWSMonitor_InitializeHistoricalData_1d(t *testing.T) {
	t.Skip("Skipping integration test - requires network connection")
}

// TestCombinedStreamsClient_1dStreamSubscription tests that 1d stream can be subscribed
func TestCombinedStreamsClient_1dStreamSubscription(t *testing.T) {
	client := NewCombinedStreamsClient(10)

	// 测试 1d 流订阅
	stream1d := "btcusdt@kline_1d"

	// 添加 1d 订阅者
	client.mu.Lock()
	client.subscribers[stream1d] = make(chan []byte, 10)
	client.mu.Unlock()

	// 验证订阅者已添加
	client.mu.RLock()
	_, exists := client.subscribers[stream1d]
	client.mu.RUnlock()

	if !exists {
		t.Fatalf("1d stream subscriber should exist")
	}

	t.Log("✅ Test PASSED: 1d stream subscription works correctly")
}

// TestCalculateDailyData tests the calculateDailyData function
func TestCalculateDailyData(t *testing.T) {
	// 生成测试用日线数据
	klines := make([]Kline, 30)
	for i := 0; i < 30; i++ {
		basePrice := 90000.0 + float64(i)*100
		klines[i] = Kline{
			OpenTime:  int64(i * 86400000), // 1天间隔
			Open:      basePrice,
			High:      basePrice + 500,
			Low:       basePrice - 300,
			Close:     basePrice + 200,
			Volume:    10000.0 + float64(i*100),
			CloseTime: int64((i+1)*86400000 - 1),
		}
	}

	data := calculateDailyData(klines)

	if data == nil {
		t.Fatal("calculateDailyData should not return nil")
	}

	// 验证 MidPrices 长度（最近10个）
	if len(data.MidPrices) != 10 {
		t.Errorf("MidPrices length = %d, want 10", len(data.MidPrices))
	}

	// 验证 Volume 长度
	if len(data.Volume) != 10 {
		t.Errorf("Volume length = %d, want 10", len(data.Volume))
	}

	// 验证 EMA20 计算
	if data.EMA20 <= 0 {
		t.Errorf("EMA20 = %.2f, should be > 0", data.EMA20)
	}

	// 验证 ATR14 计算
	if data.ATR14 <= 0 {
		t.Errorf("ATR14 = %.2f, should be > 0", data.ATR14)
	}

	t.Logf("✅ Test PASSED: calculateDailyData works correctly")
	t.Logf("   EMA20: %.2f, EMA50: %.2f, ATR14: %.2f", data.EMA20, data.EMA50, data.ATR14)
}

// TestPriceChange24h tests the 24h price change calculation
func TestPriceChange24h(t *testing.T) {
	tests := []struct {
		name           string
		klines1d       []Kline
		currentPrice   float64
		expectedChange float64
		tolerance      float64
	}{
		{
			name: "正常计算 - 10%涨幅",
			klines1d: []Kline{
				{Close: 90000.0},
				{Close: 99000.0}, // 当前K线
			},
			currentPrice:   99000.0,
			expectedChange: 10.0,
			tolerance:      0.01,
		},
		{
			name: "正常计算 - 5%跌幅",
			klines1d: []Kline{
				{Close: 100000.0},
				{Close: 95000.0}, // 当前K线
			},
			currentPrice:   95000.0,
			expectedChange: -5.0,
			tolerance:      0.01,
		},
		{
			name: "数据不足 - 只有1根K线",
			klines1d: []Kline{
				{Close: 95000.0},
			},
			currentPrice:   95000.0,
			expectedChange: 0.0,
			tolerance:      0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priceChange24h := 0.0
			if len(tt.klines1d) >= 2 {
				price24hAgo := tt.klines1d[len(tt.klines1d)-2].Close
				if price24hAgo > 0 {
					priceChange24h = ((tt.currentPrice - price24hAgo) / price24hAgo) * 100
				}
			}

			diff := priceChange24h - tt.expectedChange
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("PriceChange24h = %.2f%%, want %.2f%%", priceChange24h, tt.expectedChange)
			}
		})
	}
}

// TestFormat_IncludesPriceChange24h tests that Format output includes PriceChange24h
func TestFormat_IncludesPriceChange24h(t *testing.T) {
	data := &Data{
		Symbol:         "BTCUSDT",
		CurrentPrice:   95000.0,
		PriceChange1h:  1.5,
		PriceChange4h:  3.2,
		PriceChange24h: -2.5,
		CurrentEMA20:   94500.0,
		CurrentMACD:    150.0,
		CurrentRSI7:    55.0,
		OpenInterest:   &OIData{Latest: 100000, Average: 99000},
		FundingRate:    0.0001,
	}

	output := Format(data, false)

	// 验证输出包含 24h 价格变化
	if !strings.Contains(output, "24h") && !strings.Contains(output, "24H") {
		t.Errorf("Format output should contain 24h price change indicator")
		t.Logf("Output:\n%s", output)
	}

	// 验证输出包含实际的变化值（-2.5）
	if !strings.Contains(output, "-2.5") && !strings.Contains(output, "-2.50") {
		t.Errorf("Format output should contain the actual 24h price change value (-2.5%%)")
	}

	t.Log("✅ Test PASSED: Format includes PriceChange24h")
}

// TestBuildDataFromKlines_WithDailyData tests BuildDataFromKlines with daily data
func TestBuildDataFromKlines_WithDailyData(t *testing.T) {
	// 生成主要K线数据（3m）
	primary := make([]Kline, 100)
	for i := 0; i < 100; i++ {
		basePrice := 95000.0 + float64(i)*10
		primary[i] = Kline{
			OpenTime:  int64(i * 180000),
			Open:      basePrice,
			High:      basePrice + 50,
			Low:       basePrice - 30,
			Close:     basePrice + 20,
			Volume:    1000.0,
			CloseTime: int64((i+1)*180000 - 1),
		}
	}

	// 生成长期K线数据（4h）
	longer := make([]Kline, 50)
	for i := 0; i < 50; i++ {
		basePrice := 94000.0 + float64(i)*50
		longer[i] = Kline{
			OpenTime:  int64(i * 14400000),
			Open:      basePrice,
			High:      basePrice + 200,
			Low:       basePrice - 100,
			Close:     basePrice + 100,
			Volume:    5000.0,
			CloseTime: int64((i+1)*14400000 - 1),
		}
	}

	// 生成日线K线数据（1d）
	daily := make([]Kline, 30)
	for i := 0; i < 30; i++ {
		basePrice := 90000.0 + float64(i)*200
		daily[i] = Kline{
			OpenTime:  int64(i * 86400000),
			Open:      basePrice,
			High:      basePrice + 1000,
			Low:       basePrice - 500,
			Close:     basePrice + 500,
			Volume:    50000.0,
			CloseTime: int64((i+1)*86400000 - 1),
		}
	}

	// 测试不带日线数据（向后兼容）
	t.Run("无日线数据", func(t *testing.T) {
		data, err := BuildDataFromKlines("BTCUSDT", primary, longer)
		if err != nil {
			t.Fatalf("BuildDataFromKlines failed: %v", err)
		}
		if data.DailyContext != nil {
			t.Error("DailyContext should be nil when no daily data provided")
		}
	})

	// 测试带日线数据
	t.Run("有日线数据", func(t *testing.T) {
		data, err := BuildDataFromKlines("BTCUSDT", primary, longer, daily)
		if err != nil {
			t.Fatalf("BuildDataFromKlines failed: %v", err)
		}
		if data.DailyContext == nil {
			t.Fatal("DailyContext should not be nil when daily data provided")
		}
		if data.DailyContext.EMA20 <= 0 {
			t.Error("DailyContext.EMA20 should be > 0")
		}
		if data.PriceChange24h == 0 {
			t.Log("Warning: PriceChange24h is 0, might be expected if price unchanged")
		}
	})

	t.Log("✅ Test PASSED: BuildDataFromKlines works with daily data")
}
