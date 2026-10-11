import { Dialog as BaseDialog } from '@base-ui/react/dialog'
import { IconX } from '@tabler/icons-react'
import type { ReactNode } from 'react'

import { cn } from '@/lib/cn'

export const Dialog = BaseDialog

export interface ModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  children?: ReactNode
  footer?: ReactNode
  size?: 'sm' | 'md' | 'lg'
}

const sizeClasses = {
  sm: 'max-w-sm',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
}

export function Modal({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
  size = 'md',
}: ModalProps) {
  return (
    <BaseDialog.Root open={open} onOpenChange={onOpenChange}>
      <BaseDialog.Portal>
        <BaseDialog.Backdrop
          className={cn(
            'fixed inset-0 z-40 bg-black/30 backdrop-blur-[2px] transition-opacity duration-150 ease-out-quint dark:bg-black/65',
            'data-ending-style:opacity-0 data-ending-style:duration-100 data-starting-style:opacity-0',
          )}
        />
        <BaseDialog.Popup
          className={cn(
            'fixed top-1/2 left-1/2 z-50 w-[calc(100vw-2rem)] -translate-x-1/2 -translate-y-1/2 overscroll-contain',
            'rounded-panel border border-border bg-surface-overlay shadow-popover outline-none',
            'transition-[opacity,transform] duration-150 ease-out-quint data-ending-style:scale-[0.98] data-ending-style:opacity-0 data-ending-style:duration-100 data-starting-style:scale-[0.98] data-starting-style:opacity-0',
            sizeClasses[size],
          )}
        >
          <div className="flex items-start justify-between gap-4 border-b border-border p-4">
            <div className="flex flex-col gap-0.5">
              <BaseDialog.Title className="text-base font-semibold tracking-[-0.01em] text-balance text-foreground">
                {title}
              </BaseDialog.Title>
              {description ? (
                <BaseDialog.Description className="text-xs leading-relaxed text-pretty text-muted">
                  {description}
                </BaseDialog.Description>
              ) : null}
            </div>
            <BaseDialog.Close
              aria-label="Close dialog"
              className="hit-area press rounded-tile p-1 text-subtle transition-[background-color,color,transform] duration-150 hover:bg-surface-raised hover:text-foreground"
            >
              <IconX aria-hidden className="size-4" />
            </BaseDialog.Close>
          </div>
          {children ? <div className="p-4">{children}</div> : null}
          {footer ? (
            <div className="flex items-center justify-end gap-2 border-t border-border p-4">
              {footer}
            </div>
          ) : null}
        </BaseDialog.Popup>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  )
}
