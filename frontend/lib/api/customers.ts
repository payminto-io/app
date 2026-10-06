/**
 * Customers domain API.
 *
 * Customers are auto-created member records (member_type = "customer") that
 * appear when a payment is created with a customerEmail. This module exposes
 * a read-only list endpoint for the merchant dashboard.
 */
import { apiFetch } from "./client";

export interface Customer {
  id: number;
  name: string;
  email?: string;
  customerID: string;
  state: string;
  memberType: string;
  createdAt: string;
  updatedAt: string;
}

export interface CustomersListResponse {
  customers: Customer[];
  total: number;
}

export const customersApi = {
  list: (params?: { search?: string; limit?: number; offset?: number }) => {
    const query = new URLSearchParams();
    if (params?.search) query.set("search", params.search);
    if (params?.limit) query.set("limit", String(params.limit));
    if (params?.offset) query.set("offset", String(params.offset));
    const suffix = query.toString() ? `?${query}` : "";
    return apiFetch<CustomersListResponse>(`/customers${suffix}`);
  },
};
