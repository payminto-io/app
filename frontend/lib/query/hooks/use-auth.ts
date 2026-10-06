"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { signin, signup, signout, me } from "@/lib/api/auth";
import { membersApi, type ChangePasswordInput, type UpdateProfileInput } from "@/lib/api/members";
import { authStore } from "@/lib/auth/store";

export function useSignin() {
  return useMutation({
    mutationFn: signin,
    onSuccess: (result) => {
      authStore.getState().setSession(result);
    },
    onError: () => {
      authStore.getState().clear();
    },
  });
}

export function useSignout() {
  const router = useRouter();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: signout,
    onSettled: () => {
      authStore.getState().clear();
      qc.clear();
      router.replace("/signin");
      router.refresh();
    },
  });
}

export function useSignup() {
  return useMutation({
    mutationFn: signup,
    onSuccess: (result) => {
      authStore.getState().setSession(result);
    },
  });
}

export function useUpdateProfile() {
  return useMutation({
    mutationFn: (input: UpdateProfileInput) => membersApi.updateProfile(input),
    onSuccess: (member) => {
      const current = authStore.getState().member;
      if (current) {
        authStore.getState().setSession({
          accessToken: authStore.getState().accessToken!,
          member: { ...current, name: member.name, email: member.email },
        });
      }
    },
  });
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (input: ChangePasswordInput) =>
      membersApi.changeMyPassword(input),
  });
}

export { me };
export { isApiError } from "@/lib/api/errors";
export type { PublicPayment } from "@/lib/api/public";
