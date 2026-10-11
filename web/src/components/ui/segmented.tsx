import { cn } from '@/lib/cn'

export interface SegmentedOption<T extends string> {
  value: T
  label: string
}

export interface SegmentedProps<T extends string> {
  value: T
  options: SegmentedOption<T>[]
  onValueChange: (value: T) => void
  className?: string
  'aria-label'?: string
}

export function Segmented<T extends string>({
  value,
  options,
  onValueChange,
  className,
  ...props
}: SegmentedProps<T>) {
  return (
    <div
      role="group"
      className={cn(
        'inline-flex items-center gap-0.5 rounded-full bg-surface-raised p-0.5 shadow-border',
        className,
      )}
      {...props}
    >
      {options.map((option) => {
        const selected = option.value === value

        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={selected}
            onClick={() => onValueChange(option.value)}
            className={cn(
              'rounded-full px-2.5 py-1 text-xs font-medium transition-colors duration-150 ease-out-quint',
              selected
                ? 'bg-background text-foreground'
                : 'text-subtle hover:text-foreground',
            )}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}
