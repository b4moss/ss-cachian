/**
 * @b4moss/ss-cachian — Node.js / TypeScript port of ss-cachian (Go v0.7.0 parity).
 */

export const SSCACHIAN_RUNTIME = "node" as const;

export {
  ErrEmptyKey,
  ErrNegativeTTL,
  ErrNotInteger,
  ErrIncrOverflow,
  ErrNoLayer,
  ErrNoKeyBuilder,
  ErrInvalidVersion,
  ErrNoLoader,
  ErrTypeMismatch,
  ErrInvalidContext,
  ErrCorruptVersion,
  ErrInvalidLayerIndex,
  SscachianError,
  isVersionDataKey,
  defaultKeyBuilder,
  type Entry,
  type Layer,
  type DurationMs,
  type KeyContext,
  type KeyBuilder,
  type Loader,
  type TypeGuard,
} from "./types.js";

export { define, Builder, CacheType } from "./cache.js";

export { MemoryStore, newMemoryStore } from "./driver/memory.js";
export {
  FirestoreStore,
  newFirestoreStore,
  type FirestoreOptions,
} from "./driver/firestore.js";
