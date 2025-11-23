import React from 'react'

/**
 * Helper function to generate exchange URLs
 */
export function getExchangeUrl(
  exchangeId: string | undefined,
  symbol: string
): string | null {
  if (!exchangeId || !symbol) return null

  const upperSymbol = symbol.toUpperCase()

  switch (exchangeId.toLowerCase()) {
    case 'binance':
      // Binance Futures URL (e.g., BTCUSDT)
      return `https://www.binance.com/en/futures/${upperSymbol}`

    case 'hyperliquid':
      // Hyperliquid URL (e.g., BTC) - Strip 'USDT' suffix
      // Assuming symbols are stored as 'BTCUSDT' internally
      // eslint-disable-next-line no-case-declarations
      const hlSymbol = upperSymbol.endsWith('USDT')
        ? upperSymbol.replace('USDT', '')
        : upperSymbol
      return `https://app.hyperliquid.xyz/trade/${hlSymbol}`

    case 'aster':
      // Aster DEX URL (e.g. BTCUSDT)
      return `https://www.asterdex.com/en/futures/${upperSymbol}`

    default:
      return null
  }
}

interface ExchangeLinkProps
  extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  exchangeId?: string
  symbol: string
  children?: React.ReactNode
}

/**
 * Component to link to exchange trading page
 */
export const ExchangeLink: React.FC<ExchangeLinkProps> = ({
  exchangeId,
  symbol,
  children,
  className = '',
  ...props
}) => {
  const url = getExchangeUrl(exchangeId, symbol)
  const content = children || symbol

  if (!url) {
    return <span className={className}>{content}</span>
  }

  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className={`hover:text-blue-400 hover:underline transition-colors ${className}`}
      title={`Trade ${symbol} on ${exchangeId}`}
      onClick={(e) => e.stopPropagation()} // Prevent triggering row clicks in tables
      {...props}
    >
      {content}
    </a>
  )
}
