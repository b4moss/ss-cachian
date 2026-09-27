import assert from "node:assert/strict";
import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

import {
  ErrInvalidConfig,
  ErrInvalidContext,
  ErrNoLoader,
  createRegistry,
  loadTypes,
  loadTypesJSON,
  loadTypesYAML,
  type KeyContext,
} from "./index.js";

function sampleKC(): KeyContext {
  return { appSlug: "my-app", tenantId: "0123456", queryType: "client_list_page1" };
}

const hasEmu = Boolean(process.env.FIRESTORE_EMULATOR_HOST);

test("registry register and default", async () => {
  const reg = createRegistry();
  reg.registerLoader("load_x", async () => "v");
  reg.registerKeyBuilder("custom", (kc) => `custom:${kc.queryType}`);
  reg.registerKeyBuilder("custom", (kc) => `custom2:${kc.queryType}`);

  const types = await loadTypesYAML<string>(
    reg,
    `
schema_version: "0.9"
types:
  - name: a
    key_builder: default
    layers:
      - driver: memory
  - name: b
    key_builder: custom
    loader: load_x
    layers:
      - driver: memory
`,
  );
  assert.ok(types.get("a"));
  assert.equal(await types.get("b")!.getOrLoad(sampleKC()), "v");
  assert.equal(await types.get("b")!.buildKey(sampleKC(), 1), "custom2:client_list_page1:1");
});

test("registry register errors", () => {
  const reg = createRegistry();
  assert.throws(() => reg.registerLoader("", async () => "x"));
  assert.throws(() => reg.registerLoader("x", null as never));
  assert.throws(() => reg.registerKeyBuilder("", () => "x"));
  assert.throws(() => reg.registerKeyBuilder("x", null as never));
});

test("loadTypes minimal yaml and json", async () => {
  const reg = createRegistry();
  const fromYAML = await loadTypesYAML<string>(
    reg,
    `
schema_version: "0.9"
types:
  - name: user_profile
    key_builder: default
    layers:
      - driver: memory
        ttl: 5m
`,
  );
  const ct = fromYAML.get("user_profile")!;
  assert.equal(ct.layerTTL(0), 5 * 60_000);
  await ct.set(sampleKC(), "hello");
  const got = await ct.get(sampleKC());
  assert.equal(got.ok, true);
  if (got.ok) assert.equal(got.value, "hello");

  const fromJSON = await loadTypesJSON<string>(
    reg,
    JSON.stringify({
      schema_version: "0.9",
      types: [
        {
          name: "user_profile",
          key_builder: "default",
          layers: [{ driver: "memory", ttl: "5m" }],
        },
      ],
    }),
  );
  assert.equal(fromJSON.get("user_profile")!.layerTTL(0), 5 * 60_000);
});

test("loadTypes multiple independent", async () => {
  const reg = createRegistry();
  const types = await loadTypesYAML<string>(
    reg,
    `
schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory}]
  - name: b
    layers: [{driver: memory}]
`,
  );
  const kc = sampleKC();
  await types.get("a")!.set(kc, "A");
  await types.get("b")!.set(kc, "B");
  const ga = await types.get("a")!.get(kc);
  const gb = await types.get("b")!.get(kc);
  assert.equal(ga.ok && ga.value, "A");
  assert.equal(gb.ok && gb.value, "B");
});

test("loadTypes config errors", async () => {
  const reg = createRegistry();
  const mustErr = async (yaml: string) => {
    await assert.rejects(() => loadTypesYAML(reg, yaml));
  };
  await mustErr(`schema_version: "0.8"
types:
  - name: a
    layers: [{driver: memory}]`);
  await mustErr(`types:
  - name: a
    layers: [{driver: memory}]`);
  await mustErr(`schema_version: "0.9"
types: []`);
  await mustErr(`schema_version: "0.9"
types:
  - name: ""
    layers: [{driver: memory}]`);
  await mustErr(`schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory}]
  - name: a
    layers: [{driver: memory}]`);
  await mustErr(`schema_version: "0.9"
types:
  - name: a
    layers: []`);
  await mustErr(`schema_version: "0.9"
types:
  - name: a
    layers: [{driver: valkey}]`);
  await mustErr(`schema_version: "0.9"
types:
  - name: a
    key_builder: missing
    layers: [{driver: memory}]`);
  await mustErr(`schema_version: "0.9"
types:
  - name: a
    loader: missing
    layers: [{driver: memory}]`);
  await mustErr(`schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory, ttl: abc}]`);
  await mustErr(`schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory, ttl: -1s}]`);
  await mustErr(`schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory, ttl: 60}]`);
  assert.ok(ErrInvalidConfig);
});

test("loadTypes TTL precedence", async () => {
  const reg = createRegistry();
  const onlyLayer = await loadTypesYAML(
    reg,
    `
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: memory
        ttl: 1m
`,
  );
  assert.equal(onlyLayer.get("t")!.layerTTL(0), 60_000);

  const onlyType = await loadTypesYAML(
    reg,
    `
schema_version: "0.9"
types:
  - name: t
    ttl: 30m
    layers:
      - driver: memory
      - driver: memory
`,
  );
  assert.equal(onlyType.get("t")!.layerTTL(0), 30 * 60_000);
  assert.equal(onlyType.get("t")!.layerTTL(1), 30 * 60_000);

  const mixed = await loadTypesYAML(
    reg,
    `
schema_version: "0.9"
types:
  - name: t
    ttl: 30m
    layers:
      - driver: memory
        ttl: 1m
      - driver: memory
`,
  );
  assert.equal(mixed.get("t")!.layerTTL(0), 60_000);
  assert.equal(mixed.get("t")!.layerTTL(1), 30 * 60_000);
});

test("loadTypes driver options", async () => {
  const reg = createRegistry();
  await assert.rejects(() =>
    loadTypesYAML(
      reg,
      `
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: firestore
        options:
          emulator_host: localhost:8080
`,
    ),
  );
  await assert.rejects(() =>
    loadTypesYAML(
      reg,
      `
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: memory
        options:
          foo: bar
`,
    ),
  );
  const ok = await loadTypesYAML(
    reg,
    `
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: memory
`,
  );
  assert.ok(ok.get("t"));
});

test("loadTypes firestore collection", async (t) => {
  if (!hasEmu) {
    t.skip("FIRESTORE_EMULATOR_HOST not set");
    return;
  }
  const reg = createRegistry();
  const coll = `sscachian_cfg_node_${Date.now()}`;
  const types = await loadTypesYAML<string>(
    reg,
    `
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: firestore
        options:
          project_id: ss-cachian-dev
          collection: ${coll}
`,
  );
  const kc = sampleKC();
  await types.get("t")!.set(kc, "fs");
  const got = await types.get("t")!.get(kc);
  assert.equal(got.ok, true);
  if (got.ok) assert.equal(got.value, "fs");

  const defaults = await loadTypesYAML(
    reg,
    `
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: firestore
`,
  );
  assert.equal(defaults.get("t")!.layers().length, 1);
});

test("loadTypes integration loader and key builder", async () => {
  const reg = createRegistry();
  let loads = 0;
  reg.registerLoader("load_client_list", async () => {
    loads += 1;
    return "loaded";
  });
  reg.registerKeyBuilder("report_v2", (kc) => `report:${kc.tenantId}:${kc.queryType}`);

  const types = await loadTypesYAML<string>(
    reg,
    `
schema_version: "0.9"
types:
  - name: client_list
    key_builder: default
    loader: load_client_list
    layers:
      - driver: memory
        ttl: 1m
  - name: bare
    layers:
      - driver: memory
  - name: report_cache
    key_builder: report_v2
    layers:
      - driver: memory
`,
  );
  const kc = sampleKC();
  assert.equal(await types.get("client_list")!.getOrLoad(kc), "loaded");
  assert.equal(loads, 1);
  const verBefore = await types.get("client_list")!.currentVersion(kc);
  await types.get("client_list")!.getOrLoad(kc);
  assert.equal(await types.get("client_list")!.currentVersion(kc), verBefore);
  assert.equal(loads, 1);

  await assert.rejects(() => types.get("bare")!.getOrLoad(kc), ErrNoLoader);
  assert.equal(
    await types.get("report_cache")!.buildKey(kc, 1),
    "report:0123456:client_list_page1:1",
  );

  reg.registerLoader("load_client_list", async () => "other");
  const fresh: KeyContext = { appSlug: "my-app", tenantId: "snap", queryType: "q" };
  assert.equal(await types.get("client_list")!.getOrLoad(fresh), "loaded");
});

test("loadTypes multilayer write-back", async (t) => {
  if (!hasEmu) {
    t.skip("FIRESTORE_EMULATOR_HOST not set");
    return;
  }
  const reg = createRegistry();
  const coll = `sscachian_wb_node_${Date.now()}`;
  const types = await loadTypesYAML<string>(
    reg,
    `
schema_version: "0.9"
types:
  - name: client_list
    layers:
      - driver: memory
        ttl: 1m
      - driver: firestore
        ttl: 1h
        options:
          project_id: ss-cachian-dev
          collection: ${coll}
`,
  );
  const ct = types.get("client_list")!;
  const kc = sampleKC();
  await ct.set(kc, "wb");
  const key = await ct.buildLatestKey(kc);
  await ct.layers()[0]!.delete(key);
  const got = await ct.get(kc);
  assert.equal(got.ok, true);
  if (got.ok) assert.equal(got.value, "wb");
  const l1 = await ct.layers()[0]!.get(key);
  assert.equal(l1.hit, true);
  assert.equal(l1.entry.value, "wb");
});

test("loadTypes from file", async () => {
  const reg = createRegistry();
  const dir = await mkdtemp(path.join(tmpdir(), "sscachian-"));
  const file = path.join(dir, "sscachian.yaml");
  await writeFile(
    file,
    `
schema_version: "0.9"
types:
  - name: user_profile
    key_builder: default
    layers:
      - driver: memory
        ttl: 5m
`,
  );
  const types = await loadTypes<string>(reg, file);
  assert.ok(types.get("user_profile"));
});

test("loadTypes invalid context", async () => {
  const reg = createRegistry();
  const types = await loadTypesYAML<string>(
    reg,
    `
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: memory
`,
  );
  await assert.rejects(() => types.get("t")!.get({ appSlug: "", tenantId: "", queryType: "" }), ErrInvalidContext);
});
