import { promises as fs } from "node:fs";
import path from "node:path";
import { parse as parseYaml } from "yaml";

import { CacheType, define } from "./cache.js";
import { newFirestoreStore } from "./driver/firestore.js";
import { newMemoryStore } from "./driver/memory.js";
import {
  ErrInvalidConfig,
  ErrNegativeTTL,
  defaultKeyBuilder,
  type DurationMs,
  type KeyBuilder,
  type Layer,
  type Loader,
} from "./types.js";

const SCHEMA_VERSION = "0.9";

export type Registry = {
  registerKeyBuilder(name: string, kb: KeyBuilder): void;
  registerLoader<T>(name: string, loader: Loader<T>): void;
  /** @internal */
  lookupKeyBuilder(name: string): KeyBuilder | undefined;
  /** @internal */
  lookupLoader(name: string): Loader<unknown> | undefined;
};

export function createRegistry(): Registry {
  const keyBuilders = new Map<string, KeyBuilder>([["default", defaultKeyBuilder]]);
  const loaders = new Map<string, Loader<unknown>>();

  return {
    registerKeyBuilder(name: string, kb: KeyBuilder): void {
      if (!name) throw new Error(`${ErrInvalidConfig.message}: empty key builder name`);
      if (kb == null) throw new Error(`${ErrInvalidConfig.message}: nil key builder`);
      keyBuilders.set(name, kb);
    },
    registerLoader<T>(name: string, loader: Loader<T>): void {
      if (!name) throw new Error(`${ErrInvalidConfig.message}: empty loader name`);
      if (loader == null) throw new Error(`${ErrInvalidConfig.message}: nil loader`);
      loaders.set(name, loader as Loader<unknown>);
    },
    lookupKeyBuilder(name: string): KeyBuilder | undefined {
      return keyBuilders.get(name);
    },
    lookupLoader(name: string): Loader<unknown> | undefined {
      return loaders.get(name);
    },
  };
}

type RawFile = {
  schema_version?: unknown;
  types?: RawType[];
};

type RawType = {
  name?: unknown;
  key_builder?: unknown;
  loader?: unknown;
  ttl?: unknown;
  layers?: RawLayer[];
};

type RawLayer = {
  driver?: unknown;
  ttl?: unknown;
  options?: unknown;
};

/** Go time.ParseDuration-compatible parser (ns/us/ms/s/m/h). */
export function parseDuration(s: string): DurationMs {
  if (s === "" || s === "0") return 0;
  const re = /(-?\d+(?:\.\d+)?)(ns|us|µs|μs|ms|s|m|h)/g;
  let total = 0;
  let matched = 0;
  let m: RegExpExecArray | null;
  const units: Record<string, number> = {
    ns: 1e-6,
    us: 0.001,
    µs: 0.001,
    μs: 0.001,
    ms: 1,
    s: 1000,
    m: 60_000,
    h: 3_600_000,
  };
  while ((m = re.exec(s)) !== null) {
    matched += m[0].length;
    const n = Number(m[1]);
    const u = m[2]!;
    total += n * (units[u] ?? 0);
  }
  if (matched !== s.length || Number.isNaN(total)) {
    throw new Error(`${ErrInvalidConfig.message}: invalid ttl ${JSON.stringify(s)}`);
  }
  if (total < 0) throw ErrNegativeTTL;
  return total;
}

function optionalTTL(v: unknown): { set: boolean; ms: DurationMs } {
  if (v == null) return { set: false, ms: 0 };
  if (typeof v !== "string") {
    throw new Error(`${ErrInvalidConfig.message}: ttl must be a string duration`);
  }
  if (v === "") return { set: false, ms: 0 };
  return { set: true, ms: parseDuration(v) };
}

function schemaVersion(v: unknown): string {
  if (v == null || v === "") {
    throw new Error(`${ErrInvalidConfig.message}: schema_version is required`);
  }
  if (typeof v !== "string") {
    throw new Error(`${ErrInvalidConfig.message}: schema_version must be a string`);
  }
  return v;
}

function stringOptions(opts: unknown): Record<string, string> {
  if (opts == null) return {};
  if (typeof opts !== "object" || Array.isArray(opts)) {
    throw new Error(`${ErrInvalidConfig.message}: options must be a map`);
  }
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(opts as Record<string, unknown>)) {
    if (typeof v !== "string") {
      throw new Error(`${ErrInvalidConfig.message}: option ${JSON.stringify(k)} must be a string`);
    }
    out[k] = v;
  }
  return out;
}

async function buildLayer(rl: RawLayer): Promise<Layer> {
  const driver = rl.driver;
  if (typeof driver !== "string" || driver === "") {
    throw new Error(`${ErrInvalidConfig.message}: driver must not be empty`);
  }
  const opts = stringOptions(rl.options);
  if (driver === "memory") {
    for (const k of Object.keys(opts)) {
      throw new Error(`${ErrInvalidConfig.message}: unknown memory option ${JSON.stringify(k)}`);
    }
    return newMemoryStore();
  }
  if (driver === "firestore") {
    const projectId = opts.project_id;
    const collection = opts.collection;
    for (const k of Object.keys(opts)) {
      if (k !== "project_id" && k !== "collection") {
        throw new Error(`${ErrInvalidConfig.message}: unknown firestore option ${JSON.stringify(k)}`);
      }
    }
    return newFirestoreStore({ projectId, collection });
  }
  throw new Error(`${ErrInvalidConfig.message}: unknown driver ${JSON.stringify(driver)}`);
}

async function buildOneType<T>(reg: Registry, rt: RawType): Promise<CacheType<T>> {
  if (typeof rt.name !== "string" || rt.name === "") {
    throw new Error(`${ErrInvalidConfig.message}: type name must not be empty`);
  }
  if (!Array.isArray(rt.layers) || rt.layers.length === 0) {
    throw new Error(`${ErrInvalidConfig.message}: type ${JSON.stringify(rt.name)}: layers must not be empty`);
  }

  const kbName =
    rt.key_builder == null || rt.key_builder === ""
      ? "default"
      : typeof rt.key_builder === "string"
        ? rt.key_builder
        : null;
  if (kbName == null) {
    throw new Error(`${ErrInvalidConfig.message}: key_builder must be a string`);
  }
  const kb = reg.lookupKeyBuilder(kbName);
  if (!kb) {
    throw new Error(`${ErrInvalidConfig.message}: unknown key_builder ${JSON.stringify(kbName)}`);
  }

  let loader: Loader<T> | undefined;
  if (rt.loader != null && rt.loader !== "") {
    if (typeof rt.loader !== "string") {
      throw new Error(`${ErrInvalidConfig.message}: loader must be a string`);
    }
    const found = reg.lookupLoader(rt.loader);
    if (!found) {
      throw new Error(`${ErrInvalidConfig.message}: unknown loader ${JSON.stringify(rt.loader)}`);
    }
    loader = found as Loader<T>;
  }

  const typeTTL = optionalTTL(rt.ttl);
  const layers: Layer[] = [];
  const ttls: DurationMs[] = [];
  for (let i = 0; i < rt.layers.length; i++) {
    const rl = rt.layers[i]!;
    layers.push(await buildLayer(rl));
    const layerTTL = optionalTTL(rl.ttl);
    if (layerTTL.set) ttls.push(layerTTL.ms);
    else if (typeTTL.set) ttls.push(typeTTL.ms);
    else ttls.push(0);
  }

  const b = define<T>(rt.name).withLayers(...layers).withKeyBuilder(kb).withLayerTTLs(...ttls);
  if (loader) b.withLoader(loader);
  return b.build();
}

async function buildTypes<T>(reg: Registry, raw: RawFile): Promise<Map<string, CacheType<T>>> {
  const ver = schemaVersion(raw.schema_version);
  if (ver !== SCHEMA_VERSION) {
    throw new Error(`${ErrInvalidConfig.message}: unsupported schema_version ${JSON.stringify(ver)}`);
  }
  if (!Array.isArray(raw.types) || raw.types.length === 0) {
    throw new Error(`${ErrInvalidConfig.message}: types must not be empty`);
  }
  const out = new Map<string, CacheType<T>>();
  for (const rt of raw.types) {
    const name = typeof rt.name === "string" ? rt.name : "";
    if (out.has(name)) {
      throw new Error(`${ErrInvalidConfig.message}: duplicate type name ${JSON.stringify(name)}`);
    }
    const ct = await buildOneType<T>(reg, rt);
    out.set(ct.name(), ct);
  }
  return out;
}

export async function loadTypesYAML<T = unknown>(
  reg: Registry,
  data: string | Uint8Array,
): Promise<Map<string, CacheType<T>>> {
  const text = typeof data === "string" ? data : Buffer.from(data).toString("utf8");
  let raw: RawFile;
  try {
    raw = parseYaml(text) as RawFile;
  } catch (e) {
    throw new Error(`${ErrInvalidConfig.message}: yaml: ${e instanceof Error ? e.message : e}`);
  }
  return buildTypes<T>(reg, raw ?? {});
}

export async function loadTypesJSON<T = unknown>(
  reg: Registry,
  data: string | Uint8Array,
): Promise<Map<string, CacheType<T>>> {
  const text = typeof data === "string" ? data : Buffer.from(data).toString("utf8");
  let raw: RawFile;
  try {
    raw = JSON.parse(text) as RawFile;
  } catch (e) {
    throw new Error(`${ErrInvalidConfig.message}: json: ${e instanceof Error ? e.message : e}`);
  }
  return buildTypes<T>(reg, raw ?? {});
}

export async function loadTypes<T = unknown>(
  reg: Registry,
  filePath: string,
): Promise<Map<string, CacheType<T>>> {
  const data = await fs.readFile(filePath);
  const ext = path.extname(filePath).toLowerCase();
  if (ext === ".json") return loadTypesJSON<T>(reg, data);
  if (ext === ".yaml" || ext === ".yml") return loadTypesYAML<T>(reg, data);
  throw new Error(`${ErrInvalidConfig.message}: unsupported config extension ${JSON.stringify(ext)}`);
}
