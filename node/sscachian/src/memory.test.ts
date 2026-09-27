import assert from "node:assert/strict";
import test from "node:test";

import { ErrEmptyKey, ErrNegativeTTL, ErrNotInteger, isVersionDataKey } from "./index.js";
import { newMemoryStore } from "./driver/memory.js";

test("memory get set delete ttl", async () => {
  const s = newMemoryStore();
  await s.set("k", { value: "v", createdAt: new Date(0), expiresAt: null }, 0);
  let r = await s.get("k");
  assert.equal(r.hit, true);
  assert.equal(r.entry.value, "v");

  await s.set("k2", { value: 1, createdAt: new Date(0), expiresAt: null }, 50);
  r = await s.get("k2");
  assert.equal(r.hit, true);
  await s.set("exp", { value: 1, createdAt: new Date(), expiresAt: new Date(Date.now() - 1000) }, 0);
  r = await s.get("exp");
  assert.equal(r.hit, false);

  await s.delete("k");
  r = await s.get("k");
  assert.equal(r.hit, false);
  await s.delete("missing");
});

test("memory empty key and negative ttl", async () => {
  const s = newMemoryStore();
  await assert.rejects(() => s.get(""), ErrEmptyKey);
  await assert.rejects(() => s.set("", { value: 1, createdAt: new Date(), expiresAt: null }, 0), ErrEmptyKey);
  await assert.rejects(
    () => s.set("k", { value: 1, createdAt: new Date(), expiresAt: null }, -1),
    ErrNegativeTTL,
  );
});

test("memory incr", async () => {
  const s = newMemoryStore();
  assert.equal(await s.incr("c"), 1);
  assert.equal(await s.incr("c"), 2);
  await s.set("bad", { value: "x", createdAt: new Date(), expiresAt: null }, 0);
  await assert.rejects(() => s.incr("bad"), ErrNotInteger);
});

test("memory purgeExact", async () => {
  const s = newMemoryStore();
  await s.set("prefix:1", { value: 1, createdAt: new Date(), expiresAt: null }, 0);
  await s.set("prefix:2", { value: 2, createdAt: new Date(), expiresAt: null }, 0);
  await s.set("prefix:__version__", { value: 9, createdAt: new Date(), expiresAt: null }, 0);
  await s.set("other:1", { value: 3, createdAt: new Date(), expiresAt: null }, 0);
  await s.purgeExact("prefix");
  assert.equal((await s.get("prefix:1")).hit, false);
  assert.equal((await s.get("prefix:2")).hit, false);
  assert.equal((await s.get("prefix:__version__")).hit, true);
  assert.equal((await s.get("other:1")).hit, true);
  await assert.rejects(() => s.purgeExact(""), ErrEmptyKey);
});

test("isVersionDataKey", () => {
  assert.equal(isVersionDataKey("p", "p:1"), true);
  assert.equal(isVersionDataKey("p", "p:12"), true);
  assert.equal(isVersionDataKey("p", "p:__version__"), false);
  assert.equal(isVersionDataKey("p", "p:"), false);
  assert.equal(isVersionDataKey("p", "other:1"), false);
});

test("memory concurrent incr", async () => {
  const s = newMemoryStore();
  const n = 50;
  await Promise.all(Array.from({ length: n }, () => s.incr("x")));
  assert.equal(await s.incr("x"), n + 1);
});
