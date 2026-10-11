import { Tooltip as BaseTooltip } from '@base-ui/react/tooltip'
import type { ReactElement, ReactNode } from 'react'

export interface TooltipProps {
  content: ReactNode
  children: ReactElement
  side?: 'top' | 'bottom' | 'left' | 'right'
  sideOffset?: number
}

export function Tooltip({ content, children, side = 'top', sideOffset = 6 }: TooltipProps) {
  return (
    <BaseTooltip.Root>
      <BaseTooltip.Trigger render={children} />
      <BaseTooltip.Portal>
        <BaseTooltip.Positioner side={side} sideOffset={sideOffset} className="z-50">
          <BaseTooltip.Popup className="max-w-64 origin-[var(--transform-origin)] rounded-tile bg-foreground px-2 py-1 text-2xs font-medium text-background transition-[opacity,transform] duration-125 ease-out-quint data-ending-style:opacity-0 data-ending-style:duration-75 data-starting-style:scale-[0.97] data-starting-style:opacity-0">
            {content}
          </BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  )
}
