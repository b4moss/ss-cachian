import { Firestore, type DocumentData } from "@google-cloud/firestore";

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

const defaultCollection = "sscachian";

export type FirestoreOptions = {
  projectId?: string;
  collection?: string;
};

type DocFields = {
  value: Buffer | Uint8Array | string;
  created_at: Date;
  expires_at: Date | null;
};

export class FirestoreStore implements Layer {
  readonly #client: Firestore;
  readonly #collection: string;

  constructor(client: Firestore, collection: string) {
    this.#client = client;
    this.#collection = collection;
  }

  static async create(opts: FirestoreOptions = {}): Promise<FirestoreStore> {
    const projectId = opts.projectId || "ss-cachian-dev";
    const collection = opts.collection || defaultCollection;
    const client = new Firestore({ projectId });
    return new FirestoreStore(client, collection);
  }

  async close(): Promise<void> {
    await this.#client.terminate();
  }

  #doc(key: string) {
    return this.#client.collection(this.#collection).doc(key);
  }

  async get(key: string, _signal?: AbortSignal): Promise<{ entry: Entry; hit: boolean }> {
    if (key === "") throw ErrEmptyKey;
    const snap = await this.#doc(key).get();
    if (!snap.exists) return { entry: emptyEntry(), hit: false };
    const f = snap.data() as DocFields;
    const expiresAt = toDateOrNull(f.expires_at);
    if (isExpired(expiresAt)) return { entry: emptyEntry(), hit: false };
    let val: unknown = null;
    const raw = f.value;
    if (raw != null && (typeof raw === "string" ? raw.length > 0 : raw.length > 0)) {
      const text = typeof raw === "string" ? raw : Buffer.from(raw).toString("utf8");
      try {
        val = JSON.parse(text) as unknown;
      } catch {
        throw new Error("sscachian/firestore: invalid JSON value");
      }
    }
    return {
      entry: {
        value: val,
        createdAt: toDateOrNull(f.created_at) ?? new Date(0),
        expiresAt,
      },
      hit: true,
    };
  }

  async set(key: string, entry: Entry, ttl: DurationMs, _signal?: AbortSignal): Promise<void> {
    if (key === "") throw ErrEmptyKey;
    if (ttl < 0) throw ErrNegativeTTL;
    const now = new Date();
    let createdAt = entry.createdAt.getTime() === 0 ? now : entry.createdAt;
    let expiresAt = entry.expiresAt;
    if (ttl > 0) {
      expiresAt = new Date(now.getTime() + ttl);
    }
    let raw: string;
    try {
      raw = JSON.stringify(entry.value === undefined ? null : entry.value);
    } catch (err) {
      throw new Error(`sscachian/firestore: marshal value: ${String(err)}`);
    }
    const data: DocumentData = {
      value: Buffer.from(raw, "utf8"),
      created_at: createdAt,
      expires_at: expiresAt ?? new Date(0),
    };
    await this.#doc(key).set(data);
  }

  async delete(key: string, _signal?: AbortSignal): Promise<void> {
    if (key === "") throw ErrEmptyKey;
    try {
      await this.#doc(key).delete();
    } catch {
      // missing is ok
    }
  }

  async incr(key: string, _signal?: AbortSignal): Promise<number> {
    if (key === "") throw ErrEmptyKey;
    const ref = this.#doc(key);
    let last: unknown;
    // Emulator aborts under contention; keep trying with backoff (Go parity).
    for (let attempt = 0; attempt < 32; attempt++) {
      try {
        const out = await this.#client.runTransaction(async (tx) => {
          const snap = await tx.get(ref);
          let n = 0;
          if (snap.exists) {
            const f = snap.data() as DocFields;
            const text =
              typeof f.value === "string"
                ? f.value
                : Buffer.from(f.value ?? []).toString("utf8");
            let val: unknown;
            try {
              val = JSON.parse(text) as unknown;
            } catch {
              throw ErrNotInteger;
            }
            n = asIncrInt(val);
          }
          if (n >= Number.MAX_SAFE_INTEGER) throw ErrIncrOverflow;
          n += 1;
          const raw = JSON.stringify(n);
          tx.set(ref, {
            value: Buffer.from(raw, "utf8"),
            created_at: new Date(),
            expires_at: new Date(0),
          });
          return n;
        });
        return out;
      } catch (err) {
        last = err;
        if (err === ErrNotInteger || err === ErrIncrOverflow) throw err;
        if (!isAbortedTxn(err)) throw err;
        await sleep(Math.min(50, (attempt + 1) * 5));
      }
    }
    throw last instanceof Error ? last : new Error(String(last));
  }

  async purgeExact(logicalPrefix: string, _signal?: AbortSignal): Promise<void> {
    if (logicalPrefix === "") throw ErrEmptyKey;
    const snap = await this.#client.collection(this.#collection).get();
    for (const doc of snap.docs) {
      if (!isVersionDataKey(logicalPrefix, doc.id)) continue;
      await doc.ref.delete();
    }
  }

  /** Tests only. */
  async clearCollection(): Promise<void> {
    const snap = await this.#client.collection(this.#collection).get();
    for (const doc of snap.docs) {
      await doc.ref.delete();
    }
  }
}

export async function newFirestoreStore(opts?: FirestoreOptions): Promise<FirestoreStore> {
  return FirestoreStore.create(opts);
}

function emptyEntry(): Entry {
  return { value: undefined, createdAt: new Date(0), expiresAt: null };
}

function toDateOrNull(v: unknown): Date | null {
  if (v == null) return null;
  if (v instanceof Date) {
    if (v.getTime() === 0) return null;
    return v;
  }
  if (typeof v === "object" && v !== null && "toDate" in v && typeof (v as { toDate: () => Date }).toDate === "function") {
    const d = (v as { toDate: () => Date }).toDate();
    if (d.getTime() === 0) return null;
    return d;
  }
  return null;
}

function asIncrInt(v: unknown): number {
  if (typeof v === "number") {
    if (!Number.isInteger(v)) throw ErrNotInteger;
    return v;
  }
  throw ErrNotInteger;
}

function isAbortedTxn(err: unknown): boolean {
  const code = (err as { code?: number | string }).code;
  if (code === 10 || code === "ABORTED") return true;
  if (err instanceof Error) {
    return /ABORTED|aborted|Transaction lock timeout/i.test(err.message);
  }
  return false;
}

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}
