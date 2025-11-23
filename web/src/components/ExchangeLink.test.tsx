import { describe, it, expect, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { ExchangeLink, getExchangeUrl } from './ExchangeLink'
import React from 'react'

// Test the helper function logic
describe('getExchangeUrl', () => {
  it('should return correct URL for Binance', () => {
    const url = getExchangeUrl('binance', 'BTCUSDT')
    expect(url).toBe('https://www.binance.com/en/futures/BTCUSDT')
  })

  it('should return correct URL for Hyperliquid with raw symbol', () => {
    const url = getExchangeUrl('hyperliquid', 'BTC')
    expect(url).toBe('https://app.hyperliquid.xyz/trade/BTC')
  })

  it('should strip USDT suffix for Hyperliquid', () => {
    const url = getExchangeUrl('hyperliquid', 'BTCUSDT')
    expect(url).toBe('https://app.hyperliquid.xyz/trade/BTC')
  })

  it('should return correct URL for Aster', () => {
    const url = getExchangeUrl('aster', 'BTCUSDT')
    expect(url).toBe('https://www.asterdex.com/en/futures/BTCUSDT')
  })

  it('should handle case insensitivity for exchange ID', () => {
    expect(getExchangeUrl('BINANCE', 'ETHUSDT')).toBe(
      'https://www.binance.com/en/futures/ETHUSDT'
    )
  })

  it('should return null for unknown exchange', () => {
    expect(getExchangeUrl('unknown', 'BTCUSDT')).toBeNull()
  })
})

// Test the Component rendering
describe('ExchangeLink Component', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders an anchor tag with correct href for valid exchange', () => {
    render(<ExchangeLink exchangeId="binance" symbol="BTCUSDT" />)
    const link = screen.getByRole('link', { name: 'BTCUSDT' })
    expect(link).toBeDefined()
    expect(link.getAttribute('href')).toBe(
      'https://www.binance.com/en/futures/BTCUSDT'
    )
    expect(link.getAttribute('target')).toBe('_blank')
  })

  it('renders text only for unknown exchange', () => {
    render(<ExchangeLink exchangeId="unknown" symbol="BTCUSDT" />)
    // Should not find a link
    const link = screen.queryByRole('link')
    expect(link).toBeNull()
    // Should find the text
    expect(screen.getByText('BTCUSDT')).toBeDefined()
  })

  it('renders custom children', () => {
    render(
      <ExchangeLink exchangeId="hyperliquid" symbol="BTC">
        <span>Custom Text</span>
      </ExchangeLink>
    )
    const link = screen.getByRole('link', { name: 'Custom Text' })
    expect(link.getAttribute('href')).toBe(
      'https://app.hyperliquid.xyz/trade/BTC'
    )
  })

  it('applies custom class names', () => {
    render(
      <ExchangeLink
        exchangeId="binance"
        symbol="BTCUSDT"
        className="custom-class"
      />
    )
    const link = screen.getByRole('link')
    expect(link.className).toContain('custom-class')
  })
})
