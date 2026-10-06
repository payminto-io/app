import type { QueryKey } from "@tanstack/react-query"

export type LiveChannel = string

export type LiveEvent<T = unknown> = {
  channel: LiveChannel
  type: string
  payload?: T
  // Optional explicit replacement payload — when present, providers may
  // setQueryData() instead of full invalidation.
  data?: T
}

export type LiveSubscription = {
  channel: LiveChannel
  queryKey: QueryKey
}
