import {
  type DurationMs,
  type Entry,
  type Layer,
  ErrEmptyKey,
  ErrIncrOverflow,
  ErrNegativeTTL,
  ErrNotInteger,
  isExpired,
  isVersionDataKey,
} from "../types.js";

/** Process-local Layer (mutex via sync Map ops on event loop). */
export class MemoryStore implements Layer {
  #m = new Map<string, unknown>();

  /** Test helper: store arbitrary raw value. */
  injectRaw(key: string, v: unknown): void {
    this.#m.set(key, v);
  }

  async get(key: string, _signal?: AbortSignal): Promise<{ entry: Entry; hit: boolean }> {
    if (key === "") throw ErrEmptyKey;
    const raw = this.#m.get(key);
    if (raw === undefined) return { entry: emptyEntry(), hit: false };
    if (!isEntry(raw)) {
      this.#m.delete(key);
      return { entry: emptyEntry(), hit: false };
    }
    if (isExpired(raw.expiresAt)) {
      this.#m.delete(key);
      return { entry: emptyEntry(), hit: false };
    }
    return { entry: { ...raw }, hit: true };
  }

  async set(key: string, entry: Entry, ttl: DurationMs, _signal?: AbortSignal): Promise<void> {
    if (key === "") throw ErrEmptyKey;
    if (ttl < 0) throw ErrNegativeTTL;
    const now = new Date();
    const stored: Entry = {
      value: entry.value,
      createdAt: entry.createdAt.getTime() === 0 ? now : new Date(entry.createdAt),
      expiresAt: entry.expiresAt,
    };
    if (ttl > 0) {
      stored.expiresAt = new Date(now.getTime() + ttl);
    }
    this.#m.set(key, stored);
  }

  async delete(key: string, _signal?: AbortSignal): Promise<void> {
    if (key === "") throw ErrEmptyKey;
    this.#m.delete(key);
  }

  async incr(key: string, _signal?: AbortSignal): Promise<number> {
    if (key === "") throw ErrEmptyKey;
    const raw = this.#m.get(key);
    let n = 0;
    if (raw !== undefined) {
      if (!isEntry(raw)) throw ErrNotInteger;
      n = asIncrInt(raw.value);
    }
    if (n >= Number.MAX_SAFE_INTEGER) throw ErrIncrOverflow;
    n += 1;
    this.#m.set(key, { value: n, createdAt: new Date(), expiresAt: null } satisfies Entry);
    return n;
  }

  async purgeExact(logicalPrefix: string, _signal?: AbortSignal): Promise<void> {
    if (logicalPrefix === "") throw ErrEmptyKey;
    for (const k of [...this.#m.keys()]) {
      if (isVersionDataKey(logicalPrefix, k)) this.#m.delete(k);
    }
  }
}

export function newMemoryStore(): MemoryStore {
  return new MemoryStore();
}

function emptyEntry(): Entry {
  return { value: undefined, createdAt: new Date(0), expiresAt: null };
}

function isEntry(v: unknown): v is Entry {
  if (v == null || typeof v !== "object") return false;
  const o = v as Record<string, unknown>;
  return "value" in o && "createdAt" in o && "expiresAt" in o;
}

function asIncrInt(v: unknown): number {
  if (typeof v === "number" && Number.isInteger(v)) return v;
  if (typeof v === "bigint") {
    const n = Number(v);
    if (!Number.isSafeInteger(n)) throw ErrNotInteger;
    return n;
  }
  throw ErrNotInteger;
}
