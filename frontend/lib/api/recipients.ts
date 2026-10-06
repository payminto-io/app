/**
 * Recipients (address book) domain API.
 * Routes: /recipients (CRUD). externalPlatformID is derived from the token,
 * never sent in the body (PR0 hole #17 fix).
 *
 * Backend wraps the list response as { recipients: [...] } — we unwrap so
 * hook consumers can work with a plain array.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface Recipient {
  id: number;
  externalPlatformID: number;
  memberID: number;
  name: string;
  email?: string;
  blockchainCode: string;
  currencyCode: string;
  address: string;
  memo?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateRecipientInput {
  name: string;
  email?: string;
  blockchainCode: string;
  currencyCode: string;
  address: string;
  memo?: string;
}

export interface UpdateRecipientInput {
  name?: string;
  email?: string;
  address?: string;
  memo?: string;
}

export const recipientsApi = {
  list: async (): Promise<Recipient[]> => {
    try {
      const res = await apiFetch<{ recipients: Recipient[] }>("/recipients");
      return res.recipients ?? [];
    } catch (err) {
      if (isApiError(err) && err.isNotFound) return [];
      throw err;
    }
  },
  get: (id: number) => apiFetch<Recipient>(`/recipients/${id}`),
  create: (input: CreateRecipientInput) =>
    apiFetch<Recipient>("/recipients", { method: "POST", body: input }),
  update: (id: number, input: UpdateRecipientInput) =>
    apiFetch<Recipient>(`/recipients/${id}`, { method: "PUT", body: input }),
  remove: (id: number) =>
    apiFetch<void>(`/recipients/${id}`, { method: "DELETE" }),
};
