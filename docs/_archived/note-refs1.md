---
type: Note
title: 壁打ち参考その1
description: 生成AIと壁打ちした結果の旧稿。現行決定は docs/ 直下を正とする。
tags: [archived]
role: note
timestamp: 2026-09-07T12:00:00Z
---

# Server-side Cache Library

Goで実装する、**Cache Typeを中心とした汎用サーバーサイドキャッシュライブラリ**。

単一のキャッシュストレージを抽象化するのではなく、

> 「この種類のデータを、どのようなキーで、どのキャッシュ層に、どのポリシーで保存するか」

という**キャッシュ戦略そのものを定義・実行する**ことを目的とする。

---

## 1. 基本コンセプト

中心となる概念は `Cache Type`。

例えば、

- 商品
- ユーザーセッション
- 記事
- 検索結果
- APIレスポンス
- 集計結果

など、キャッシュ対象の種類ごとにキャッシュ戦略を定義する。

```text
Cache Type
│
├── Key Strategy
│
├── Layers
│   ├── L1
│   ├── L2
│   └── L3 ...
│
├── Policy
│
├── Invalidation
│
└── Loader
```

---

# 2. Cache Type

例えば商品キャッシュ。

```go
productCache := cache.Define("product", CacheType[ProductKey, Product]{
    Key: productKey,

    Layers: []Layer{
        Memory(...),
        Valkey(...),
        ObjectStorage(...),
    },

    Policy: Policy{
        TTL: time.Hour,
    },

    Loader: loadProduct,
})
```

利用側は、

```go
product, err := productCache.Get(ctx, ProductKey{
    TenantID:  123,
    ProductID: 456,
    Locale:    "ja",
})
```

だけでよい。

利用側は、

- どこに保存されているか
- L1/L2/L3がどう構成されているか
- キーがどう生成されるか
- キャッシュミス時に何を取得するか

を意識しなくてよい。

---

# 3. 多層キャッシュ

基本的な構成は、

```text
             Cache Type
                 │
                 ▼
            ┌─────────┐
            │   L1    │
            │ Memory  │
            └────┬────┘
                 │ miss
                 ▼
            ┌─────────┐
            │   L2    │
            │ Valkey  │
            └────┬────┘
                 │ miss
                 ▼
            ┌─────────┐
            │   L3    │
            │ S3/R2   │
            └────┬────┘
                 │ miss
                 ▼
             ┌───────┐
             │ Loader│
             └───┬───┘
                 ▼
             DB / API
```

例えば、

```text
L1 Memory
    ↓ miss
L2 Valkey
    ↓ miss
L3 Object Storage
    ↓ miss
Database
```

という構成。

上位層で取得できた場合、必要に応じて下位層へ結果を書き戻す。

---

# 4. Cache Typeの例

## Session

```text
Session

L1: Memory
L2: Valkey

Key:
session:{session_id}

TTL:
24h
```

通常はL1で処理し、インスタンスをまたいだ場合などはL2のValkeyから取得する。

---

## Product

```text
Product

L1: Memory
L2: Valkey
L3: Object Storage

Key:
tenant:{tenant_id}:product:{product_id}:{locale}

TTL:
1h
```

---

## Article

```text
Article

L1: Memory
L2: Object Storage

Key:
article:{article_id}:{locale}:{version}

TTL:
24h
```

頻繁に変更されず、生成済みJSONなどが数十KB〜数百KBになる場合に適する。

---

## Search Result

```text
SearchResult

L1: Memory
L2: Valkey

Key:
search:{hash(query + filters + user_context)}

TTL:
5m
```

---

# 5. Key Strategy

キー生成もCache Typeの一部として定義する。

```go
Key: func(k ProductKey) string {
    return fmt.Sprintf(
        "tenant:%d:product:%d:%s",
        k.TenantID,
        k.ProductID,
        k.Locale,
    )
}
```

単純なIDだけではなく、アプリケーションのビジネスコンテキストをキーに含められる。

```text
tenant:123:product:456:ja
tenant:123:product:456:en
tenant:456:product:456:ja
```

これによりマルチテナント環境でも衝突を防げる。

---

# 6. Key Builder

Key Builderを独立した抽象としてもよい。

```go
type KeyBuilder[K any] interface {
    Build(K) string
}
```

例えば、

```go
type ProductKey struct {
    TenantID  int64
    ProductID int64
    Locale    string
}
```

に対して、

```go
func ProductKeyBuilder(k ProductKey) string {
    return fmt.Sprintf(
        "product:%d:%d:%s",
        k.TenantID,
        k.ProductID,
        k.Locale,
    )
}
```

とする。

---

# 7. Layer

Layerは実際のデータ保存場所。

```go
type Layer interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

実装例：

```text
Memory
Valkey
Redis
Memcached
Firestore
S3
R2
GCS
SQLite
Filesystem
```

など。

---

# 8. Memory Layer

L1として利用するプロセス内キャッシュ。

候補：

- Ristretto
- BigCache
- go-cache

第一候補はRistretto。

```text
Cloud Run Instance A
    └── L1 Memory

Cloud Run Instance B
    └── L1 Memory

Cloud Run Instance C
    └── L1 Memory
```

インスタンス間では共有されない。

そのため、

```text
L1 Memory
    ↓ miss
L2 Valkey
```

という構成が有効。

---

# 9. Valkey Layer

共有キャッシュとしてValkeyを利用。

Goクライアント候補：

```text
valkey-io/valkey-go
```

またはRedis互換クライアントとして、

```text
redis/rueidis
```

など。

用途：

- セッション
- 頻繁なAPIレスポンス
- 高頻度キャッシュ
- Rate Limit
- Lock
- Stampede Protection

など。

---

# 10. Object Storage Layer

S3互換Object Storageをキャッシュ層として利用する。

候補：

```text
AWS S3
Cloudflare R2
MinIO
その他S3互換ストレージ
```

用途：

- 数十KB〜数MB程度の大きなレスポンス
- 生成済みJSON
- HTML
- APIレスポンス
- 集計結果
- 画像・ファイル
- 長めのTTLのキャッシュ

など。

これは従来型の「高速キャッシュ」というより、

> **計算済み成果物を保存して再計算を避けるキャッシュ**

として考える。

---

# 11. Firestore Layer

FirestoreもDriver候補。

```text
L1 Memory
    ↓
L2 Firestore
    ↓
DB
```

のような構成が可能。

特に、

- 小規模SaaS
- Cloud Run
- 常時起動Redisを置きたくない
- セッション
- 低〜中頻度のキャッシュ

などでは有効。

ただし、高頻度アクセスやAtomic Operation、Lock、Pub/SubなどはValkeyの方が適している。

---

# 12. Cache Policy

Cache TypeにはPolicyを持たせる。

```go
type Policy struct {
    TTL               time.Duration
    StaleWhileRevalidate time.Duration
    StaleIfError      time.Duration
}
```

候補：

```text
TTL
max-age
stale-while-revalidate
stale-if-error
negative cache
stampede protection
```

など。

---

# 13. TTL

基本的なTTL。

```go
Policy: Policy{
    TTL: time.Hour,
}
```

ただし、LayerごとにTTLを変えられるようにしてもよい。

```text
L1 Memory
TTL: 30s

L2 Valkey
TTL: 10m

L3 Object Storage
TTL: 1h
```

これにより、

```text
L1 → 超高速・短TTL
L2 → 高速・中TTL
L3 → 低コスト・長TTL
```

という構成が可能。

---

# 14. Stale-While-Revalidate

期限切れでも古い値を一時的に返し、バックグラウンドで更新する。

```text
Request
   │
   ▼
Cache
   │
   ├── Fresh → return
   │
   └── Stale
        │
        ├── return stale
        │
        └── background refresh
```

APIレスポンスや集計結果などに特に有効。

---

# 15. Stale-If-Error

LoaderやDBが障害になった場合、期限切れのキャッシュを返す。

```text
Cache expired
      │
      ▼
   Loader
      │
      ├── success → new value
      │
      └── error
           │
           ▼
      stale value
```

障害時の可用性向上に使える。

---

# 16. Stampede Protection

大量のリクエストが同時にCache Missした場合、

```text
100 requests
     │
     ▼
100 DB queries
```

となる問題を防ぐ。

理想的には、

```text
100 requests
     │
     ▼
  Cache Miss
     │
     ▼
  1 request → DB
     │
     ▼
  Cache Set
     │
     ▼
  99 requests → cached value
```

とする。

単一インスタンスなら`singleflight`系。

複数インスタンスではValkeyなどを利用したDistributed Lockなどを検討。

---

# 17. Invalidation

Cache TypeごとにInvalidate戦略を持てるようにする。

基本：

```text
Exact
Prefix
Tag
Predicate
```

---

## Exact

```go
cache.Product.Delete(ctx, key)
```

---

## Prefix

```text
tenant:123:product:*
```

など。

---

## Tag

例えば、

```text
product:456
category:10
tenant:123
```

のようなタグを付ける。

```text
Cache Entry
    │
    ├── tag: product:456
    ├── tag: category:10
    └── tag: tenant:123
```

商品更新時に、

```go
cache.PurgeByTag(ctx, "product:456")
```

とできる。

---

# 18. Loader

Cache Miss時にデータを取得する処理。

```go
Loader: loadProduct
```

例えば、

```go
func loadProduct(
    ctx context.Context,
    key ProductKey,
) (Product, error) {
    return repository.FindProduct(
        ctx,
        key.TenantID,
        key.ProductID,
        key.Locale,
    )
}
```

Cache Typeが、

```text
Get
 ↓
L1
 ↓
L2
 ↓
L3
 ↓
Loader
```

を管理する。

---

# 19. Read-through Cache

LoaderをCache Typeに持たせることで、

```go
product, err := productCache.Get(ctx, key)
```

だけで、

```text
L1
 ↓
L2
 ↓
L3
 ↓
Loader
 ↓
Set L3
 ↓
Set L2
 ↓
Set L1
 ↓
return
```

というRead-through Cacheを実現できる。

---

# 20. Write / Set

明示的なSetも可能にする。

```go
err := productCache.Set(ctx, key, product)
```

Layer設定に従って保存する。

```text
Set
 │
 ├── L1
 ├── L2
 └── L3
```

ただし、Layerごとに保存するかどうかはPolicyで制御可能にする余地がある。

---

# 21. Cache-asideとの共存

必ずしもLoaderを使わなくてもよい。

```go
value, err := cache.Get(ctx, key)

if err == cache.ErrMiss {
    value, err = repository.Find(...)
    if err == nil {
        cache.Set(ctx, key, value)
    }
}
```

つまり、

```text
Read-through
Cache-aside
Write-through
```

など複数の利用パターンを許容する。

---

# 22. Namespace

アプリケーションやCache TypeごとにNamespaceを持たせる。

```text
app:a:session:xxx
app:a:product:123
app:b:session:xxx
app:b:article:456
```

例えば、

```go
cache.Define("product", ...)
```

なら内部的に、

```text
product:{key}
```

などのNamespaceを自動付与できる。

---

# 23. Cache Typeを独立した設定として扱う

例えば設定ファイルで、

```yaml
cache_types:

  product:
    layers:
      - memory
      - valkey
      - object_storage

    key:
      pattern: "tenant:{tenant_id}:product:{product_id}:{locale}"

    policy:
      ttl: 1h

  session:
    layers:
      - memory
      - valkey

    key:
      pattern: "session:{session_id}"

    policy:
      ttl: 24h

  article:
    layers:
      - memory
      - object_storage

    key:
      pattern: "article:{id}:{locale}:{version}"

    policy:
      ttl: 24h
```

という設計も可能。

ただし、Goではまず型安全なAPIを優先し、設定ファイル対応は後からでもよい。

---

# 24. DriverとCache Typeを分離する

重要なのは、

```text
Driver
```

と、

```text
Cache Type
```

を明確に分離すること。

Driver：

```text
「どこに保存するか」
```

Cache Type：

```text
「何を、どうキャッシュするか」
```

例えば、

```text
             Cache Type: Product
                     │
           ┌─────────┼─────────┐
           ▼         ▼         ▼
        Memory     Valkey      R2
          │           │         │
       Driver       Driver    Driver
```

とする。

---

# 25. 想定Driver

最初から全部実装する必要はない。

第一段階：

```text
Memory
Valkey
S3-compatible Object Storage
```

第二段階：

```text
Firestore
GCS
Redis
Memcached
SQLite
Filesystem
```

など。

---

# 26. 既存Goライブラリとの関係

候補：

### Ristretto

L1 Memoryの候補。

高速なプロセス内キャッシュとして利用。

### BigCache

大量のエントリを扱うMemory Cache候補。

### go-cache

シンプルなTTL Memory Cache。

### valkey-go

Valkey用Driverの内部実装候補。

### rueidis

Redis/Valkey系の高性能クライアント候補。

### minio-go

S3互換Object Storage Driverの内部実装候補。

---

# 27. eko/gocache

Goには既に`eko/gocache`があり、

- 複数Cache Store
- Chain Cache
- Loadable Cache
- Tag
- Generics

など、かなり近い機能を持っている。

そのため実装時には設計・APIを参考にする価値が高い。

ただし今回の構想では、

```text
gocache
    ↓
複数のCache Storeを組み合わせる

今回のライブラリ
    ↓
Cache Typeとして
キャッシュ戦略そのものを定義する
```

という思想の違いを明確にする。

---

# 28. ライブラリ全体の構造

概念的には、

```text
Cache
│
├── CacheType
│   ├── KeyBuilder
│   ├── Layer[]
│   ├── Policy
│   ├── Invalidation
│   └── Loader
│
├── Layer
│   ├── Memory
│   ├── Valkey
│   ├── ObjectStorage
│   ├── Firestore
│   └── ...
│
├── Policy
│   ├── TTL
│   ├── SWR
│   ├── SIE
│   └── StampedeProtection
│
├── Key
│   └── KeyBuilder
│
└── Invalidation
    ├── Exact
    ├── Prefix
    └── Tag
```

---

# 29. 目指すAPI

最終的には、利用者が意識するのはCache Type。

```go
productCache.Get(ctx, key)

productCache.Set(ctx, key, product)

productCache.Delete(ctx, key)

productCache.Purge(ctx, ...)

productCache.GetOrLoad(ctx, key)
```

など。

内部では、

```text
Cache Type
     │
     ▼
Key generation
     │
     ▼
L1
     │ miss
     ▼
L2
     │ miss
     ▼
L3
     │ miss
     ▼
Loader
     │
     ▼
populate layers
```

をライブラリ側が処理する。

---

# 30. 設計上の重要なポイント

## ① Driverを中心にしない

「Redis Cache Library」ではなく、

> Cache Strategy Library

を目指す。

---

## ② Layer数を固定しない

```text
L1 → L2
```

だけではなく、

```text
L1 → L2 → L3
```

や、

```text
L1 → L2 → L3 → L4
```

も可能にする。

---

## ③ Object Storageを第一級のLayerにする

従来のCache Libraryでは、

```text
Memory
Redis
Memcached
```

が中心になりやすい。

このライブラリでは、

```text
Memory
Valkey
Firestore
S3/R2/GCS
```

などを同列のLayerとして扱う。

---

## ④ ビジネスキーを重視する

単なる、

```text
cache.Get("abc")
```

ではなく、

```text
tenant
product
locale
version
user
permissions
```

など、アプリケーションのコンテキストを反映したキーを作れるようにする。

---

## ⑤ Cache Typeをアプリケーションの設計単位にする

例えば、

```text
ProductCache
SessionCache
ArticleCache
SearchResultCache
DashboardCache
```

という形で、アプリケーションの設計に自然に組み込めるようにする。

---

# 31. 最初のMVP

最初から全部実装しない。

まずは、

```text
Cache Type
    │
    ├── Key Builder
    │
    ├── Layer[]
    │
    │   ├── Memory
    │   └── Valkey
    │
    ├── TTL
    │
    └── Loader
```

まで。

つまり、

```text
L1 Memory
    ↓
L2 Valkey
    ↓
Loader
```

を完成させる。

その後、

```text
Object Storage
Firestore
Tag Invalidation
SWR
Stale-if-error
Stampede Protection
```

を追加する。

---

# 32. 最終的な位置付け

このライブラリの本質は、

```text
Redis abstraction
```

ではない。

また、

```text
Memory cache abstraction
```

でもない。

目指すものは、

```text
                 Application
                      │
                      ▼
                ┌───────────┐
                │ Cache Type│
                └─────┬─────┘
                      │
          ┌───────────┼───────────┐
          ▼           ▼           ▼
       Key/Policy   Layering   Invalidation
                      │
              ┌───────┼───────┐
              ▼       ▼       ▼
           Memory   Valkey    S3/R2
                      │
                      ▼
                  DB / API
```

という、

> **アプリケーションがキャッシュ戦略を宣言し、その戦略を複数のストレージにまたがって実行するためのライブラリ**

とする。

Goファーストで実装し、将来的にTypeScript/PHPなどへ同じ概念を移植できるよう、**Cache Type / Layer / Policy / Key / Invalidationのセマンティクスを明確に分離する**。

---

以上
