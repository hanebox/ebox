import type { StreamEvent } from '@/api/types'

export type StreamStatus = 'connecting' | 'open' | 'closed'

type EventListener = (event: StreamEvent) => void
type StatusListener = (status: StreamStatus) => void

const reconnectBaseDelay = 1000
const reconnectMaxDelay = 15_000

class EventStreamClient {
  private source: EventSource | null = null
  private eventListeners = new Set<EventListener>()
  private statusListeners = new Set<StatusListener>()
  private reconnectAttempts = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private started = false
  private status: StreamStatus = 'closed'

  start() {
    if (this.started) {
      return
    }

    this.started = true
    this.connect()
  }

  stop() {
    this.started = false
    this.clearReconnect()
    this.source?.close()
    this.source = null
    this.setStatus('closed')
  }

  getStatus(): StreamStatus {
    return this.status
  }

  subscribe(listener: EventListener): () => void {
    this.eventListeners.add(listener)
    return () => this.eventListeners.delete(listener)
  }

  subscribeStatus(listener: StatusListener): () => void {
    this.statusListeners.add(listener)
    listener(this.status)
    return () => this.statusListeners.delete(listener)
  }

  private connect() {
    if (this.source) {
      return
    }

    this.setStatus('connecting')

    const source = new EventSource('/api/v1/events', { withCredentials: true })
    this.source = source

    source.addEventListener('ready', () => {
      this.reconnectAttempts = 0
      this.setStatus('open')
    })

    source.addEventListener('message', (raw) => {
      if (!(raw instanceof MessageEvent)) {
        return
      }

      try {
        const event = JSON.parse(raw.data as string) as StreamEvent
        for (const listener of this.eventListeners) {
          listener(event)
        }
      } catch {
        // ignore malformed payloads
      }
    })

    source.onerror = () => {
      source.close()
      this.source = null

      if (!this.started) {
        this.setStatus('closed')
        return
      }

      this.setStatus('connecting')
      this.scheduleReconnect()
    }
  }

  private scheduleReconnect() {
    this.clearReconnect()

    const delay = Math.min(
      reconnectBaseDelay * 2 ** this.reconnectAttempts,
      reconnectMaxDelay,
    )
    this.reconnectAttempts += 1

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      if (this.started) {
        this.connect()
      }
    }, delay)
  }

  private clearReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  private setStatus(status: StreamStatus) {
    if (this.status === status) {
      return
    }

    this.status = status
    for (const listener of this.statusListeners) {
      listener(status)
    }
  }
}

export const eventStream = new EventStreamClient()
