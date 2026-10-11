import { AlertDialog } from '@base-ui/react/alert-dialog'
import type { ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'

export interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  loading?: boolean
  onConfirm: () => void
  children?: ReactNode
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  danger = false,
  loading = false,
  onConfirm,
  children,
}: ConfirmDialogProps) {
  return (
    <AlertDialog.Root open={open} onOpenChange={onOpenChange}>
      <AlertDialog.Portal>
        <AlertDialog.Backdrop
          className={cn(
            'fixed inset-0 z-40 bg-black/30 backdrop-blur-[2px] transition-opacity duration-150 ease-out-quint dark:bg-black/65',
            'data-ending-style:opacity-0 data-ending-style:duration-100 data-starting-style:opacity-0',
          )}
        />
        <AlertDialog.Popup
          className={cn(
            'fixed top-1/2 left-1/2 z-50 w-[calc(100vw-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 overscroll-contain',
            'rounded-panel border border-border bg-surface-overlay p-4 shadow-popover outline-none',
            'transition-[opacity,transform] duration-150 ease-out-quint data-ending-style:scale-[0.98] data-ending-style:opacity-0 data-ending-style:duration-100 data-starting-style:scale-[0.98] data-starting-style:opacity-0',
          )}
        >
          <AlertDialog.Title className="text-base font-semibold tracking-[-0.01em] text-balance text-foreground">
            {title}
          </AlertDialog.Title>
          {description ? (
            <AlertDialog.Description className="mt-1 text-xs leading-relaxed text-pretty text-muted">
              {description}
            </AlertDialog.Description>
          ) : null}
          {children ? <div className="mt-3">{children}</div> : null}
          <div className="mt-4 flex items-center justify-end gap-2">
            <AlertDialog.Close
              render={
                <Button variant="ghost" size="sm" disabled={loading}>
                  {cancelLabel}
                </Button>
              }
            />
            <Button
              variant={danger ? 'danger' : 'primary'}
              size="sm"
              loading={loading}
              onClick={onConfirm}
            >
              {confirmLabel}
            </Button>
          </div>
        </AlertDialog.Popup>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  )
}
