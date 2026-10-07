"use client";

/**
 * Admin-section hooks bundled into one file because each domain has
 * only a handful of calls and the RBAC-gated pages import many of them
 * together. Every key lives under qk.admin.*.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  externalPlatformsApi,
  type ExternalPlatform,
  type ExternalPlatformBlockchainCurrency,
  type CreateExternalPlatformInput,
  type UpsertEPBCInput,
} from "@/lib/api/external-platforms";
import { membersApi, type Member, type InviteMemberInput } from "@/lib/api/members";
import { rolesApi, permissionsApi, type Role, type Permission } from "@/lib/api/roles";
import { configurationsApi, type Configuration } from "@/lib/api/configurations";
import { systemApi, type SystemInfo, type WorkerStatus } from "@/lib/api/system";
import { missedDepositsApi, type MissedDeposit } from "@/lib/api/missed-deposits";
export type {
  ExternalPlatform, ExternalPlatformBlockchainCurrency,
  CreateExternalPlatformInput, UpsertEPBCInput,
  Member, InviteMemberInput,
  Role, Permission,
  Configuration,
  SystemInfo, WorkerStatus,
  MissedDeposit,
};
import { qk } from "@/lib/query/keys";
import { useMemberScope } from "./use-member-scope";

/** Every admin hook must call this to get the member scope for query keys. */
function useMS() {
  return useMemberScope();
}

// External platforms
export function useExternalPlatforms() {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.externalPlatforms.list(ms),
    queryFn: () => externalPlatformsApi.list(),
  });
}
export function useExternalPlatform(id: number | undefined) {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.externalPlatforms.detail(ms, id ?? 0),
    queryFn: () => externalPlatformsApi.get(id as number),
    enabled: Boolean(id),
  });
}
export function useExternalPlatformCurrencies(id: number | undefined) {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.externalPlatforms.currencies(ms, id ?? 0),
    queryFn: () => externalPlatformsApi.currencies(id as number),
    enabled: Boolean(id),
  });
}
export function useCreateExternalPlatform() {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (input: CreateExternalPlatformInput) =>
      externalPlatformsApi.create(input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.admin.externalPlatforms.list(ms) }),
  });
}
export function useRegenerateEPAPIKey() {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (id: number) => externalPlatformsApi.regenerateAPIKey(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: qk.admin.externalPlatforms.detail(ms, id) });
      qc.invalidateQueries({ queryKey: qk.admin.externalPlatforms.list(ms) });
    },
  });
}
export function useUpsertEPBC(platformId: number) {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (input: UpsertEPBCInput) =>
      externalPlatformsApi.upsertCurrency(platformId, input),
    onSuccess: () =>
      qc.invalidateQueries({
        queryKey: qk.admin.externalPlatforms.currencies(ms, platformId),
      }),
  });
}

// Members
export function useAdminMembers() {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.members.list(ms),
    queryFn: () => membersApi.adminList(),
  });
}
export function useAdminMember(id: number | undefined) {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.members.detail(ms, id ?? 0),
    queryFn: () => membersApi.adminGet(id as number),
    enabled: Boolean(id),
  });
}
export function useInviteMember() {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (input: InviteMemberInput) => membersApi.adminInvite(input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.admin.members.list(ms) }),
  });
}
export function useDeactivateMember() {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (id: number) => membersApi.adminDeactivate(id),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.admin.members.list(ms) }),
  });
}

// Roles + permissions
export function useRoles() {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.roles.list(ms),
    queryFn: () => rolesApi.list(),
  });
}
export function usePermissions() {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.permissions.list(ms),
    queryFn: () => permissionsApi.list(),
    staleTime: 5 * 60_000,
  });
}
export function useSetRolePermissions() {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (input: { id: number; permissions: string[] }) =>
      rolesApi.setPermissions(input.id, input.permissions),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.admin.roles.list(ms) }),
  });
}

// Configurations
export function useConfigurations() {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.configurations.list(ms),
    queryFn: () => configurationsApi.list(),
  });
}
export function useSetConfiguration() {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (input: { key: string; value: string }) =>
      configurationsApi.set(input.key, input.value),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.admin.configurations.list(ms) }),
  });
}

// System / workers
export function useSystemInfo() {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.system.info(ms),
    queryFn: () => systemApi.info(),
    staleTime: 30_000,
  });
}
export function useWorkersStatus() {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.system.workers(ms),
    queryFn: () => systemApi.workers(),
    refetchInterval: 10_000,
  });
}

// Missed deposits
export function useMissedDeposits(filters?: {
  status?: string;
  limit?: number;
  offset?: number;
}) {
  const ms = useMS();
  return useQuery({
    queryKey: qk.admin.missedDeposits.list(ms, filters),
    queryFn: () => missedDepositsApi.list(filters),
  });
}
export function useResolveMissedDeposit() {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (input: { id: number; paymentReferenceID?: string }) =>
      missedDepositsApi.resolve(input.id, input.paymentReferenceID),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.admin.missedDeposits.list(ms) }),
  });
}
export function useDismissMissedDeposit() {
  const qc = useQueryClient();
  const ms = useMS();
  return useMutation({
    mutationFn: (input: { id: number; reason?: string }) =>
      missedDepositsApi.dismiss(input.id, input.reason),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.admin.missedDeposits.list(ms) }),
  });
}
