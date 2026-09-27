import assert from "node:assert/strict";
import test from "node:test";

import {
  CacheType,
  ErrInvalidContext,
  ErrInvalidVersion,
  ErrNoKeyBuilder,
  ErrNoLayer,
  ErrNoLoader,
  ErrTypeMismatch,
  defaultKeyBuilder,
  define,
  newMemoryStore,
  type KeyContext,
} from "./index.js";

function sampleKC(): KeyContext {
  return { appSlug: "my-app", tenantId: "0123456", queryType: "client_list_page1" };
}

function newStringCache(
  opts?: (b: ReturnType<typeof define<string>>) => void,
): CacheType<string> {
  const b = define<string>("test")
    .withLayers(newMemoryStore())
    .withKeyBuilder(defaultKeyBuilder)
    .withTypeGuard((v): v is string => typeof v === "string");
  opts?.(b);
  return b.build();
}

test("build ok", () => {
  const ct = newStringCache();
  assert.equal(ct.name(), "test");
});

test("build with ttl and loader", async () => {
  let loads = 0;
  const ct = newStringCache((b) => {
    b.withLayerTTL(3_600_000).withLoader(async () => {
      loads += 1;
      return "loaded";
    });
  });
  const kc = sampleKC();
  const v = await ct.getOrLoad(kc);
  assert.equal(v, "loaded");
  assert.equal(loads, 1);
  await ct.set(kc, "x");
  const got = await ct.get(kc);
  assert.equal(got.ok, true);
  if (got.ok) assert.equal(got.value, "x");
});

test("build errors", () => {
  assert.throws(() => define("t").withKeyBuilder(defaultKeyBuilder).build(), ErrNoLayer);
  assert.throws(
    () => define("t").withLayers(newMemoryStore()).withKeyBuilder(null).build(),
    ErrNoKeyBuilder,
  );
  assert.throws(() => define("t").withLayers().withKeyBuilder(defaultKeyBuilder).build(), ErrNoLayer);
});

test("buildKey and versionKey", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  assert.equal(await ct.buildKey(kc, 7), "my-app:cache:0123456:client_list_page1:7");
  assert.equal(await ct.versionKey(kc), "my-app:cache:0123456:client_list_page1:__version__");
});

test("buildLatestKey matches current", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  const v = await ct.currentVersion(kc);
  const k = await ct.buildLatestKey(kc);
  assert.equal(k, await ct.buildKey(kc, v));
});

test("buildKey errors", async () => {
  const ct = newStringCache();
  await assert.rejects(() => ct.buildKey({ appSlug: "", tenantId: "", queryType: "" }, 1), ErrInvalidContext);
  await assert.rejects(() => ct.buildKey(sampleKC(), 0), ErrInvalidVersion);
  const ct2 = newStringCache((b) => {
    b.withKeyBuilder(async () => {
      throw new Error("kb boom");
    });
  });
  await assert.rejects(() => ct2.buildKey(sampleKC(), 1), /kb boom/);
});

test("currentVersion init and read", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  assert.equal(await ct.currentVersion(kc), 1);
  assert.equal(await ct.currentVersion(kc), 1);
});

test("bumpVersion advances and keeps data", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  await ct.set(kc, "a");
  const before = await ct.currentVersion(kc);
  const after = await ct.bumpVersion(kc);
  assert.equal(after, before + 1);
  // old version key still present on L1 for previous data — Get uses latest
  const got = await ct.get(kc);
  assert.equal(got.ok, false);
});

test("get set round trip", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  await ct.set(kc, "hello");
  const got = await ct.get(kc);
  assert.equal(got.ok, true);
  if (got.ok) assert.equal(got.value, "hello");
});

test("first get inits version and misses", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  const got = await ct.get(kc);
  assert.equal(got.ok, false);
  assert.equal(await ct.currentVersion(kc), 1);
});

test("get type mismatch", async () => {
  const mem = newMemoryStore();
  const ct = define<string>("t")
    .withLayers(mem)
    .withTypeGuard((v): v is string => typeof v === "string")
    .build();
  const kc = sampleKC();
  const v = await ct.currentVersion(kc);
  const k = await ct.buildKey(kc, v);
  await mem.set(k, { value: 123, createdAt: new Date(), expiresAt: null }, 0);
  await assert.rejects(() => ct.get(kc), ErrTypeMismatch);
});

test("set bumps version", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  const before = await ct.currentVersion(kc);
  await ct.set(kc, "x");
  assert.equal(await ct.currentVersion(kc), before + 1);
});

test("delete bumps and keeps version key", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  await ct.set(kc, "x");
  const before = await ct.currentVersion(kc);
  await ct.delete(kc);
  assert.equal(await ct.currentVersion(kc), before + 1);
  const got = await ct.get(kc);
  assert.equal(got.ok, false);
  const layers = ct.layers();
  const vk = await ct.versionKey(kc);
  const { hit } = await layers[0]!.get(vk);
  assert.equal(hit, true);
});

test("delete missing still bumps", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  const before = await ct.currentVersion(kc);
  await ct.delete(kc);
  assert.equal(await ct.currentVersion(kc), before + 1);
});

test("getOrLoad hit miss loader", async () => {
  let loads = 0;
  const ct = newStringCache((b) => {
    b.withLoader(async () => {
      loads += 1;
      return "from-loader";
    });
  });
  const kc = sampleKC();
  assert.equal(await ct.getOrLoad(kc), "from-loader");
  assert.equal(loads, 1);
  assert.equal(await ct.getOrLoad(kc), "from-loader");
  assert.equal(loads, 1);
  assert.equal(await ct.currentVersion(kc), 1);
});

test("getOrLoad no loader", async () => {
  const ct = newStringCache();
  await assert.rejects(() => ct.getOrLoad(sampleKC()), ErrNoLoader);
});

test("getOrLoad loader error", async () => {
  const ct = newStringCache((b) => {
    b.withLoader(async () => {
      throw new Error("load fail");
    });
  });
  await assert.rejects(() => ct.getOrLoad(sampleKC()), /load fail/);
});

test("concurrent ensureCurrentVersion", async () => {
  const ct = newStringCache();
  const kc = sampleKC();
  const results = await Promise.all(Array.from({ length: 20 }, () => ct.currentVersion(kc)));
  for (const v of results) assert.equal(typeof v, "number");
  assert.ok(results.every((v) => v >= 1));
});

test("SSCACHIAN_RUNTIME", async () => {
  const { SSCACHIAN_RUNTIME } = await import("./index.js");
  assert.equal(SSCACHIAN_RUNTIME, "node");
});
