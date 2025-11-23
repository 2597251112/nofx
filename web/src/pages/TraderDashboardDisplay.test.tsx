import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import TraderDashboard from './TraderDashboard'

// Mocks
vi.mock('react-router-dom', () => ({
  useNavigate: () => vi.fn(),
  useSearchParams: () => [new URLSearchParams('trader=test-trader-1'), vi.fn()],
}))

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 'user1' }, token: 'fake-token' }),
}))

vi.mock('../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))

vi.mock('../lib/api', () => ({
  api: {
    getTraders: vi.fn(),
    getStatus: vi.fn(),
    getAccount: vi.fn(),
    getPositions: vi.fn(),
    getLatestDecisions: vi.fn(),
    getStatistics: vi.fn(),
  }
}))

// Mock useSWR to return data based on key
vi.mock('swr', () => {
  return {
    default: (key: any) => {
      if (key === 'traders') {
        return {
          data: [
            {
              trader_id: 'test-trader-1',
              trader_name: 'Test Trader',
              ai_model: 'gpt-4',
              // This is the field we are testing. 
              // Initially, TS might complain if we pass it before updating types, 
              // but in JS runtime it's just an object. 
              // We will rely on the test failing because the UI doesn't render it yet.
              scan_interval_minutes: 5 
            }
          ],
          error: null,
        }
      }
      if (typeof key === 'string' && key.startsWith('status-')) {
        return {
            data: {
                call_count: 10,
                runtime_minutes: 100
            },
            error: null
        }
      }
      return { data: null, error: null }
    }
  }
})

// Mock child components to avoid rendering complexity
vi.mock('../components/EquityChart', () => ({ EquityChart: () => <div>Chart</div> }))
vi.mock('../components/AILearning', () => ({ default: () => <div>AI Learning</div> }))

describe('TraderDashboard Display', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should display scan interval in the header', async () => {
    render(<TraderDashboard />)

    // Check if trader name is rendered (sanity check)
    expect(screen.getAllByText('Test Trader').length).toBeGreaterThan(0)

    // Check for the new Scan Interval display
    // This expectation should FAIL initially
    await waitFor(() => {
        const intervalText = screen.queryByText(/Interval: 5 min/i)
        if (!intervalText) {
            throw new Error('Interval text not found')
        }
        expect(intervalText).toBeInTheDocument()
    })
  })
})
