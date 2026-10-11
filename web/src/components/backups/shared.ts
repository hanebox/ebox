export const schedulePresets = [
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Daily at 03:00', value: '0 3 * * *' },
  { label: 'Weekly, Sunday 04:00', value: '0 4 * * 0' },
  { label: 'Monthly, 1st at 05:00', value: '0 5 1 * *' },
  { label: 'Every hour', value: '@hourly' },
]

export function formatBytes(bytes: number): string {
  if (bytes <= 0) {
    return '—'
  }

  const units = ['B', 'kB', 'MB', 'GB', 'TB']
  let value = bytes
  let index = 0

  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index++
  }

  return `${value.toFixed(value < 10 && index > 0 ? 1 : 0)} ${units[index]}`
}

export function runStatusVariant(status: string) {
  switch (status) {
    case 'success':
      return 'success' as const
    case 'failed':
      return 'danger' as const
    case 'running':
      return 'info' as const
    default:
      return 'outline' as const
  }
}
