import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-background px-4 text-center">
      <p className="font-mono text-sm text-subtle">404</p>
      <h1 className="text-lg font-semibold tracking-tight">Page not found</h1>
      <p className="max-w-sm text-sm text-muted">
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
