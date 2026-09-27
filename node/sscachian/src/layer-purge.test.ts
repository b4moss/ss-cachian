import assert from "node:assert/strict";
import test from "node:test";

import {
  ErrInvalidLayerIndex,
  ErrNoLoader,
  defaultKeyBuilder,
  define,
  newMemoryStore,
  type KeyContext,
  type Layer,
} from "./index.js";

function sampleKC(): KeyContext {
  return { appSlug: "my-app", tenantId: "0123456", queryType: "client_list_page1" };
}

test("multilayer get write-back", async () => {
  const l1 = newMemoryStore();
  const l2 = newMemoryStore();
  const ct = define<string>("t").withLayers(l1, l2).withKeyBuilder(defaultKeyBuilder).build();
  const kc = sampleKC();
  await ct.set(kc, "v");
  // clear L1 only
  const key = await ct.buildLatestKey(kc);
  await l1.delete(key);
  const got = await ct.get(kc);
  assert.equal(got.ok, true);
  if (got.ok) assert.equal(got.value, "v");
  assert.equal((await l1.get(key)).hit, true);
});

test("multilayer get l1 hit skips l2", async () => {
  const l1 = newMemoryStore();
  let l2Gets = 0;
  const l2: Layer = {
    async get(key, signal) {
      l2Gets += 1;
      return newMemoryStore().get(key, signal);
    },
    async set() {},
    async delete() {},
    async incr() {
      return 1;
    },
    async purgeExact() {},
  };
  const ct = define<string>("t").withLayers(l1, l2).build();
  const kc = sampleKC();
  await ct.set(kc, "x");
  l2Gets = 0;
  await ct.get(kc);
  assert.equal(l2Gets, 0);
});

test("multilayer all miss", async () => {
  const ct = define<string>("t").withLayers(newMemoryStore(), newMemoryStore()).build();
  const got = await ct.get(sampleKC());
  assert.equal(got.ok, false);
});

test("getOrLoad multilayer no bump", async () => {
  let loads = 0;
  const ct = define<string>("t")
    .withLayers(newMemoryStore(), newMemoryStore())
    .withLoader(async () => {
      loads += 1;
      return "L";
    })
    .build();
  const kc = sampleKC();
  assert.equal(await ct.getOrLoad(kc), "L");
  assert.equal(await ct.currentVersion(kc), 1);
  assert.equal(loads, 1);
  assert.equal(await ct.getOrLoad(kc), "L");
  assert.equal(loads, 1);
});

test("getOrLoad no loader multilayer", async () => {
  const ct = define("t").withLayers(newMemoryStore(), newMemoryStore()).build();
  await assert.rejects(() => ct.getOrLoad(sampleKC()), ErrNoLoader);
});

test("set writes all layers with per-layer ttl", async () => {
  const l1 = newMemoryStore();
  const l2 = newMemoryStore();
  const ct = define<string>("t")
    .withLayers(l1, l2)
    .withLayerTTLs(3_600_000, 60_000)
    .build();
  assert.equal(ct.layerTTL(0), 3_600_000);
  assert.equal(ct.layerTTL(1), 60_000);
  const kc = sampleKC();
  await ct.set(kc, "z");
  const key = await ct.buildLatestKey(kc);
  assert.equal((await l1.get(key)).hit, true);
  assert.equal((await l2.get(key)).hit, true);
});

test("delete all layers", async () => {
  const l1 = newMemoryStore();
  const l2 = newMemoryStore();
  const ct = define<string>("t").withLayers(l1, l2).build();
  const kc = sampleKC();
  await ct.set(kc, "z");
  const keyBefore = await ct.buildLatestKey(kc);
  await ct.delete(kc);
  // latest key after bump is different; previous key should be gone from both
  assert.equal((await l1.get(keyBefore)).hit, false);
  assert.equal((await l2.get(keyBefore)).hit, false);
});

test("purge all layers keeps version", async () => {
  const l1 = newMemoryStore();
  const l2 = newMemoryStore();
  const ct = define<string>("t").withLayers(l1, l2).build();
  const kc = sampleKC();
  await ct.set(kc, "a");
  await ct.set(kc, "b");
  const ver = await ct.currentVersion(kc);
  await ct.purge(kc);
  assert.equal(await ct.currentVersion(kc), ver);
  const got = await ct.get(kc);
  assert.equal(got.ok, false);
  const vk = await ct.versionKey(kc);
  assert.equal((await l1.get(vk)).hit, true);
});

test("purge layer subset", async () => {
  const l1 = newMemoryStore();
  const l2 = newMemoryStore();
  const ct = define<string>("t").withLayers(l1, l2).build();
  const kc = sampleKC();
  await ct.set(kc, "a");
  const key = await ct.buildLatestKey(kc);
  await ct.purge(kc, [1]);
  assert.equal((await l1.get(key)).hit, true);
  assert.equal((await l2.get(key)).hit, false);
});

test("purge invalid index", async () => {
  const ct = define("t").withLayers(newMemoryStore()).build();
  await assert.rejects(() => ct.purge(sampleKC(), [5]), ErrInvalidLayerIndex);
});

test("purge after multi set then bump set", async () => {
  const ct = define<string>("t").withLayers(newMemoryStore()).build();
  const kc = sampleKC();
  await ct.set(kc, "a");
  await ct.set(kc, "b");
  await ct.purge(kc);
  await ct.bumpVersion(kc);
  await ct.set(kc, "c");
  const got = await ct.get(kc);
  assert.equal(got.ok, true);
  if (got.ok) assert.equal(got.value, "c");
});
