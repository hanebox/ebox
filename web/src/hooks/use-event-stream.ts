import { useEffect, useRef, useState } from 'react'

import type { StreamEvent } from '@/api/types'
import { eventStream, type StreamStatus } from '@/lib/events'

export function useEventStream(onEvent?: (event: StreamEvent) => void) {
  const handlerRef = useRef(onEvent)

  useEffect(() => {
    handlerRef.current = onEvent
  }, [onEvent])

  useEffect(() => {
    eventStream.start()

    const unsubscribe = eventStream.subscribe((event) => {
      handlerRef.current?.(event)
    })

    return unsubscribe
  }, [])
}

export function useStreamStatus(): StreamStatus {
  const [status, setStatus] = useState<StreamStatus>(() => eventStream.getStatus())

  useEffect(() => {
    return eventStream.subscribeStatus(setStatus)
  }, [])

  return status
}
