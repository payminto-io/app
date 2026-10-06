"use client";

import { useAuth } from "@/lib/auth/store";

export function useMemberScope(): { memberId: number; ready: boolean } {
  const member = useAuth((s) => s.member);
  return {
    memberId: member?.id ?? 0,
    ready: Boolean(member?.id),
  };
}
