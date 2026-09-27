import {
  type DurationMs,
  type Entry,
  type KeyBuilder,
  type KeyContext,
  type Layer,
  type Loader,
  type TypeGuard,
  ErrCorruptVersion,
  ErrInvalidLayerIndex,
  ErrInvalidVersion,
  ErrNoKeyBuilder,
  ErrNoLayer,
  ErrNoLoader,
  ErrNegativeTTL,
  ErrTypeMismatch,
  defaultKeyBuilder,
} from "./types.js";

function logWarn(msg: string, err: unknown): void {
  const detail = err instanceof Error ? err.message : String(err);
  console.warn(msg, detail);
}

function asInt64(v: unknown): number {
  if (typeof v === "number") {
    if (!Number.isInteger(v)) throw ErrCorruptVersion;
    return v;
  }
  if (typeof v === "bigint") {
    const n = Number(v);
    if (!Number.isSafeInteger(n)) throw ErrCorruptVersion;
    return n;
  }
  throw ErrCorruptVersion;
}

function castValue<T>(v: unknown, guard?: TypeGuard<T>): T {
  if (guard) {
    if (!guard(v)) throw ErrTypeMismatch;
    return v;
  }
  return v as T;
}

export class Builder<T> {
  readonly #name: string;
  #layers: Layer[] = [];
  #ttls: DurationMs[] = [];
  #single: DurationMs | null = null;
  #kb: KeyBuilder | null = defaultKeyBuilder;
  #loader: Loader<T> | null = null;
  #guard: TypeGuard<T> | undefined;

  constructor(name: string) {
    this.#name = name;
  }

  withLayers(...layers: Layer[]): this {
    this.#layers = [...layers];
    return this;
  }

  withKeyBuilder(kb: KeyBuilder | null): this {
    this.#kb = kb;
    return this;
  }

  withLayerTTL(ttl: DurationMs): this {
    this.#single = ttl;
    return this;
  }

  withPolicy(ttl: DurationMs): this {
    return this.withLayerTTL(ttl);
  }

  withLayerTTLs(...ttls: DurationMs[]): this {
    this.#ttls = [...ttls];
    this.#single = null;
    return this;
  }

  withLoader(loader: Loader<T>): this {
    this.#loader = loader;
    return this;
  }

  /** Optional runtime type check on Get (Go type assertion equivalent). */
  withTypeGuard(guard: TypeGuard<T>): this {
    this.#guard = guard;
    return this;
  }

  build(): CacheType<T> {
    if (this.#layers.length === 0) throw ErrNoLayer;
    if (this.#kb == null) throw ErrNoKeyBuilder;
    const ttls: DurationMs[] = new Array(this.#layers.length).fill(0);
    if (this.#single != null) {
      if (this.#single < 0) throw ErrNegativeTTL;
      for (let i = 0; i < ttls.length; i++) ttls[i] = this.#single;
    } else {
      for (let i = 0; i < ttls.length; i++) {
        if (i < this.#ttls.length) {
          const t = this.#ttls[i]!;
          if (t < 0) throw ErrNegativeTTL;
          ttls[i] = t;
        }
      }
    }
    return new CacheType(
      this.#name,
      [...this.#layers],
      ttls,
      this.#kb,
      this.#loader,
      this.#guard,
    );
  }
}

export function define<T = unknown>(name: string): Builder<T> {
  return new Builder<T>(name);
}

export class CacheType<T> {
  readonly #name: string;
  readonly #layers: Layer[];
  readonly #ttls: DurationMs[];
  readonly #kb: KeyBuilder;
  readonly #loader: Loader<T> | null;
  readonly #guard: TypeGuard<T> | undefined;

  constructor(
    name: string,
    layers: Layer[],
    ttls: DurationMs[],
    kb: KeyBuilder,
    loader: Loader<T> | null,
    guard?: TypeGuard<T>,
  ) {
    this.#name = name;
    this.#layers = layers;
    this.#ttls = ttls;
    this.#kb = kb;
    this.#loader = loader;
    this.#guard = guard;
  }

  name(): string {
    return this.#name;
  }

  layers(): Layer[] {
    return [...this.#layers];
  }

  layerTTL(i: number): DurationMs {
    if (i < 0 || i >= this.#ttls.length) return 0;
    return this.#ttls[i]!;
  }

  #l1(): Layer {
    return this.#layers[0]!;
  }

  async #logicalPrefix(kc: KeyContext, signal?: AbortSignal): Promise<string> {
    return await this.#kb(kc, signal);
  }

  async buildKey(kc: KeyContext, version: number, signal?: AbortSignal): Promise<string> {
    if (version < 1) throw ErrInvalidVersion;
    const prefix = await this.#logicalPrefix(kc, signal);
    return `${prefix}:${version}`;
  }

  async versionKey(kc: KeyContext, signal?: AbortSignal): Promise<string> {
    const prefix = await this.#logicalPrefix(kc, signal);
    return `${prefix}:__version__`;
  }

  async #ensureCurrentVersion(kc: KeyContext, signal?: AbortSignal): Promise<number> {
    const vk = await this.versionKey(kc, signal);
    const { entry, hit } = await this.#l1().get(vk, signal);
    if (hit) return asInt64(entry.value);
    await this.#l1().set(vk, { value: 1, createdAt: new Date(0), expiresAt: null }, 0, signal);
    return 1;
  }

  currentVersion(kc: KeyContext, signal?: AbortSignal): Promise<number> {
    return this.#ensureCurrentVersion(kc, signal);
  }

  async bumpVersion(kc: KeyContext, signal?: AbortSignal): Promise<number> {
    const vk = await this.versionKey(kc, signal);
    await this.#ensureCurrentVersion(kc, signal);
    return this.#l1().incr(vk, signal);
  }

  async buildLatestKey(kc: KeyContext, signal?: AbortSignal): Promise<string> {
    const v = await this.#ensureCurrentVersion(kc, signal);
    return this.buildKey(kc, v, signal);
  }

  async #writeBack(key: string, e: Entry, upToExclusive: number, signal?: AbortSignal): Promise<void> {
    for (let j = 0; j < upToExclusive; j++) {
      try {
        await this.#layers[j]!.set(key, e, this.layerTTL(j), signal);
      } catch (err) {
        logWarn(`sscachian: write-back to layer ${j} failed:`, err);
      }
    }
  }

  async #writeAll(key: string, e: Entry, signal?: AbortSignal): Promise<void> {
    for (let i = 0; i < this.#layers.length; i++) {
      try {
        await this.#layers[i]!.set(key, e, this.layerTTL(i), signal);
      } catch (err) {
        logWarn(`sscachian: write-back to layer ${i} failed:`, err);
      }
    }
  }

  async get(
    kc: KeyContext,
    signal?: AbortSignal,
  ): Promise<{ value: T; ok: true } | { value?: undefined; ok: false }> {
    const key = await this.buildLatestKey(kc, signal);
    for (let i = 0; i < this.#layers.length; i++) {
      const { entry, hit } = await this.#layers[i]!.get(key, signal);
      if (!hit) continue;
      await this.#writeBack(key, entry, i, signal);
      const tv = castValue<T>(entry.value, this.#guard);
      return { value: tv, ok: true };
    }
    return { ok: false };
  }

  async set(kc: KeyContext, value: T, signal?: AbortSignal): Promise<void> {
    await this.#ensureCurrentVersion(kc, signal);
    const newV = await this.bumpVersion(kc, signal);
    const key = await this.buildKey(kc, newV, signal);
    const entry: Entry = { value, createdAt: new Date(0), expiresAt: null };
    await this.#l1().set(key, entry, this.layerTTL(0), signal);
    for (let i = 1; i < this.#layers.length; i++) {
      try {
        await this.#layers[i]!.set(key, entry, this.layerTTL(i), signal);
      } catch (err) {
        logWarn(`sscachian: set layer ${i} failed:`, err);
      }
    }
  }

  async delete(kc: KeyContext, signal?: AbortSignal): Promise<void> {
    const v = await this.#ensureCurrentVersion(kc, signal);
    const key = await this.buildKey(kc, v, signal);
    await this.#l1().delete(key, signal);
    for (let i = 1; i < this.#layers.length; i++) {
      try {
        await this.#layers[i]!.delete(key, signal);
      } catch (err) {
        logWarn(`sscachian: delete layer ${i} failed:`, err);
      }
    }
    await this.bumpVersion(kc, signal);
  }

  async purge(kc: KeyContext, layerIdx: number[] = [], signal?: AbortSignal): Promise<void> {
    const prefix = await this.#logicalPrefix(kc, signal);
    const targets = this.#resolveLayerIndexes(layerIdx);
    const includesL1 = targets.includes(0);
    for (const i of targets) {
      try {
        await this.#layers[i]!.purgeExact(prefix, signal);
      } catch (err) {
        if (includesL1) {
          if (i === 0) throw err;
          logWarn(`sscachian: purge layer ${i} failed:`, err);
          continue;
        }
        throw err;
      }
    }
  }

  #resolveLayerIndexes(layerIdx: number[]): number[] {
    const n = this.#layers.length;
    if (layerIdx.length === 0) {
      return Array.from({ length: n }, (_, i) => i);
    }
    const seen = new Set<number>();
    const out: number[] = [];
    for (const i of layerIdx) {
      if (i < 0 || i >= n) throw ErrInvalidLayerIndex;
      if (seen.has(i)) continue;
      seen.add(i);
      out.push(i);
    }
    out.sort((a, b) => a - b);
    return out;
  }

  async getOrLoad(kc: KeyContext, signal?: AbortSignal): Promise<T> {
    const got = await this.get(kc, signal);
    if (got.ok) return got.value;
    if (this.#loader == null) throw ErrNoLoader;
    const loaded = await this.#loader(kc, signal);
    const ver = await this.#ensureCurrentVersion(kc, signal);
    const key = await this.buildKey(kc, ver, signal);
    await this.#writeAll(key, { value: loaded, createdAt: new Date(0), expiresAt: null }, signal);
    return loaded;
  }
}
