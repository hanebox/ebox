import { describe, expect, it } from 'vitest'

import { formatBytes, formatPercent, formatRate, formatUptime } from '@/lib/format'

describe('formatBytes', () => {
  it('formats zero and invalid input', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(-5)).toBe('0 B')
    expect(formatBytes(Number.NaN)).toBe('0 B')
  })

  it('formats binary units', () => {
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1024 ** 2)).toBe('1.0 MB')
    expect(formatBytes(1.5 * 1024 ** 3)).toBe('1.5 GB')
  })
})

describe('formatPercent', () => {
  it('formats with one decimal by default', () => {
    expect(formatPercent(42.123)).toBe('42.1%')
    expect(formatPercent(Number.NaN)).toBe('0%')
  })
})

describe('formatRate', () => {
  it('appends per-second suffix', () => {
    expect(formatRate(2048)).toBe('2.0 KB/s')
  })
})

describe('formatUptime', () => {
  it('formats days, hours and minutes', () => {
    expect(formatUptime(90)).toBe('1m')
    expect(formatUptime(3700)).toBe('1h 1m')
    expect(formatUptime(90061)).toBe('1d 1h')
    expect(formatUptime(0)).toBe('—')
  })
})
