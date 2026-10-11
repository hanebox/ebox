import { Switch as BaseSwitch } from '@base-ui/react/switch'

import { cn } from '@/lib/cn'

export interface SwitchProps {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
  id?: string
  className?: string
  'aria-describedby'?: string
}

export function Switch({
  checked,
  onCheckedChange,
  disabled = false,
  id,
  className,
  ...props
}: SwitchProps) {
  return (
    <BaseSwitch.Root
      id={id}
      checked={checked}
      onCheckedChange={onCheckedChange}
      disabled={disabled}
      {...props}
      className={cn(
        'relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border border-border bg-surface-raised transition-colors',
        'data-[checked]:border-accent data-[checked]:bg-accent',
        'disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
    >
      <BaseSwitch.Thumb className="block size-3.5 translate-x-[3px] rounded-full bg-muted transition-transform data-[checked]:translate-x-[17px] data-[checked]:bg-white" />
    </BaseSwitch.Root>
  )
}
