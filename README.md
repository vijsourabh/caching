# Caching

A thread-safe, in-memory key-value cache for Go with configurable TTL, automatic background eviction, and optional transparent AES-256-GCM value obfuscation.

---

## Features

- **Generic key/value store** — keys and values are `any`; works with structs, primitives, pointers, and CGo objects
- **Configurable TTL** — set a cache-wide default expiry, override it per entry
- **Background cleanup** — a goroutine evicts expired entries on a configurable interval
- **Lazy eviction** — `Get` also checks expiry on access, so stale values are never returned even before the cleaner fires
- **Optional AES-256-GCM obfuscation** — values are JSON-encoded and encrypted in memory; the key is ephemeral per cache instance
- **Runtime TTL updates** — change expiry and clean interval live; new entries use the new values immediately
- **External locking primitives** — exported `Lock/Unlock/RLock/RUnlock` for coordinating multi-step operations atomically
- **Zero external dependencies** — only the Go standard library (obfuscation uses `crypto/aes` + `crypto/cipher`)

---

## Installation

```bash
go get github.com/vijsourabh/caching
```

---

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/vijsourabh/caching"
)

type User struct {
    Name string
    Age  int
}

func main() {
    // Create a cache: entries expire after 5 minutes, cleaner runs every 1 minute
    c := caching.NewCache(&caching.CreateCacheParams{
        Expiry:        5 * time.Minute,
        CleanInterval: 1 * time.Minute,
    })

    // Add an entry
    err := c.Add(&caching.AddCacheParams{
        Key:   "user:42",
        Value: &User{Name: "Alice", Age: 30},
    })
    if err != nil {
        panic(err)
    }

    // Retrieve it
    var user User
    if err := c.Get("user:42", &user); err != nil {
        fmt.Println("not found:", err)
    } else {
        fmt.Printf("Found: %+v\n", user) // Found: {Name:Alice Age:30}
    }

    // Remove a specific entry
    c.Remove("user:42")

    // Drain and stop the cache (cancels background goroutine)
    c.Clean()
}
```

---

## API Reference

### Creating a Cache

```go
func NewCache(params *CreateCacheParams) *Cache
```

Allocates a `Cache` and starts the background cleanup goroutine.

| `CreateCacheParams` field | Type | Description |
|---|---|---|
| `Expiry` | `time.Duration` | Default TTL for all entries. Set to `0` or negative for no expiry. |
| `CleanInterval` | `time.Duration` | How often the background goroutine scans for and removes expired entries. |
| `IsCacheObfuscated` | `bool` | If `true`, values are AES-256-GCM encrypted before storage (see [Obfuscation](#obfuscation)). |

---

### Adding an Entry

```go
func (cache *Cache) Add(params *AddCacheParams) error
```

Inserts or overwrites a key. The per-entry `Expiry` overrides the cache-wide default when set to a positive value.

| `AddCacheParams` field | Type | Description |
|---|---|---|
| `Key` | `any` | Cache key — any comparable value. |
| `Value` | `any` | Value to store. Must be JSON-serializable when obfuscation is enabled. |
| `Expiry` | `time.Duration` | Per-entry TTL override. Ignored if ≤ 0 (falls back to cache-wide default). |

---

### Retrieving an Entry

```go
func (cache *Cache) Get(key any, value any) error
```

Populates `value` with the cached data. Returns an error if the key does not exist or has expired.

- **Obfuscated cache**: pass a pointer to a concrete type (e.g. `*User`); the value is JSON-unmarshalled into it.
- **Non-obfuscated cache**: pass a `*any` to receive the stored value as-is, or a typed pointer for non-JSON types (e.g. CGo cipher objects).

```go
// Typed retrieval (non-obfuscated)
var u User
err := c.Get("user:42", &u)

// any retrieval (non-obfuscated)
var raw any
err := c.Get("user:42", &raw)
```

---

### Updating an Entry's Value

```go
func (cache *Cache) Update(params *UpdateCacheParams) error
```

Updates the value of an existing key **without** resetting its insertion time or expiry. Returns an error if the key does not exist.

| `UpdateCacheParams` field | Type | Description |
|---|---|---|
| `Key` | `any` | Key to update. |
| `Value` | `any` | New value. |

---

### Removing an Entry

```go
func (cache *Cache) Remove(key any)
```

Immediately deletes the entry for `key`. No-op if the key does not exist.

---

### Retrieving All Live Entries

```go
func (cache *Cache) GetAllCacheInfo() map[any]*GetCacheResponse
```

Returns a snapshot of all non-expired entries as a `map[key → *GetCacheResponse]`. Returns `nil` if the cache is empty or all entries have expired.

```go
type GetCacheResponse struct {
    Value any
}
```

---

### Updating TTL and Clean Interval at Runtime

```go
func (cache *Cache) UpdateTime(params *UpdateCacheTimeParams)
```

Changes the cache-wide expiry and/or clean interval. Only entries added **after** this call inherit the new expiry; existing entries keep the TTL they were inserted with.

| `UpdateCacheTimeParams` field | Type | Description |
|---|---|---|
| `Expiry` | `time.Duration` | New default TTL for future entries. |
| `CleanInterval` | `time.Duration` | New background cleaner interval. |

---

### Clearing the Cache

```go
func (cache *Cache) Clean()
```

- Cancels the background cleanup goroutine.
- Deletes all entries.
- Destroys the obfuscation key (if obfuscation was enabled).

> ⚠️ After `Clean()`, the cache is no longer usable. Create a new instance with `NewCache` if needed.

---

### External Locking

The cache exposes a `sync.RWMutex` for callers who need to perform multi-step read/write sequences atomically:

```go
func (cache *Cache) Lock()
func (cache *Cache) Unlock()
func (cache *Cache) RLock()
func (cache *Cache) RUnlock()
```

> **Note:** `Add`, `Get`, `Update`, and `Remove` are individually safe for concurrent use via the underlying `sync.Map`. The external mutex is only needed when you need to coordinate multiple cache operations as a single atomic unit.

---

## Expiry Behaviour

Understanding exactly when entries expire is important for correct usage:

| Scenario | Behaviour |
|---|---|
| `NewCache` with `Expiry ≤ 0` | Entries never expire (unless a per-entry expiry is set via `Add`) |
| `Add` with `Expiry ≤ 0` | Inherits the cache-wide expiry |
| `Add` with `Expiry > 0` | Uses the per-entry expiry, overriding the cache-wide default |
| `UpdateTime` called after entries exist | Existing entries keep their original expiry; only new entries use the updated value |
| Expired entry on `Get` | Entry is lazily deleted and `Get` returns an error |
| Expired entry on background sweep | Entry is proactively deleted after the next `CleanInterval` tick |

```go
// Example: per-entry expiry override
c := caching.NewCache(&caching.CreateCacheParams{
    Expiry:        10 * time.Minute, // default: 10 min
    CleanInterval: 1 * time.Minute,
})

// This entry expires in 30 seconds, not 10 minutes
c.Add(&caching.AddCacheParams{
    Key:    "short-lived",
    Value:  "data",
    Expiry: 30 * time.Second,
})
```

---

## Obfuscation

When `IsCacheObfuscated: true`, all values stored in the cache are:

1. **JSON-marshalled** (`encoding/json`)
2. **Encrypted** with AES-256-GCM using a randomly generated 32-byte key (unique per `Cache` instance)
3. Stored as `nonce | ciphertext | GCM tag`

On retrieval, the process is reversed: decrypt → JSON-unmarshal into the destination pointer.

```go
type Secret struct {
    Token string
}

c := caching.NewCache(&caching.CreateCacheParams{
    Expiry:            5 * time.Minute,
    CleanInterval:     1 * time.Minute,
    IsCacheObfuscated: true,
})

c.Add(&caching.AddCacheParams{
    Key:   "token",
    Value: &Secret{Token: "s3cr3t"},
})

var s Secret
c.Get("token", &s)
fmt.Println(s.Token) // s3cr3t
```

**Constraints:**

- Values must be JSON-serializable (`json.Marshal` must succeed).
- The encryption key lives only in memory with the `Cache` instance — calling `Clean()` destroys it permanently.
- An obfuscated cache cannot store non-JSON types (e.g. raw CGo pointers); use a non-obfuscated cache for those.

---

## Thread Safety

| Concern | Mechanism |
|---|---|
| Concurrent `Add` / `Get` / `Remove` | `sync.Map` — safe without external locking |
| Concurrent `Update` (read-modify-write) | Safe: `Update` uses `sync.Map.Load` + `sync.Map.Store` on the same `*cacheEntry` pointer |
| Background goroutine vs. foreground ops | `sync.Map.Range` and individual `Delete`/`Store` calls are all safe concurrently |
| Multi-step atomic sequences | Use the exported `Lock/Unlock` or `RLock/RUnlock` |

---

## Development

Dependencies are vendored. Use the provided `Makefile` targets:

| Target | Description |
|---|---|
| `make prep` | Full pipeline: vendor → tools → fmt → lint → vet → cover |
| `make test` | Run all tests (`go test ./...`) |
| `make cover` | Run tests with coverage; outputs HTML + function-level report to `build/` |
| `make lint` | Run `golangci-lint` (5-minute timeout) |
| `make fmt` | Format all Go source with `go fmt` |
| `make vet` | Run `go vet ./...` |
| `make vendor` | Re-vendor dependencies (`go mod tidy && go mod vendor`) |
| `make update` | Upgrade all dependencies, tidy, and re-vendor |
| `make tools` | Install `cover` and `golangci-lint` |
| `make clean` | Remove `build/` artifacts |

```bash
# Run the full check suite
make prep

# Run tests only
make test
```
