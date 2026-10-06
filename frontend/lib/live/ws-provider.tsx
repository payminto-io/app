"use client"

import * as React from "react"
import { useQueryClient, type QueryKey } from "@tanstack/react-query"

import type { LiveChannel, LiveEvent } from "./types"

// ---------------------------------------------------------------------------
// LiveProvider — single WebSocket per tab. In dev/mocked mode this is a no-op
// stub that logs what it WOULD do; the real backend wiring is later.
//
// Per AI_CONTEXT.md and the architectural decision in the implementation
// plan: we open one WS via `POST /api/v1/websocket-token/create` and route
// incoming events to TanStack Query keys via the channel registry.
// ---------------------------------------------------------------------------

type Registry = Map<LiveChannel, Set<string>> // channel -> serialized query keys

type LiveContextValue = {
  subscribe: (channel: LiveChannel, queryKey: QueryKey) => () => void
  status: "idle" | "connecting" | "open" | "closed" | "error" | "stub"
}

const LiveContext = React.createContext<LiveContextValue | null>(null)

const WS_TOKEN_ENDPOINT = "/api/v1/websocket-token/create"

const ENABLED =
  typeof process !== "undefined" &&
  process.env.NEXT_PUBLIC_PAYMINTO_LIVE === "1"

export function LiveProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient()
  const registryRef = React.useRef<Registry>(new Map())
  const [status, setStatus] = React.useState<LiveContextValue["status"]>(
    ENABLED ? "idle" : "stub"
  )

  // Stub mode: no socket. Real connection wiring lives below the early return.
  React.useEffect(() => {
    if (ENABLED) return
    console.info(
      "[live] stub mode — WS would connect via POST",
      WS_TOKEN_ENDPOINT,
      "(set NEXT_PUBLIC_PAYMINTO_LIVE=1 once backend is wired)"
    )
  }, [])

  // Real mode: open WS, dispatch events. Reconnect with backoff.
  React.useEffect(() => {
    if (!ENABLED) return

    let socket: WebSocket | null = null
    let cancelled = false
    let retry = 0
    let timer: ReturnType<typeof setTimeout> | null = null

    async function connect() {
      try {
        setStatus("connecting")
        const res = await fetch(WS_TOKEN_ENDPOINT, { method: "POST" })
        if (!res.ok) throw new Error(`token ${res.status}`)
        const { url } = (await res.json()) as { url: string }
        if (cancelled) return

        socket = new WebSocket(url)
        socket.onopen = () => {
          retry = 0
          setStatus("open")
        }
        socket.onmessage = (ev) => {
          try {
            const event = JSON.parse(ev.data as string) as LiveEvent
            const keys = registryRef.current.get(event.channel)
            if (!keys) return
            for (const serialized of keys) {
              const queryKey = JSON.parse(serialized) as QueryKey
              if (event.data !== undefined) {
                queryClient.setQueryData(queryKey, event.data)
              } else {
                queryClient.invalidateQueries({ queryKey })
              }
            }
          } catch (err) {
            console.warn("[live] bad message", err)
          }
        }
        socket.onerror = () => setStatus("error")
        socket.onclose = () => {
          if (cancelled) return
          setStatus("closed")
          // Exponential backoff up to 30s
          const delay = Math.min(30_000, 500 * 2 ** retry++)
          timer = setTimeout(connect, delay)
        }
      } catch (err) {
        console.warn("[live] connect failed", err)
        setStatus("error")
        const delay = Math.min(30_000, 500 * 2 ** retry++)
        timer = setTimeout(connect, delay)
      }
    }

    connect()

    return () => {
      cancelled = true
      if (timer) clearTimeout(timer)
      if (socket && socket.readyState === WebSocket.OPEN) socket.close()
    }
  }, [queryClient])

  const subscribe = React.useCallback(
    (channel: LiveChannel, queryKey: QueryKey) => {
      const serialized = JSON.stringify(queryKey)
      const reg = registryRef.current
      let set = reg.get(channel)
      if (!set) {
        set = new Set()
        reg.set(channel, set)
      }
      set.add(serialized)

      return () => {
        const s = registryRef.current.get(channel)
        if (!s) return
        s.delete(serialized)
        if (s.size === 0) registryRef.current.delete(channel)
      }
    },
    []
  )

  const value = React.useMemo<LiveContextValue>(
    () => ({ subscribe, status }),
    [subscribe, status]
  )

  return <LiveContext.Provider value={value}>{children}</LiveContext.Provider>
}

export function useLiveContext(): LiveContextValue {
  const ctx = React.useContext(LiveContext)
  if (!ctx) {
    // Tolerant fallback: if a page outside the dashboard layout calls a hook
    // that depends on this, return a no-op so we don't crash render.
    return {
      subscribe: () => () => {},
      status: "stub",
    }
  }
  return ctx
}
