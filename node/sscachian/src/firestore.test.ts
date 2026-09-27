import assert from "node:assert/strict";
import test from "node:test";

import { ErrEmptyKey, ErrNotInteger } from "./index.js";
import { newFirestoreStore } from "./driver/firestore.js";

const hasEmu = Boolean(process.env.FIRESTORE_EMULATOR_HOST);

function skipWithoutEmu(t: test.TestContext): boolean {
  if (!hasEmu) {
    t.skip("FIRESTORE_EMULATOR_HOST not set");
    return true;
  }
  return false;
}

test("firestore get set delete ttl", async (t) => {
  if (skipWithoutEmu(t)) return;
  const s = await newFirestoreStore({
    projectId: "ss-cachian-dev",
    collection: `sscachian_test_${Date.now()}_round`,
  });
  try {
    await s.set("k", { value: "hello", createdAt: new Date(0), expiresAt: null }, 0);
    let r = await s.get("k");
    assert.equal(r.hit, true);
    assert.equal(r.entry.value, "hello");

    await s.set("exp", { value: 1, createdAt: new Date(), expiresAt: new Date(Date.now() - 60_000) }, 0);
    r = await s.get("exp");
    assert.equal(r.hit, false);

    await s.delete("k");
    assert.equal((await s.get("k")).hit, false);
    await s.delete("missing");

    await assert.rejects(() => s.get(""), ErrEmptyKey);
  } finally {
    await s.clearCollection();
    await s.close();
  }
});

test("firestore incr", async (t) => {
  if (skipWithoutEmu(t)) return;
  const s = await newFirestoreStore({
    projectId: "ss-cachian-dev",
    collection: `sscachian_test_${Date.now()}_incr`,
  });
  try {
    assert.equal(await s.incr("c"), 1);
    assert.equal(await s.incr("c"), 2);
    await s.set("bad", { value: "x", createdAt: new Date(), expiresAt: null }, 0);
    await assert.rejects(() => s.incr("bad"), ErrNotInteger);

    const n = 20;
    // Emulator may abort under contention; keep trying until each call succeeds (Go parity).
    await Promise.all(
      Array.from({ length: n }, async () => {
        for (;;) {
          try {
            await s.incr("par");
            return;
          } catch (err) {
            const code = (err as { code?: number | string }).code;
            const aborted =
              code === 10 ||
              code === "ABORTED" ||
              (err instanceof Error && /ABORTED|aborted|lock timeout/i.test(err.message));
            if (!aborted) throw err;
            await new Promise((r) => setTimeout(r, 1));
          }
        }
      }),
    );
    const final = await s.incr("par");
    assert.equal(final, n + 1);
  } finally {
    await s.clearCollection();
    await s.close();
  }
});

test("firestore purgeExact", async (t) => {
  if (skipWithoutEmu(t)) return;
  const s = await newFirestoreStore({
    projectId: "ss-cachian-dev",
    collection: `sscachian_test_${Date.now()}_purge`,
  });
  try {
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
  } finally {
    await s.clearCollection();
    await s.close();
  }
});
