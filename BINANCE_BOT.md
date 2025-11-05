# Binance EMA Quant Bot

This document explains how to run the lightweight EMA cross quantitative bot that lives in `cmd/binance-bot`.

## Overview

- Exchange: Binance USDⓈ-M futures
- Strategy: Dual EMA crossover (default 9 / 26)
- Position logic: Switches between long / short / flat, uses leverage and fixed USDT notional
- Risk controls: Optional stop-loss & take-profit, flat threshold to exit congested markets
- Data source: Binance REST klines (`github.com/adshao/go-binance/v2/futures`)
- Execution: Reuses project `trader.FuturesTrader` for precision, leverage & order management

⚠️ **Important**: Futures trading is high risk. Run on Binance Testnet or with very small size until you fully trust the bot.

## Build the Bot

```bash
cd /path/to/nofx
go build -o binance-bot ./cmd/binance-bot
```

The generated binary can be copied to your server or run locally.

## Configure Credentials

Set your Binance API credentials (Futures-enabled) either via environment variables or a JSON config file.

### Option 1 — Environment Variables

```bash
export BINANCE_API_KEY="your_api_key"
export BINANCE_SECRET_KEY="your_secret_key"
export BOT_SYMBOLS="BTCUSDT,ETHUSDT"
```

You can then rely on CLI flags for the remaining parameters.

### Option 2 — JSON Config File

Copy the example file and fill it in:

```bash
cp config/binance_bot.example.json binance_bot.json
vim binance_bot.json  # or any editor
```

`poll_interval` accepts any value supported by `time.ParseDuration`, e.g. `"45s"`, `"2m"`, `"1h"`.

## Run the Bot

```bash
./binance-bot -config binance_bot.json
```

Or, using environment variables + CLI overrides:

```bash
./binance-bot \
  -symbols BTCUSDT,ETHUSDT \
  -position 80 \
  -leverage 3 \
  -fast-ema 12 \
  -slow-ema 26 \
  -stop-loss 1.5 \
  -take-profit 3 \
  -poll 1m
```

Press `Ctrl+C` to stop. The bot listens for `SIGINT`/`SIGTERM` and exits gracefully.

## Key Flags

| Flag | Description | Default |
| --- | --- | --- |
| `-api-key` | Binance API key (overrides env and config) | – |
| `-secret-key` | Binance API secret | – |
| `-config` | Path to JSON config file | – |
| `-symbols` | Comma separated trading pairs (auto-appends `USDT`) | `BTCUSDT,ETHUSDT` |
| `-position` | Position size per trade in USDT | `50` |
| `-leverage` | Futures leverage for all trades | `3` |
| `-fast-ema` | Fast EMA period | `9` |
| `-slow-ema` | Slow EMA period (must be > fast) | `26` |
| `-signal-buffer` | Required % (decimal) gap between EMAs to trigger new direction | `0.001` |
| `-flat-threshold` | If `|fast-slow|/slow` falls below this, existing positions are closed | `0.0005` |
| `-stop-loss` | Stop-loss percent (0 to disable) | `1.5` |
| `-take-profit` | Take-profit percent (0 to disable) | `3.0` |
| `-kline-interval` | Kline interval (e.g. `3m`, `5m`, `15m`) | `5m` |
| `-kline-limit` | Number of klines pulled per cycle | `300` |
| `-poll` | Loop interval | `1m` |

## Strategy Behaviour

1. Fetch klines for every configured symbol.
2. Compute current and previous EMA values.
3. When fast EMA crosses above slow EMA (with buffer), open/maintain a long position.
4. When fast EMA crosses below slow EMA (with buffer), open/maintain a short position.
5. If already positioned but EMAs converge within the flat threshold, exit to flat.
6. Optional stop-loss / take-profit orders are updated after each entry.

## Tips

- Test on Binance Futures **testnet** (`https://testnet.binancefuture.com`) by changing API base URL via environment variables supported by the go-binance SDK (`BINANCE_FUTURES_API_BASE_URL`).
- Use small `position_size_usdt` until you observe the bot for multiple days.
- Monitor logs closely. All major decisions are logged with emoji markers for quick scanning.
- Run behind a process manager (systemd, supervisord, pm2) for production use.

Stay safe and happy trading! ✨
