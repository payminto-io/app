export const WITHDRAWAL_STATES = [
  "pending-otp",
  "pending-approval",
  "pending",
  "initiated",
  "sent",
  "processed",
  "failed",
  "cancelled",
] as const;

export type WithdrawalState = (typeof WITHDRAWAL_STATES)[number];

const WITHDRAWAL_STATUS_KEYS: Record<WithdrawalState, string> = {
  "pending-otp": "pending_otp",
  "pending-approval": "pending_approval",
  pending: "pending",
  initiated: "initiated",
  sent: "sent",
  processed: "processed",
  failed: "failed",
  cancelled: "cancelled",
};

export function presentWithdrawalState(state: WithdrawalState): string {
  return WITHDRAWAL_STATUS_KEYS[state];
}

export interface Withdrawal {
  id: number;
  externalPlatformID: number;
  memberID: number;
  referenceID?: string;
  recipientAddress: string;
  blockchainCode: string;
  currencyCode: string;
  amount: string;
  memo?: string;
  state: WithdrawalState;
  transactionHash?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateWithdrawalInput {
  recipientAddress: string;
  blockchainCode: string;
  currencyCode: string;
  amount: string;
  memo?: string;
}

export interface CreateWithdrawalResult {
  withdrawal: Withdrawal;
  otp: {
    otpRequired: boolean;
    message?: string;
  };
}

type JsonRecord = Record<string, unknown>;

function record(value: unknown, label: string): JsonRecord {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new Error(`Invalid ${label} response`);
  }
  return value as JsonRecord;
}

function requiredString(value: unknown, field: string): string {
  if (typeof value !== "string") throw new Error(`Invalid withdrawal ${field}`);
  return value;
}

function requiredNumber(value: unknown, field: string): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw new Error(`Invalid withdrawal ${field}`);
  }
  return value;
}

function normalizeState(value: unknown): WithdrawalState {
  const state = requiredString(value, "state");
  if (!WITHDRAWAL_STATES.includes(state as WithdrawalState)) {
    throw new Error(`Invalid withdrawal state: ${state}`);
  }
  return state as WithdrawalState;
}

export function toCreateWithdrawalRequest(input: CreateWithdrawalInput) {
  if (typeof input.amount !== "string") {
    throw new Error("Invalid withdrawal amount");
  }
  return {
    blockchainCode: input.blockchainCode,
    currencyCode: input.currencyCode,
    amount: input.amount,
    toAddress: input.recipientAddress,
    ...(input.memo ? { memo: input.memo } : {}),
  };
}

export function normalizeWithdrawal(value: unknown): Withdrawal {
  const wire = record(value, "withdrawal");
  return {
    id: requiredNumber(wire.id, "id"),
    externalPlatformID: requiredNumber(
      wire.externalPlatformID,
      "externalPlatformID"
    ),
    memberID: requiredNumber(wire.memberID, "memberID"),
    ...(typeof wire.referenceID === "string"
      ? { referenceID: wire.referenceID }
      : {}),
    recipientAddress: requiredString(wire.toAddress, "toAddress"),
    blockchainCode: requiredString(wire.blockchainCode, "blockchainCode"),
    currencyCode: requiredString(wire.currencyCode, "currencyCode"),
    amount: requiredString(wire.amount, "amount"),
    ...(typeof wire.memo === "string" ? { memo: wire.memo } : {}),
    state: normalizeState(wire.state),
    ...(typeof wire.transactionHash === "string"
      ? { transactionHash: wire.transactionHash }
      : {}),
    createdAt: requiredString(wire.createdAt, "createdAt"),
    updatedAt: requiredString(wire.updatedAt, "updatedAt"),
  };
}

export function unwrapWithdrawal(value: unknown): Withdrawal {
  return normalizeWithdrawal(record(value, "withdrawal envelope").withdrawal);
}

export function normalizeCreatedWithdrawal(value: unknown): CreateWithdrawalResult {
  const envelope = record(value, "withdrawal create envelope");
  const otp = record(envelope.otp, "withdrawal OTP");
  if (typeof otp.otpRequired !== "boolean") {
    throw new Error("Invalid withdrawal OTP requirement");
  }
  return {
    withdrawal: normalizeWithdrawal(envelope.withdrawal),
    otp: {
      otpRequired: otp.otpRequired,
      ...(typeof otp.message === "string" ? { message: otp.message } : {}),
    },
  };
}

export function normalizeWithdrawalList(value: unknown): {
  withdrawals: Withdrawal[];
} {
  const envelope = record(value, "withdrawal list envelope");
  if (!Array.isArray(envelope.withdrawals)) {
    throw new Error("Invalid withdrawal list");
  }
  return { withdrawals: envelope.withdrawals.map(normalizeWithdrawal) };
}
