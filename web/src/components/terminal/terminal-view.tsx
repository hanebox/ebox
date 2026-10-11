import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Terminal } from '@xterm/xterm'
import { RotateCw, Wifi, WifiOff } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'

import { terminalWebSocketURL } from '@/api/endpoints'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'

import '@xterm/xterm/css/xterm.css'

const terminalTheme = {
  background: '#0b0b12',
  foreground: '#d6d6e4',
  cursor: '#8b8bf5',
  selectionBackground: '#2a2a44',
  black: '#1b1b28',
  red: '#ff6b81',
  green: '#5ddb9a',
  yellow: '#f5c76a',
  blue: '#7aa2f7',
  magenta: '#bb9af7',
  cyan: '#7dcfff',
  white: '#d6d6e4',
  brightBlack: '#414155',
  brightRed: '#ff8fa3',
  brightGreen: '#7ee2b8',
  brightYellow: '#ffd98a',
  brightBlue: '#9ab8ff',
  brightMagenta: '#d0b8ff',
  brightCyan: '#a4e0ff',
  brightWhite: '#ffffff',
}

type Status = 'connecting' | 'open' | 'closed'

export function TerminalView({ target, className }: { target: string; className?: string }) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const terminalRef = useRef<Terminal | null>(null)
  const socketRef = useRef<WebSocket | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const [status, setStatus] = useState<Status>('connecting')
  const [attempt, setAttempt] = useState(0)

  const reconnect = useCallback(() => setAttempt((value) => value + 1), [])

  useEffect(() => {
    const element = containerRef.current
    if (!element || !target) {
      return
    }

    const terminal = new Terminal({
      theme: terminalTheme,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
      fontSize: 12,
      lineHeight: 1.35,
      cursorBlink: true,
      scrollback: 5000,
      convertEol: false,
      macOptionIsMeta: true,
    })

    const fit = new FitAddon()
    terminal.loadAddon(fit)
    terminal.loadAddon(new WebLinksAddon())
    terminal.open(element)

    try {
      fit.fit()
    } catch {
      // The container may not be measurable yet.
    }

    terminalRef.current = terminal
    fitRef.current = fit
    setStatus('connecting')

    const socket = new WebSocket(terminalWebSocketURL(target, terminal.cols, terminal.rows))
    socket.binaryType = 'arraybuffer'
    socketRef.current = socket

    const sendResize = () => {
      if (socket.readyState !== WebSocket.OPEN) {
        return
      }

      socket.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }))
    }

    socket.onopen = () => {
      setStatus('open')
      try {
        fit.fit()
      } catch {
        // ignore
      }
      sendResize()
      terminal.focus()
    }

    socket.onmessage = (event) => {
      if (typeof event.data === 'string') {
        terminal.write(event.data)

        return
      }

      terminal.write(new Uint8Array(event.data as ArrayBuffer))
    }

    socket.onclose = () => setStatus('closed')
    socket.onerror = () => setStatus('closed')

    const disposable = terminal.onData((data) => {
      if (socket.readyState === WebSocket.OPEN) {
        socket.send(new TextEncoder().encode(data))
      }
    })

    const observer = new ResizeObserver(() => {
      try {
        fit.fit()
      } catch {
        return
      }
      sendResize()
    })
    observer.observe(element)

    return () => {
      observer.disconnect()
      disposable.dispose()
      socket.close()
      terminal.dispose()
      socketRef.current = null
      terminalRef.current = null
      fitRef.current = null
    }
  }, [target, attempt])

  return (
    <div className={cn('flex flex-col overflow-hidden rounded-lg border border-border bg-[#0b0b12]', className)}>
      <div className="flex items-center justify-between gap-2 border-b border-border bg-surface-raised px-3 py-1.5">
        <div className="flex items-center gap-2 text-[11px]">
          <Badge variant={status === 'open' ? 'success' : status === 'connecting' ? 'info' : 'danger'}>
            {status === 'open' ? (
              <>
                <Wifi aria-hidden className="size-3" /> connected
              </>
            ) : status === 'connecting' ? (
              'connecting'
            ) : (
              <>
                <WifiOff aria-hidden className="size-3" /> disconnected
              </>
            )}
          </Badge>
          <span className="font-mono text-subtle">{target === 'host' ? 'host' : target.slice(0, 12)}</span>
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="sm" onClick={() => terminalRef.current?.clear()}>
            Clear
          </Button>
          <Button variant="ghost" size="sm" onClick={reconnect}>
            <RotateCw aria-hidden className="size-3.5" />
            Reconnect
          </Button>
        </div>
      </div>
      <div ref={containerRef} className="h-[28rem] w-full p-2" />
    </div>
  )
}
