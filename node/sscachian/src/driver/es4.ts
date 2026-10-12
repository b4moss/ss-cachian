import {
  open as es4Open,
  defaults as es4Defaults,
  ErrClosed,
  ErrNestedTx,
  ErrNotFound,
  isNotFound,
  type DB,
  type JsonRaw,
  type Tx,
} from "@b4moss/es4";

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

const incrMaxAttempts = 8;

/** Entry JSON B wire format. */
type WireEntry = {
  value: unknown;
  expires_at?: string;
};

export type Es4DriverOptions = {
  memoryOnly?: boolean;
  stateBackend?: string;
  statePath?: string;
  stateRedisUrl?: string;
  stateRedisKeyPrefix?: string;
  stateFirestoreProjectId?: string;
  stateFirestoreDatabaseId?: string;
  stateFirestoreCollection?: string;
};

export type Es4HTTPOptions = {
  baseUrl: string;
  fetchImpl?: typeof fetch;
};

/** Minimal es4 State surface used by the Layer (embedded or HTTP). */
export type Es4Backend = {
  set(key: string, value: JsonRaw, signal?: AbortSignal): Promise<void>;
  get(key: string, signal?: AbortSignal): Promise<JsonRaw>;
  delete(key: string, signal?: AbortSignal): Promise<void>;
  beginTx(signal?: AbortSignal): Promise<Tx>;
  scan(prefix: string, signal?: AbortSignal): Promise<string[]>;
  close(): Promise<void>;
};

function encodeEntry(value: unknown, expiresAt: Date | null): JsonRaw {
  const w: WireEntry = { value };
  if (expiresAt != null && expiresAt.getTime() !== 0) {
    // RFC3339 / ISO-8601 (es4 stores JSON text).
    w.expires_at = expiresAt.toISOString();
  }
  return JSON.stringify(w);
}

function decodeEntry(raw: JsonRaw): { value: unknown; expiresAt: Date | null; ok: boolean } {
  try {
    const text = typeof raw === "string" ? raw : Buffer.from(raw).toString("utf8");
    const parsed = JSON.parse(text) as unknown;
    if (parsed == null || typeof parsed !== "object" || Array.isArray(parsed)) {
      return { value: null, expiresAt: null, ok: false };
    }
    const obj = parsed as Record<string, unknown>;
    if (!("value" in obj)) return { value: null, expiresAt: null, ok: false };
    let expiresAt: Date | null = null;
    if (typeof obj.expires_at === "string" && obj.expires_at !== "") {
      const d = new Date(obj.expires_at);
      if (Number.isNaN(d.getTime())) return { value: null, expiresAt: null, ok: false };
      expiresAt = d;
    }
    return { value: obj.value, expiresAt, ok: true };
  } catch {
    return { value: null, expiresAt: null, ok: false };
  }
}

function asInt64(v: unknown): number | null {
  if (typeof v === "number" && Number.isInteger(v)) return v;
  if (typeof v === "bigint") {
    const n = Number(v);
    if (Number.isSafeInteger(n)) return n;
  }
  return null;
}

class EmbeddedBackend implements Es4Backend {
  constructor(private readonly db: DB) {}

  set(key: string, value: JsonRaw, signal?: AbortSignal): Promise<void> {
    return this.db.set(key, value, signal);
  }
  get(key: string, signal?: AbortSignal): Promise<JsonRaw> {
    return this.db.get(key, signal);
  }
  delete(key: string, signal?: AbortSignal): Promise<void> {
    return this.db.delete(key, signal);
  }
  beginTx(signal?: AbortSignal): Promise<Tx> {
    return this.db.beginTx(signal);
  }
  scan(prefix: string, signal?: AbortSignal): Promise<string[]> {
    return this.db.scan(prefix, signal);
  }
  async close(): Promise<void> {
    await this.db.close();
  }
}

export class Es4Store implements Layer {
  #backend: Es4Backend;
  #closed = false;

  constructor(backend: Es4Backend) {
    this.#backend = backend;
  }

  static async create(opts: Es4DriverOptions = {}): Promise<Es4Store> {
    const es4Opts = es4Defaults();
    es4Opts.memory_only = opts.memoryOnly ?? false;
    es4Opts.state_backend = opts.stateBackend ?? "";
    es4Opts.state_path = opts.statePath ?? "";
    es4Opts.state_redis_url = opts.stateRedisUrl ?? "";
    es4Opts.state_redis_key_prefix = opts.stateRedisKeyPrefix ?? "";
    es4Opts.state_firestore_project_id = opts.stateFirestoreProjectId ?? "";
    es4Opts.state_firestore_database_id = opts.stateFirestoreDatabaseId ?? "";
    es4Opts.state_firestore_collection = opts.stateFirestoreCollection ?? "";
    if (!opts.memoryOnly && !opts.stateBackend && !opts.statePath) {
      es4Opts.memory_only = true;
    }
    const db = await es4Open(es4Opts);
    return new Es4Store(new EmbeddedBackend(db));
  }

  static async createHTTP(opts: Es4HTTPOptions): Promise<Es4Store> {
    if (!opts.baseUrl) throw new Error("sscachian/es4: baseUrl is required");
    const backend = await HttpBackend.connect(opts);
    return new Es4Store(backend);
  }

  /** Test / advanced injection. */
  static fromBackend(backend: Es4Backend): Es4Store {
    return new Es4Store(backend);
  }

  async close(): Promise<void> {
    if (this.#closed) return;
    this.#closed = true;
    await this.#backend.close();
  }

  #guard(): void {
    if (this.#closed) throw ErrClosed;
  }

  async get(key: string, signal?: AbortSignal): Promise<{ entry: Entry; hit: boolean }> {
    if (key === "") throw ErrEmptyKey;
    this.#guard();
    let raw: JsonRaw;
    try {
      raw = await this.#backend.get(key, signal);
    } catch (e) {
      if (isNotFound(e) || e === ErrNotFound) {
        return { entry: emptyEntry(), hit: false };
      }
      throw e;
    }
    const decoded = decodeEntry(raw);
    if (!decoded.ok) {
      await this.#deleteIgnoreMissing(key, signal);
      return { entry: emptyEntry(), hit: false };
    }
    if (isExpired(decoded.expiresAt)) {
      await this.#deleteIgnoreMissing(key, signal);
      return { entry: emptyEntry(), hit: false };
    }
    return {
      entry: {
        value: decoded.value,
        createdAt: new Date(0),
        expiresAt: decoded.expiresAt,
      },
      hit: true,
    };
  }

  async set(key: string, entry: Entry, ttl: DurationMs, signal?: AbortSignal): Promise<void> {
    if (key === "") throw ErrEmptyKey;
    if (ttl < 0) throw ErrNegativeTTL;
    this.#guard();
    let exp = entry.expiresAt;
    if (ttl > 0) {
      exp = new Date(Date.now() + ttl);
    } else if (ttl === 0 && (entry.expiresAt == null || entry.expiresAt.getTime() === 0)) {
      exp = null;
    }
    const raw = encodeEntry(entry.value, exp);
    await this.#backend.set(key, raw, signal);
  }

  async delete(key: string, signal?: AbortSignal): Promise<void> {
    if (key === "") throw ErrEmptyKey;
    this.#guard();
    await this.#deleteIgnoreMissing(key, signal);
  }

  async #deleteIgnoreMissing(key: string, signal?: AbortSignal): Promise<void> {
    try {
      await this.#backend.delete(key, signal);
    } catch (e) {
      if (isNotFound(e) || e === ErrNotFound) return;
      throw e;
    }
  }

  async incr(key: string, signal?: AbortSignal): Promise<number> {
    if (key === "") throw ErrEmptyKey;
    this.#guard();
    let last: unknown;
    for (let attempt = 0; attempt < incrMaxAttempts; attempt++) {
      try {
        return await this.#incrOnce(key, signal);
      } catch (e) {
        last = e;
        if (e === ErrNotInteger || e === ErrIncrOverflow) throw e;
        if (e !== ErrNestedTx) throw e;
        await sleep((attempt + 1) * 5);
      }
    }
    throw last;
  }

  async #incrOnce(key: string, signal?: AbortSignal): Promise<number> {
    const tx = await this.#backend.beginTx(signal);
    try {
      let n = 0;
      try {
        const raw = await tx.get(key, signal);
        const decoded = decodeEntry(raw);
        if (!decoded.ok) throw ErrNotInteger;
        const parsed = asInt64(decoded.value);
        if (parsed == null) throw ErrNotInteger;
        n = parsed;
      } catch (e) {
        if (!(isNotFound(e) || e === ErrNotFound)) throw e;
        n = 0;
      }
      if (n === Number.MAX_SAFE_INTEGER) throw ErrIncrOverflow;
      n += 1;
      await tx.set(key, encodeEntry(n, null), signal);
      await tx.commit();
      return n;
    } catch (e) {
      try {
        await tx.rollback();
      } catch {
        /* ignore */
      }
      throw e;
    }
  }

  async purgeExact(logicalPrefix: string, signal?: AbortSignal): Promise<void> {
    if (logicalPrefix === "") throw ErrEmptyKey;
    this.#guard();
    const vk = `${logicalPrefix}:__version__`;
    let max = 0;
    try {
      const raw = await this.#backend.get(vk, signal);
      const decoded = decodeEntry(raw);
      if (decoded.ok) {
        const n = asInt64(decoded.value);
        if (n != null) max = n;
      }
    } catch (e) {
      if (!(isNotFound(e) || e === ErrNotFound)) throw e;
    }
    for (let i = 1; i <= max; i++) {
      const k = `${logicalPrefix}:${i}`;
      if (!isVersionDataKey(logicalPrefix, k)) continue;
      await this.#deleteIgnoreMissing(k, signal);
    }
  }

  async purgePrefix(prefix: string, signal?: AbortSignal): Promise<void> {
    if (prefix === "") throw ErrEmptyKey;
    this.#guard();
    const keys = await this.#backend.scan(prefix, signal);
    for (const k of keys) {
      await this.#deleteIgnoreMissing(k, signal);
    }
  }
}

class HttpBackend implements Es4Backend {
  private constructor(
    private readonly base: string,
    private readonly fetchImpl: typeof fetch,
  ) {}

  static async connect(opts: Es4HTTPOptions): Promise<HttpBackend> {
    const base = opts.baseUrl.replace(/\/$/, "");
    const fetchImpl = opts.fetchImpl ?? fetch;
    const res = await fetchImpl(`${base}/readyz`);
    if (res.status === 503) throw new Error("sscachian/es4: server not ready (503)");
    if (!res.ok) throw new Error(`sscachian/es4: readyz status ${res.status}`);
    return new HttpBackend(base, fetchImpl);
  }

  private keyUrl(key: string): string {
    return `${this.base}/v1/keys/${encodeURIComponent(key)}`;
  }

  async set(key: string, value: JsonRaw, signal?: AbortSignal): Promise<void> {
    const body = typeof value === "string" ? value : Buffer.from(value).toString("utf8");
    const res = await this.fetchImpl(this.keyUrl(key), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body,
      signal,
    });
    await mapHttp(res);
  }

  async get(key: string, signal?: AbortSignal): Promise<JsonRaw> {
    const res = await this.fetchImpl(this.keyUrl(key), { method: "GET", signal });
    if (res.status === 404) throw ErrNotFound;
    await mapHttp(res);
    return await res.text();
  }

  async delete(key: string, signal?: AbortSignal): Promise<void> {
    const res = await this.fetchImpl(this.keyUrl(key), { method: "DELETE", signal });
    if (res.status === 404) throw ErrNotFound;
    await mapHttp(res);
  }

  async scan(prefix: string, signal?: AbortSignal): Promise<string[]> {
    const url = `${this.base}/v1/scan?prefix=${encodeURIComponent(prefix)}`;
    const res = await this.fetchImpl(url, { method: "GET", signal });
    await mapHttp(res);
    const keys = (await res.json()) as string[];
    return keys ?? [];
  }

  async beginTx(signal?: AbortSignal): Promise<Tx> {
    const res = await this.fetchImpl(`${this.base}/v1/tx`, { method: "POST", signal });
    await mapHttp(res);
    const { id } = (await res.json()) as { id: string };
    if (!id) throw new Error("sscachian/es4: empty tx id");
    return new HttpTx(this.base, this.fetchImpl, id);
  }

  async close(): Promise<void> {
    /* no-op */
  }
}

class HttpTx implements Tx {
  #done = false;
  constructor(
    private readonly base: string,
    private readonly fetchImpl: typeof fetch,
    private readonly id: string,
  ) {}

  private keyUrl(key: string): string {
    return `${this.base}/v1/tx/${encodeURIComponent(this.id)}/keys/${encodeURIComponent(key)}`;
  }

  async set(key: string, value: JsonRaw, signal?: AbortSignal): Promise<void> {
    const body = typeof value === "string" ? value : Buffer.from(value).toString("utf8");
    const res = await this.fetchImpl(this.keyUrl(key), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body,
      signal,
    });
    await mapHttp(res);
  }

  async get(key: string, signal?: AbortSignal): Promise<JsonRaw> {
    const res = await this.fetchImpl(this.keyUrl(key), { method: "GET", signal });
    if (res.status === 404) throw ErrNotFound;
    await mapHttp(res);
    return await res.text();
  }

  async delete(key: string, signal?: AbortSignal): Promise<void> {
    const res = await this.fetchImpl(this.keyUrl(key), { method: "DELETE", signal });
    if (res.status === 404) throw ErrNotFound;
    await mapHttp(res);
  }

  async exists(key: string, signal?: AbortSignal): Promise<boolean> {
    const res = await this.fetchImpl(this.keyUrl(key), { method: "HEAD", signal });
    if (res.status === 404) return false;
    await mapHttp(res);
    return true;
  }

  async scan(prefix: string, signal?: AbortSignal): Promise<string[]> {
    // Provisional until Es4 Server exposes Tx Scan; use non-Tx /v1/scan.
    const url = `${this.base}/v1/scan?prefix=${encodeURIComponent(prefix)}`;
    const res = await this.fetchImpl(url, { method: "GET", signal });
    await mapHttp(res);
    const keys = (await res.json()) as string[];
    return keys ?? [];
  }

  async clear(signal?: AbortSignal): Promise<void> {
    const res = await this.fetchImpl(`${this.base}/v1/tx/${encodeURIComponent(this.id)}/clear`, {
      method: "POST",
      signal,
    });
    await mapHttp(res);
  }

  async commit(): Promise<void> {
    if (this.#done) throw new Error("state: transaction finished");
    const res = await this.fetchImpl(`${this.base}/v1/tx/${encodeURIComponent(this.id)}/commit`, {
      method: "POST",
    });
    await mapHttp(res);
    this.#done = true;
  }

  async rollback(): Promise<void> {
    if (this.#done) throw new Error("state: transaction finished");
    const res = await this.fetchImpl(`${this.base}/v1/tx/${encodeURIComponent(this.id)}/rollback`, {
      method: "POST",
    });
    await mapHttp(res);
    this.#done = true;
  }
}

async function mapHttp(res: Response): Promise<void> {
  if (res.ok) return;
  if (res.status === 404) throw ErrNotFound;
  if (res.status === 503) throw new Error("sscachian/es4: server not ready (503)");
  const text = await res.text();
  throw new Error(`sscachian/es4: http ${res.status}: ${text}`);
}

function emptyEntry(): Entry {
  return { value: null, createdAt: new Date(0), expiresAt: null };
}

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

export async function newEs4Store(opts?: Es4DriverOptions): Promise<Es4Store> {
  return Es4Store.create(opts);
}

export async function newEs4HTTPStore(opts: Es4HTTPOptions): Promise<Es4Store> {
  return Es4Store.createHTTP(opts);
}
