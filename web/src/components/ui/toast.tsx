import { Toast } from '@base-ui/react/toast'
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from 'lucide-react'

import { cn } from '@/lib/cn'

export const toastManager = Toast.createToastManager()

type ToastType = 'success' | 'error' | 'warning' | 'info'

function add(type: ToastType, title: string, description?: string) {
  toastManager.add({ title, description, type, timeout: type === 'error' ? 8000 : 4500 })
}

export const toast = {
  success: (title: string, description?: string) => add('success', title, description),
  error: (title: string, description?: string) => add('error', title, description),
  warning: (title: string, description?: string) => add('warning', title, description),
  info: (title: string, description?: string) => add('info', title, description),
}

const typeStyles: Record<ToastType, { icon: typeof Info; className: string }> = {
  success: { icon: CheckCircle2, className: 'text-success' },
  error: { icon: XCircle, className: 'text-danger' },
  warning: { icon: AlertTriangle, className: 'text-warning' },
  info: { icon: Info, className: 'text-info' },
}

export function Toaster() {
  const { toasts } = Toast.useToastManager()

  return (
    <Toast.Portal>
      <Toast.Viewport className="fixed right-4 bottom-4 z-50 flex w-[min(24rem,calc(100vw-2rem))] flex-col gap-2 outline-none">
        {toasts.map((item) => {
          const type = (item.type ?? 'info') as ToastType
          const style = typeStyles[type] ?? typeStyles.info
          const Icon = style.icon

          return (
            <Toast.Root
              key={item.id}
              toast={item}
              className={cn(
                'rounded-panel border border-border bg-surface-overlay p-3 shadow-popover',
                'transition-all duration-200 data-ending-style:translate-x-2 data-ending-style:opacity-0',
                'data-starting-style:translate-x-2 data-starting-style:opacity-0',
              )}
            >
              <Toast.Content className="flex items-start gap-3">
                <Icon aria-hidden className={cn('mt-0.5 size-4 shrink-0', style.className)} />
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <Toast.Title className="text-sm font-medium text-foreground" />
                  <Toast.Description className="text-xs text-muted" />
                </div>
                <Toast.Close
                  className="rounded-md p-0.5 text-subtle transition-colors hover:text-foreground"
                  aria-label="Dismiss notification"
                >
                  <X aria-hidden className="size-3.5" />
                </Toast.Close>
              </Toast.Content>
            </Toast.Root>
          )
        })}
      </Toast.Viewport>
    </Toast.Portal>
  )
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  return (
    <Toast.Provider toastManager={toastManager} limit={4}>
      {children}
      <Toaster />
    </Toast.Provider>
  )
}
