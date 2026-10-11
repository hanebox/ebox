import { IconMoon, IconSun } from '@tabler/icons-react'

import { Button } from '@/components/ui/button'
import { Tooltip } from '@/components/ui/tooltip'
import { useTheme } from '@/hooks/use-theme'
import { cn } from '@/lib/cn'

export function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const dark = theme === 'dark'
  const label = dark ? 'Switch to light theme' : 'Switch to dark theme'

  return (
    <Tooltip content={label}>
      <Button variant="ghost" size="icon" onClick={toggle} aria-label={label}>
        <span className="relative inline-flex size-4 items-center justify-center">
          <IconSun
            aria-hidden
            className={cn(
              'absolute inset-0 size-4 transition-[opacity,filter,scale] duration-300 ease-icon',
              dark ? 'scale-100 opacity-100 blur-0' : 'scale-[0.25] opacity-0 blur-[4px]',
            )}
          />
          <IconMoon
            aria-hidden
            className={cn(
              'size-4 transition-[opacity,filter,scale] duration-300 ease-icon',
              dark ? 'scale-[0.25] opacity-0 blur-[4px]' : 'scale-100 opacity-100 blur-0',
            )}
          />
        </span>
      </Button>
    </Tooltip>
  )
}
