import { useId, type ReactNode } from 'react'

import { cn } from '@/lib/cn'

export interface FieldProps {
  label: string
  description?: string
  error?: string | string[]
  children: ReactNode | ((props: FieldChildProps) => ReactNode)
  className?: string
  htmlFor?: string
}

export interface FieldChildProps {
  id: string
  'aria-describedby': string | undefined
  invalid: boolean
}

export function Field({ label, description, error, children, className, htmlFor }: FieldProps) {
  const generatedId = useId()
  const id = htmlFor ?? generatedId
  const descriptionId = description ? `${id}-description` : undefined
  const errorId = error ? `${id}-error` : undefined
  const describedBy = [descriptionId, errorId].filter(Boolean).join(' ') || undefined
  const errorMessage = Array.isArray(error) ? error[0] : error

  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <label htmlFor={id} className="text-sm font-medium text-foreground">
        {label}
      </label>
      {typeof children === 'function'
        ? children({ id, 'aria-describedby': describedBy, invalid: Boolean(error) })
        : children}
      {description ? (
        <p id={descriptionId} className="text-xs text-subtle">
          {description}
        </p>
      ) : null}
      {errorMessage ? (
        <p id={errorId} className="text-xs text-danger">
          {errorMessage}
        </p>
      ) : null}
    </div>
  )
}
