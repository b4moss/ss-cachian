/** Shared errors and Layer contract (Go parity). */

export class SscachianError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "SscachianError";
  }
}

export const ErrEmptyKey = new SscachianError("sscachian: empty key");
export const ErrNegativeTTL = new SscachianError("sscachian: negative TTL");
export const ErrNotInteger = new SscachianError("sscachian: value is not an integer");
export const ErrIncrOverflow = new SscachianError("sscachian: incr overflow");
export const ErrNoLayer = new SscachianError("sscachian: no layers configured");
export const ErrNoKeyBuilder = new SscachianError("sscachian: key builder not configured");
export const ErrInvalidVersion = new SscachianError("sscachian: version must be >= 1");
export const ErrNoLoader = new SscachianError("sscachian: loader not configured");
export const ErrTypeMismatch = new SscachianError("sscachian: cached value type mismatch");
export const ErrInvalidContext = new SscachianError("sscachian: invalid key context");
export const ErrCorruptVersion = new SscachianError("sscachian: corrupt current-version value");
export const ErrInvalidLayerIndex = new SscachianError("sscachian: invalid layer index");
export const ErrInvalidConfig = new SscachianError("sscachian: invalid config");

/** On-Layer storage wrapper. */
export type Entry = {
  value: unknown;
  createdAt: Date;
  /** Invalid / epoch 0 means no expiry (Go zero time). */
  expiresAt: Date | null;
};

export function noExpiry(): null {
  return null;
}

export function isExpired(expiresAt: Date | null, now = new Date()): boolean {
  if (expiresAt == null) return false;
  if (expiresAt.getTime() === 0) return false;
  return !(expiresAt.getTime() > now.getTime());
}

/** TTL in milliseconds. 0 = no expiry from TTL arg. */
export type DurationMs = number;

export type Layer = {
  get(key: string, signal?: AbortSignal): Promise<{ entry: Entry; hit: boolean }>;
  set(key: string, entry: Entry, ttl: DurationMs, signal?: AbortSignal): Promise<void>;
  delete(key: string, signal?: AbortSignal): Promise<void>;
  incr(key: string, signal?: AbortSignal): Promise<number>;
  purgeExact(logicalPrefix: string, signal?: AbortSignal): Promise<void>;
  purgePrefix(prefix: string, signal?: AbortSignal): Promise<void>;
};

/** True when key is `{logicalPrefix}:{digits}` (digits non-empty). */
export function isVersionDataKey(logicalPrefix: string, key: string): boolean {
  if (logicalPrefix === "" || key === "") return false;
  const want = `${logicalPrefix}:`;
  if (!key.startsWith(want)) return false;
  const rest = key.slice(want.length);
  if (rest === "") return false;
  for (const ch of rest) {
    if (ch < "0" || ch > "9") return false;
  }
  return true;
}

export type KeyContext = {
  appSlug: string;
  tenantId: string;
  queryType: string;
};

export type KeyBuilder = (kc: KeyContext, signal?: AbortSignal) => Promise<string> | string;
export type Loader<T> = (kc: KeyContext, signal?: AbortSignal) => Promise<T>;
export type TypeGuard<T> = (v: unknown) => v is T;

export function defaultKeyBuilder(kc: KeyContext): string {
  if (!kc.appSlug || !kc.tenantId || !kc.queryType) {
    throw ErrInvalidContext;
  }
  return `${kc.appSlug}:cache:${kc.tenantId}:${kc.queryType}`;
}
