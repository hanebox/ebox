import { useCallback, useEffect, useState } from 'react'

type Theme = 'light' | 'dark'

const storageKey = 'ebox-theme'

function currentTheme(): Theme {
  if (typeof document === 'undefined') {
    return 'dark'
  }

  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

function applyTheme(theme: Theme) {
  const root = document.documentElement
  const silent = document.createElement('style')
  silent.textContent = '*,*::before,*::after{transition:none !important}'
  document.head.append(silent)

  root.classList.toggle('dark', theme === 'dark')
  void root.offsetHeight

  requestAnimationFrame(() => silent.remove())
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(currentTheme)

  useEffect(() => {
    applyTheme(theme)
    try {
      localStorage.setItem(storageKey, theme)
    } catch {
      // storage unavailable
    }
  }, [theme])

  const toggle = useCallback(() => {
    setTheme((previous) => (previous === 'dark' ? 'light' : 'dark'))
  }, [])

  return { theme, setTheme, toggle }
}
