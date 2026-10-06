/**
 * Typed error thrown by `apiFetch` on any non-2xx response.
 * Callers can branch on `status` for 401/403/404/409 handling; the raw
 * `body` is kept so edge cases can introspect it without a second parse.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;

  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }

  get isUnauthorized(): boolean {
    return this.status === 401;
  }

  get isForbidden(): boolean {
    return this.status === 403;
  }

  get isNotFound(): boolean {
    return this.status === 404;
  }

  get isConflict(): boolean {
    return this.status === 409;
  }
}

/**
 * Narrow an unknown thrown value to ApiError without relying on instanceof
 * (which breaks across Vitest module boundaries).
 */
export function isApiError(err: unknown): err is ApiError {
  return (
    typeof err === "object" &&
    err !== null &&
    (err as { name?: string }).name === "ApiError" &&
    typeof (err as { status?: unknown }).status === "number"
  );
}
