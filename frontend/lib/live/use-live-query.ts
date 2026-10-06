"use client"

import * as React from "react"
import {
  useQuery,
  type QueryKey,
  type UseQueryOptions,
  type UseQueryResult,
} from "@tanstack/react-query"

import { useLiveContext } from "./ws-provider"
import type { LiveChannel } from "./types"

type UseLiveQueryOptions<TData> = Omit<
  UseQueryOptions<TData, Error, TData, QueryKey>,
  "queryKey" | "queryFn"
> & {
  /**
   * WS channel to subscribe to. Defaults to the first segment of the
   * query key when it is a string (e.g. ['payments'] → 'payments').
   */
  channel?: LiveChannel
}

export function useLiveQuery<TData>(
  queryKey: QueryKey,
  queryFn: () => Promise<TData>,
  opts: UseLiveQueryOptions<TData> = {}
): UseQueryResult<TData, Error> {
  const { channel, ...rest } = opts
  const live = useLiveContext()

  const resolvedChannel: LiveChannel =
    channel ??
    (Array.isArray(queryKey) && typeof queryKey[0] === "string"
      ? (queryKey[0] as string)
      : "default")

  React.useEffect(() => {
    return live.subscribe(resolvedChannel, queryKey)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resolvedChannel, JSON.stringify(queryKey)])

  return useQuery<TData, Error, TData, QueryKey>({
    queryKey,
    queryFn,
    ...rest,
  })
}
