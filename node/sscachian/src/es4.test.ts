import assert from "node:assert/strict";
import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import test from "node:test";

import { ErrClosed, ErrNotFound, type JsonRaw, type Tx } from "@b4moss/es4";

import {
  ErrEmptyKey,
  ErrNegativeTTL,
  ErrNotInteger,
  isVersionDataKey,
} from "./index.js";
import {
  Es4Store,
  type Es4Backend,
  newEs4Store,
} from "./driver/es4.js";

class MemBackend implements Es4Backend {
  #m = new Map<string, string>();
  #closed = false;
  scanCalls = 0;

  async set(key: string, value: JsonRaw): Promise<void> {
    this.#guard();
    this.#m.set(key, typeof value === "string" ? value : Buffer.from(value).toString("utf8"));
  }
  async get(key: string): Promise<JsonRaw> {
    this.#guard();
    const v = this.#m.get(key);
    if (v == null) throw ErrNotFound;
    return v;
  }
  async delete(key: string): Promise<void> {
    this.#guard();
    if (!this.#m.has(key)) throw ErrNotFound;
    this.#m.delete(key);
  }
  async scan(prefix: string): Promise<string[]> {
    this.#guard();
    this.scanCalls++;
    return [...this.#m.keys()].filter((k) => k.startsWith(prefix));
  }
  async beginTx(): Promise<Tx> {
    this.#guard();
    const snap = new Map(this.#m);
    const overlay = new Map<string, string | null>();
    let done = false;
    const self = this;
    const get = async (key: string): Promise<JsonRaw> => {
      if (done) throw new Error("done");
      if (overlay.has(key)) {
        const v = overlay.get(key);
        if (v == null) throw ErrNotFound;
        return v;
      }
      const v = snap.get(key);
      if (v == null) throw ErrNotFound;
      return v;
    };
    return {
      async set(key, value) {
        if (done) throw new Error("done");
        overlay.set(key, typeof value === "string" ? value : Buffer.from(value).toString("utf8"));
      },
      get,
      async delete(key) {
        if (done) throw new Error("done");
        overlay.set(key, null);
      },
      async exists(key) {
        try {
          await get(key);
          return true;
        } catch (e) {
          if (e === ErrNotFound) return false;
          throw e;
        }
      },
      async scan(prefix: string) {
        if (done) throw new Error("done");
        const keys = new Set<string>();
        for (const [k, v] of overlay) {
          if (v != null && k.startsWith(prefix)) keys.add(k);
        }
        for (const k of snap.keys()) {
          if (overlay.has(k) && overlay.get(k) == null) continue;
          if (overlay.has(k)) continue;
          if (k.startsWith(prefix)) keys.add(k);
        }
        return [...keys];
      },
      async clear() {
        for (const k of snap.keys()) overlay.set(k, null);
      },
      async commit() {
        if (done) throw new Error("done");
        for (const [k, v] of overlay) {
          if (v == null) self.#m.delete(k);
          else self.#m.set(k, v);
        }
        done = true;
      },
      async rollback() {
        done = true;
      },
    };
  }
  async close(): Promise<void> {
    this.#closed = true;
  }
  #guard(): void {
    if (this.#closed) throw ErrClosed;
  }
}

test("es4 embedded open set get", async () => {
  const s = await newEs4Store({ memoryOnly: true });
  try {
    await s.set("a:b", { value: "v", createdAt: new Date(0), expiresAt: null }, 0);
    const r = await s.get("a:b");
    assert.equal(r.hit, true);
    assert.equal(r.entry.value, "v");
  } finally {
    await s.close();
  }
});

test("es4 embedded PurgePrefix uses real Scan", async () => {
  const s = await newEs4Store({ memoryOnly: true });
  try {
    const prefix = "app:cache:t:q";
    for (const k of [`${prefix}:1`, `${prefix}:2`, `${prefix}:__version__`, `${prefix}:extra`, "other:1"]) {
      await s.set(k, { value: 1, createdAt: new Date(0), expiresAt: null }, 0);
    }
    await s.purgePrefix(prefix);
    assert.equal((await s.get(`${prefix}:1`)).hit, false);
    assert.equal((await s.get(`${prefix}:extra`)).hit, false);
    assert.equal((await s.get("other:1")).hit, true);
    await assert.rejects(
      () => s.set(":bad", { value: 1, createdAt: new Date(0), expiresAt: null }, 0),
    );
  } finally {
    await s.close();
  }
});

test("es4 close idempotent blocks ops", async () => {
  const s = Es4Store.fromBackend(new MemBackend());
  await s.close();
  await s.close();
  await assert.rejects(() => s.set("k", { value: 1, createdAt: new Date(0), expiresAt: null }, 0), ErrClosed);
});

test("es4 entry json ttl and corrupt miss", async () => {
  const b = new MemBackend();
  const s = Es4Store.fromBackend(b);
  try {
    await s.set("t", { value: 1, createdAt: new Date(0), expiresAt: null }, 60_000);
    assert.equal((await s.get("t")).hit, true);
    await s.set(
      "old",
      { value: 2, createdAt: new Date(), expiresAt: new Date(Date.now() - 60_000) },
      0,
    );
    assert.equal((await s.get("old")).hit, false);
    await b.set("bad", `"nope"`);
    assert.equal((await s.get("bad")).hit, false);
    await assert.rejects(() => s.get(""), ErrEmptyKey);
    await assert.rejects(
      () => s.set("k", { value: 1, createdAt: new Date(0), expiresAt: null }, -1),
      ErrNegativeTTL,
    );
  } finally {
    await s.close();
  }
});

test("es4 delete incr purge", async () => {
  const b = new MemBackend();
  const s = Es4Store.fromBackend(b);
  try {
    await s.set("d", { value: 1, createdAt: new Date(0), expiresAt: null }, 0);
    await s.delete("d");
    assert.equal((await s.get("d")).hit, false);
    await s.delete("d");
    assert.equal(await s.incr("c"), 1);
    assert.equal(await s.incr("c"), 2);
    await s.set("s", { value: "x", createdAt: new Date(0), expiresAt: null }, 0);
    await assert.rejects(() => s.incr("s"), ErrNotInteger);

    const prefix = "app:cache:t:q";
    await s.set(`${prefix}:1`, { value: "a", createdAt: new Date(0), expiresAt: null }, 0);
    await s.set(`${prefix}:2`, { value: "b", createdAt: new Date(0), expiresAt: null }, 0);
    await s.set(`${prefix}:__version__`, { value: 2, createdAt: new Date(0), expiresAt: null }, 0);
    await s.set("other:1", { value: 1, createdAt: new Date(0), expiresAt: null }, 0);
    b.scanCalls = 0;
    await s.purgeExact(prefix);
    assert.equal(b.scanCalls, 0);
    assert.equal((await s.get(`${prefix}:1`)).hit, false);
    assert.equal((await s.get(`${prefix}:__version__`)).hit, true);
    assert.ok(isVersionDataKey(prefix, `${prefix}:1`));

    await s.set(`${prefix}:1`, { value: 1, createdAt: new Date(0), expiresAt: null }, 0);
    await s.set(`${prefix}:extra`, { value: 1, createdAt: new Date(0), expiresAt: null }, 0);
    await s.purgePrefix(prefix);
    assert.equal((await s.get(`${prefix}:1`)).hit, false);
    assert.equal((await s.get(`${prefix}:extra`)).hit, false);
    assert.equal((await s.get("other:1")).hit, true);
  } finally {
    await s.close();
  }
});

test("es4 http adapter set get purgePrefix", async () => {
  const store = new Map<string, string>();
  const txs = new Map<string, Map<string, string | null>>();
  let txSeq = 0;
  const srv = createServer((req: IncomingMessage, res: ServerResponse) => {
    const url = new URL(req.url ?? "/", "http://127.0.0.1");
    const writeErr = (status: number, code: string, message: string) => {
      res.statusCode = status;
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify({ error: { code, message } }));
    };
    if (req.method === "GET" && url.pathname === "/readyz") {
      res.statusCode = 200;
      res.end();
      return;
    }
    if (req.method === "GET" && url.pathname === "/v1/scan") {
      const prefix = url.searchParams.get("prefix") ?? "";
      const keys = [...store.keys()].filter((k) => k.startsWith(prefix));
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify(keys));
      return;
    }
    if (req.method === "POST" && url.pathname === "/v1/tx") {
      txSeq++;
      const id = `tx-${txSeq}`;
      txs.set(id, new Map());
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify({ id }));
      return;
    }
    const keyMatch = url.pathname.match(/^\/v1\/keys\/(.+)$/);
    if (keyMatch) {
      const key = decodeURIComponent(keyMatch[1]!);
      if (req.method === "PUT") {
        const chunks: Buffer[] = [];
        req.on("data", (c) => chunks.push(c));
        req.on("end", () => {
          store.set(key, Buffer.concat(chunks).toString("utf8"));
          res.statusCode = 200;
          res.end();
        });
        return;
      }
      if (req.method === "GET") {
        const v = store.get(key);
        if (v == null) return writeErr(404, "not_found", "missing");
        res.setHeader("Content-Type", "application/json");
        res.end(v);
        return;
      }
      if (req.method === "DELETE") {
        if (!store.has(key)) return writeErr(404, "not_found", "missing");
        store.delete(key);
        res.statusCode = 200;
        res.end();
        return;
      }
    }
    const txKey = url.pathname.match(/^\/v1\/tx\/([^/]+)\/keys\/(.+)$/);
    if (txKey) {
      const id = decodeURIComponent(txKey[1]!);
      const key = decodeURIComponent(txKey[2]!);
      const ov = txs.get(id);
      if (!ov) return writeErr(404, "not_found", "tx");
      if (req.method === "PUT") {
        const chunks: Buffer[] = [];
        req.on("data", (c) => chunks.push(c));
        req.on("end", () => {
          ov.set(key, Buffer.concat(chunks).toString("utf8"));
          res.statusCode = 200;
          res.end();
        });
        return;
      }
      if (req.method === "GET") {
        if (ov.has(key)) {
          const v = ov.get(key);
          if (v == null) return writeErr(404, "not_found", "missing");
          res.setHeader("Content-Type", "application/json");
          res.end(v);
          return;
        }
        const v = store.get(key);
        if (v == null) return writeErr(404, "not_found", "missing");
        res.setHeader("Content-Type", "application/json");
        res.end(v);
        return;
      }
    }
    const txCommit = url.pathname.match(/^\/v1\/tx\/([^/]+)\/commit$/);
    if (txCommit && req.method === "POST") {
      const id = decodeURIComponent(txCommit[1]!);
      const ov = txs.get(id);
      if (!ov) return writeErr(404, "not_found", "tx");
      for (const [k, v] of ov) {
        if (v == null) store.delete(k);
        else store.set(k, v);
      }
      txs.delete(id);
      res.statusCode = 200;
      res.end();
      return;
    }
    const txRollback = url.pathname.match(/^\/v1\/tx\/([^/]+)\/rollback$/);
    if (txRollback && req.method === "POST") {
      txs.delete(decodeURIComponent(txRollback[1]!));
      res.statusCode = 200;
      res.end();
      return;
    }
    writeErr(404, "not_found", "route");
  });
  await new Promise<void>((resolve) => srv.listen(0, "127.0.0.1", resolve));
  const addr = srv.address();
  assert.ok(addr && typeof addr === "object");
  const baseUrl = `http://127.0.0.1:${addr.port}`;
  try {
    const s = await Es4Store.createHTTP({ baseUrl });
    try {
      await s.set("a:b:1", { value: "v", createdAt: new Date(0), expiresAt: null }, 0);
      assert.equal((await s.get("a:b:1")).entry.value, "v");
      await s.set("a:b:2", { value: 1, createdAt: new Date(0), expiresAt: null }, 0);
      await s.set("other:1", { value: 1, createdAt: new Date(0), expiresAt: null }, 0);
      await s.purgePrefix("a:b");
      assert.equal((await s.get("a:b:1")).hit, false);
      assert.equal((await s.get("other:1")).hit, true);
      assert.equal(await s.incr("c"), 1);
    } finally {
      await s.close();
    }
  } finally {
    await new Promise<void>((resolve, reject) => srv.close((e) => (e ? reject(e) : resolve())));
  }
});
