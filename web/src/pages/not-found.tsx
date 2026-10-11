import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { LogoMark } from '@/components/ui/logo'

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-background px-4 text-center">
      <LogoMark className="mb-1 size-9" />
      <p className="font-mono text-xs text-subtle">404</p>
      <h1 className="text-lg font-semibold tracking-[-0.02em] text-foreground">Page not found</h1>
      <p className="max-w-sm text-ui text-muted">
        The page you are looking for does not exist or has moved.
      </p>
      <Link to="/" className="mt-2">
        <Button variant="primary" size="md">
          Back to dashboard
        </Button>
      </Link>
    </div>
  )
}
