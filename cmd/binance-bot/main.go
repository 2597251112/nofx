package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"nofx/trader"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

// BotConfig 控制简单量化交易机器人的所有参数
type BotConfig struct {
	APIKey        string        `json:"api_key"`
	SecretKey     string        `json:"secret_key"`
	Symbols       []string      `json:"symbols"`
	PositionSize  float64       `json:"position_size_usdt"`
	Leverage      int           `json:"leverage"`
	FastEMAPeriod int           `json:"fast_ema_period"`
	SlowEMAPeriod int           `json:"slow_ema_period"`
	SignalBuffer  float64       `json:"signal_buffer"`
	FlatThreshold float64       `json:"flat_threshold"`
	StopLossPct   float64       `json:"stop_loss_pct"`
	TakeProfitPct float64       `json:"take_profit_pct"`
	KlineInterval string        `json:"kline_interval"`
	KlineLimit    int           `json:"kline_limit"`
	PollInterval  time.Duration `json:"poll_interval"`
}

func defaultBotConfig() BotConfig {
	return BotConfig{
		Symbols:       []string{"BTCUSDT", "ETHUSDT"},
		PositionSize:  50,
		Leverage:      3,
		FastEMAPeriod: 9,
		SlowEMAPeriod: 26,
		SignalBuffer:  0.001,
		FlatThreshold: 0.0005,
		StopLossPct:   1.5,
		TakeProfitPct: 3.0,
		KlineInterval: "5m",
		KlineLimit:    300,
		PollInterval:  time.Minute,
	}
}

func (cfg *BotConfig) normalize() {
	upperSymbols := make([]string, 0, len(cfg.Symbols))
	for _, sym := range cfg.Symbols {
		sym = strings.ToUpper(strings.TrimSpace(sym))
		if sym == "" {
			continue
		}
		if !strings.HasSuffix(sym, "USDT") {
			sym += "USDT"
		}
		upperSymbols = append(upperSymbols, sym)
	}
	cfg.Symbols = upperSymbols
}

func (cfg *BotConfig) validate() error {
	if cfg.APIKey == "" {
		return errors.New("missing Binance API key")
	}
	if cfg.SecretKey == "" {
		return errors.New("missing Binance Secret key")
	}
	if len(cfg.Symbols) == 0 {
		return errors.New("at least one symbol is required")
	}
	if cfg.PositionSize <= 0 {
		return fmt.Errorf("position_size_usdt must be > 0, got %.2f", cfg.PositionSize)
	}
	if cfg.Leverage <= 0 {
		return fmt.Errorf("leverage must be > 0, got %d", cfg.Leverage)
	}
	if cfg.FastEMAPeriod <= 0 {
		return fmt.Errorf("fast_ema_period must be > 0, got %d", cfg.FastEMAPeriod)
	}
	if cfg.SlowEMAPeriod <= 0 {
		return fmt.Errorf("slow_ema_period must be > 0, got %d", cfg.SlowEMAPeriod)
	}
	if cfg.FastEMAPeriod >= cfg.SlowEMAPeriod {
		return errors.New("fast_ema_period must be smaller than slow_ema_period")
	}
	if cfg.KlineInterval == "" {
		return errors.New("kline_interval cannot be empty")
	}
	if cfg.KlineLimit <= cfg.SlowEMAPeriod+1 {
		return fmt.Errorf("kline_limit must be greater than slow_ema_period+1, got %d", cfg.KlineLimit)
	}
	if cfg.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be > 0, got %s", cfg.PollInterval)
	}
	return nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	cfg.normalize()
	if err := cfg.validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	log.Printf("🚀 Binance 简单量化交易机器人启动，交易对: %s", strings.Join(cfg.Symbols, ", "))
	log.Printf("📈 策略: EMA(%d/%d) 交叉, 仓位 %.2f USDT, 杠杆 %dx",
		cfg.FastEMAPeriod, cfg.SlowEMAPeriod, cfg.PositionSize, cfg.Leverage)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	traderClient := trader.NewFuturesTrader(cfg.APIKey, cfg.SecretKey)
	marketClient := futures.NewClient(cfg.APIKey, cfg.SecretKey)

	// 立即执行一次，随后进入周期循环
	if err := executeCycle(ctx, traderClient, marketClient, cfg); err != nil {
		log.Printf("❌ 执行周期失败: %v", err)
	}

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 收到退出信号，机器人停止")
			return
		case <-ticker.C:
			if err := executeCycle(ctx, traderClient, marketClient, cfg); err != nil {
				log.Printf("❌ 执行周期失败: %v", err)
			}
		}
	}
}

func loadConfig() (BotConfig, error) {
	cfg := defaultBotConfig()

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	configPath := fs.String("config", "", "Path to JSON config file")
	apiKeyFlag := fs.String("api-key", "", "Binance API key (overrides BINANCE_API_KEY)")
	secretFlag := fs.String("secret-key", "", "Binance Secret key (overrides BINANCE_SECRET_KEY)")
	symbolsFlag := fs.String("symbols", "", "Comma separated symbols, e.g. BTCUSDT,ETHUSDT")
	positionFlag := fs.Float64("position", cfg.PositionSize, "Position size per trade in USDT")
	leverageFlag := fs.Int("leverage", cfg.Leverage, "Per-trade leverage")
	fastFlag := fs.Int("fast-ema", cfg.FastEMAPeriod, "Fast EMA period")
	slowFlag := fs.Int("slow-ema", cfg.SlowEMAPeriod, "Slow EMA period")
	pollFlag := fs.Duration("poll", cfg.PollInterval, "Polling interval, e.g. 30s, 1m")
	stopLossFlag := fs.Float64("stop-loss", cfg.StopLossPct, "Stop loss percent (set 0 to disable)")
	takeProfitFlag := fs.Float64("take-profit", cfg.TakeProfitPct, "Take profit percent (set 0 to disable)")
	bufferFlag := fs.Float64("signal-buffer", cfg.SignalBuffer, "Signal buffer ratio (0.001 = 0.1%)")
	flatFlag := fs.Float64("flat-threshold", cfg.FlatThreshold, "Close position when |fast-slow|/slow below this value")
	intervalFlag := fs.String("kline-interval", cfg.KlineInterval, "Kline interval, e.g. 3m,5m,15m")
	limitFlag := fs.Int("kline-limit", cfg.KlineLimit, "Number of klines to load each cycle")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return cfg, err
	}

	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		seen[f.Name] = true
	})

	if *configPath != "" {
		fileCfg, err := loadConfigFromFile(*configPath)
		if err != nil {
			return cfg, err
		}
		cfg = fileCfg
	}

	if seen["api-key"] {
		cfg.APIKey = *apiKeyFlag
	} else if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("BINANCE_API_KEY")
	}

	if seen["secret-key"] {
		cfg.SecretKey = *secretFlag
	} else if cfg.SecretKey == "" {
		cfg.SecretKey = os.Getenv("BINANCE_SECRET_KEY")
	}

	if seen["symbols"] {
		cfg.Symbols = splitSymbols(*symbolsFlag)
	} else if len(cfg.Symbols) == 0 {
		cfg.Symbols = splitSymbols(os.Getenv("BOT_SYMBOLS"))
	}

	if seen["position"] {
		cfg.PositionSize = *positionFlag
	}
	if seen["leverage"] {
		cfg.Leverage = *leverageFlag
	}
	if seen["fast-ema"] {
		cfg.FastEMAPeriod = *fastFlag
	}
	if seen["slow-ema"] {
		cfg.SlowEMAPeriod = *slowFlag
	}
	if seen["poll"] {
		cfg.PollInterval = *pollFlag
	}
	if seen["stop-loss"] {
		cfg.StopLossPct = *stopLossFlag
	}
	if seen["take-profit"] {
		cfg.TakeProfitPct = *takeProfitFlag
	}
	if seen["signal-buffer"] {
		cfg.SignalBuffer = *bufferFlag
	}
	if seen["flat-threshold"] {
		cfg.FlatThreshold = *flatFlag
	}
	if seen["kline-interval"] {
		cfg.KlineInterval = *intervalFlag
	}
	if seen["kline-limit"] {
		cfg.KlineLimit = *limitFlag
	}

	// 如果仍然没有symbol，尝试默认
	if len(cfg.Symbols) == 0 {
		cfg.Symbols = defaultBotConfig().Symbols
	}

	return cfg, nil
}

func loadConfigFromFile(path string) (BotConfig, error) {
	cfg := defaultBotConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config file: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config file: %w", err)
	}
	return cfg, nil
}

func splitSymbols(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		sym := strings.TrimSpace(p)
		if sym != "" {
			result = append(result, sym)
		}
	}
	return result
}

func executeCycle(ctx context.Context, traderClient *trader.FuturesTrader, marketClient *futures.Client, cfg BotConfig) error {
	for _, symbol := range cfg.Symbols {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		prices, err := fetchClosePrices(ctx, marketClient, symbol, cfg.KlineInterval, cfg.KlineLimit)
		if err != nil {
			log.Printf("⚠️ 获取K线失败 (%s): %v", symbol, err)
			continue
		}

		if len(prices) < cfg.SlowEMAPeriod+1 {
			log.Printf("⚠️ K线数量不足以计算EMA (%s)", symbol)
			continue
		}

		fast := ema(prices, cfg.FastEMAPeriod)
		slow := ema(prices, cfg.SlowEMAPeriod)
		fastPrev := ema(prices[:len(prices)-1], cfg.FastEMAPeriod)
		slowPrev := ema(prices[:len(prices)-1], cfg.SlowEMAPeriod)

		if fast == 0 || slow == 0 || fastPrev == 0 || slowPrev == 0 {
			log.Printf("⚠️ EMA计算结果为0，跳过 %s", symbol)
			continue
		}

		currentSide, _ := getCurrentPosition(traderClient, symbol)

		log.Printf("📊 %s EMA%d=%.4f EMA%d=%.4f (prev %.4f/%.4f) 当前仓位=%s",
			symbol, cfg.FastEMAPeriod, fast, cfg.SlowEMAPeriod, slow, fastPrev, slowPrev, currentSide)

		upperSignal := slow * (1 + cfg.SignalBuffer)
		lowerSignal := slow * (1 - cfg.SignalBuffer)
		price, err := traderClient.GetMarketPrice(symbol)
		if err != nil {
			log.Printf("⚠️ 获取市场价格失败 (%s): %v", symbol, err)
			continue
		}

		switch {
		case fastPrev <= slowPrev && fast > upperSignal:
			if currentSide == "long" {
				log.Printf("🔁 %s 已持有多仓，保持", symbol)
				continue
			}
			if err := switchToLong(traderClient, symbol, price, cfg, currentSide); err != nil {
				log.Printf("❌ 开多失败 (%s): %v", symbol, err)
			}
		case fastPrev >= slowPrev && fast < lowerSignal:
			if currentSide == "short" {
				log.Printf("🔁 %s 已持有空仓，保持", symbol)
				continue
			}
			if err := switchToShort(traderClient, symbol, price, cfg, currentSide); err != nil {
				log.Printf("❌ 开空失败 (%s): %v", symbol, err)
			}
		default:
			// 若当前持仓且指标几乎重合，则平仓
			if currentSide != "flat" {
				diffRatio := math.Abs(fast-slow) / slow
				if diffRatio < cfg.FlatThreshold {
					log.Printf("⚖️ %s EMA差值 %.5f < 阈值 %.5f，尝试平仓", symbol, diffRatio, cfg.FlatThreshold)
					if err := closePosition(traderClient, symbol, currentSide); err != nil {
						log.Printf("❌ 平仓失败 (%s): %v", symbol, err)
					}
				}
			}
		}
	}

	return nil
}

func fetchClosePrices(ctx context.Context, client *futures.Client, symbol, interval string, limit int) ([]float64, error) {
	svc := client.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit)
	klines, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}

	prices := make([]float64, 0, len(klines))
	for _, k := range klines {
		closeVal, err := strconv.ParseFloat(k.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("parse close price: %w", err)
		}
		prices = append(prices, closeVal)
	}
	return prices, nil
}

func ema(prices []float64, period int) float64 {
	if len(prices) < period || period <= 0 {
		return 0
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema := sum / float64(period)
	multiplier := 2.0 / float64(period+1)

	for i := period; i < len(prices); i++ {
		ema = (prices[i]-ema)*multiplier + ema
	}

	return ema
}

func getCurrentPosition(traderClient *trader.FuturesTrader, symbol string) (string, float64) {
	positions, err := traderClient.GetPositions()
	if err != nil {
		log.Printf("⚠️ 获取持仓失败 (%s): %v", symbol, err)
		return "flat", 0
	}

	for _, pos := range positions {
		sym, ok := pos["symbol"].(string)
		if !ok || sym != symbol {
			continue
		}
		side, _ := pos["side"].(string)
		qty, _ := pos["positionAmt"].(float64)
		if qty < 0 {
			qty = -qty
		}
		return side, qty
	}

	return "flat", 0
}

func switchToLong(traderClient *trader.FuturesTrader, symbol string, price float64, cfg BotConfig, currentSide string) error {
	if currentSide == "short" {
		if err := closePosition(traderClient, symbol, "short"); err != nil {
			return err
		}
	}

	quantity := cfg.PositionSize / price
	if quantity <= 0 {
		return fmt.Errorf("calculated quantity is not positive for %s", symbol)
	}

	log.Printf("📈 %s 开多 %.4f，价格≈%.4f", symbol, quantity, price)
	if _, err := traderClient.OpenLong(symbol, quantity, cfg.Leverage); err != nil {
		return err
	}

	setStops(traderClient, symbol, "LONG", quantity, price, cfg)
	return nil
}

func switchToShort(traderClient *trader.FuturesTrader, symbol string, price float64, cfg BotConfig, currentSide string) error {
	if currentSide == "long" {
		if err := closePosition(traderClient, symbol, "long"); err != nil {
			return err
		}
	}

	quantity := cfg.PositionSize / price
	if quantity <= 0 {
		return fmt.Errorf("calculated quantity is not positive for %s", symbol)
	}

	log.Printf("📉 %s 开空 %.4f，价格≈%.4f", symbol, quantity, price)
	if _, err := traderClient.OpenShort(symbol, quantity, cfg.Leverage); err != nil {
		return err
	}

	setStops(traderClient, symbol, "SHORT", quantity, price, cfg)
	return nil
}

func closePosition(traderClient *trader.FuturesTrader, symbol, side string) error {
	switch side {
	case "long":
		log.Printf("🔒 平多仓 %s", symbol)
		if _, err := traderClient.CloseLong(symbol, 0); err != nil {
			return err
		}
	case "short":
		log.Printf("🔒 平空仓 %s", symbol)
		if _, err := traderClient.CloseShort(symbol, 0); err != nil {
			return err
		}
	}
	return nil
}

func setStops(traderClient *trader.FuturesTrader, symbol, positionSide string, quantity, entryPrice float64, cfg BotConfig) {
	if cfg.StopLossPct > 0 {
		var stopPrice float64
		if positionSide == "LONG" {
			stopPrice = entryPrice * (1 - cfg.StopLossPct/100)
		} else {
			stopPrice = entryPrice * (1 + cfg.StopLossPct/100)
		}
		if err := traderClient.SetStopLoss(symbol, positionSide, quantity, stopPrice); err != nil {
			log.Printf("⚠️ 设置止损失败 (%s): %v", symbol, err)
		}
	}

	if cfg.TakeProfitPct > 0 {
		var takeProfit float64
		if positionSide == "LONG" {
			takeProfit = entryPrice * (1 + cfg.TakeProfitPct/100)
		} else {
			takeProfit = entryPrice * (1 - cfg.TakeProfitPct/100)
		}
		if err := traderClient.SetTakeProfit(symbol, positionSide, quantity, takeProfit); err != nil {
			log.Printf("⚠️ 设置止盈失败 (%s): %v", symbol, err)
		}
	}
}
