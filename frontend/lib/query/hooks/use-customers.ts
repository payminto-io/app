"use client";

import { useQuery } from "@tanstack/react-query";
import {
  customersApi,
  type Customer,
  type CustomersListResponse,
} from "@/lib/api/customers";
export type { Customer, CustomersListResponse };
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

export function useCustomersList(filters?: {
  search?: string;
  limit?: number;
  offset?: number;
}) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.customers.list({ platformId: scope.platformId }, filters),
    queryFn: () => customersApi.list(filters),
    enabled: scope.ready,
  });
}
