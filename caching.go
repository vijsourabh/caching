package caching

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type (
	Cache struct {
		cacheMap      sync.Map
		expiry        time.Duration
		cleanInterval time.Duration
		obfuscator    *Obfuscator
		lock          sync.RWMutex
		intervalCh    chan time.Duration // signals cleanInterval changes from UpdateTime
		cacheCtx
	}

	// cacheCtx stores ctx and cancelFunc to stop the routines
	cacheCtx struct {
		ctx        context.Context
		cancelFunc context.CancelFunc
	}

	cacheEntry struct {
		value         any
		insertionTime time.Time
		expiry        time.Duration
	}

	CreateCacheParams struct {
		Expiry            time.Duration
		CleanInterval     time.Duration
		IsCacheObfuscated bool
	}

	AddCacheParams struct {
		Key    any
		Value  any
		Expiry time.Duration
	}

	UpdateCacheParams struct {
		Key   any
		Value any
	}

	UpdateCacheTimeParams struct {
		Expiry        time.Duration
		CleanInterval time.Duration
	}

	GetCacheResponse struct {
		Value any
	}
)

const (
	defaultExpiry = -1
)

// NewCache creates a cache Instance and triggers a goroutine to Clean the cache on the basis of provided cleanInterval.
func NewCache(params *CreateCacheParams) *Cache {
	cache := &Cache{
		cacheMap:      sync.Map{},
		cleanInterval: params.CleanInterval,
		expiry:        defaultExpiry,
	}

	cache.intervalCh = make(chan time.Duration, 1)

	cache.cacheCtx.ctx, cache.cacheCtx.cancelFunc = context.WithCancel(context.Background())

	// override the expiry provided by the user
	if params.Expiry > 0 {
		cache.expiry = params.Expiry
	}

	if params.IsCacheObfuscated {
		cache.obfuscator = NewObfuscator()
	}

	// call goroutine to clean cache
	go cache.clean()

	return cache
}

// UpdateTime updates the global expiry and, when CleanInterval > 0, the
// cleanInterval of the background cleaner goroutine.
// Callers performing concurrent Add/Get alongside UpdateTime must hold the
// write Lock() before calling this method to avoid a data race on expiry.
func (cache *Cache) UpdateTime(params *UpdateCacheTimeParams) {
	cache.expiry = params.Expiry

	if params.CleanInterval > 0 {
		cache.cleanInterval = params.CleanInterval
		// Notify the clean() goroutine immediately so it resets the ticker
		// without waiting for the current tick to expire. Non-blocking send:
		// if the channel already holds a pending value the goroutine will
		// read the latest cleanInterval from cache.cleanInterval directly.
		select {
		case cache.intervalCh <- params.CleanInterval:
		default:
		}
	}
}

// GetAllCacheInfo returns all non-expired cache entries.
// Returns an empty (non-nil) map when the cache holds no live entries,
// so callers can range over the result without a nil-check.
func (cache *Cache) GetAllCacheInfo() map[any]*GetCacheResponse {
	res := make(map[any]*GetCacheResponse)
	cache.cacheMap.Range(func(key, value any) bool {
		insertedVal, found := cache.get(key, value)
		if found {
			res[key] = insertedVal
		}

		return true
	})

	return res
}

// Update updates the value for the cache
func (cache *Cache) Update(params *UpdateCacheParams) error {
	value, found := cache.cacheMap.Load(params.Key)
	if !found {
		return errors.New("value doesn't exist in cache")
	}

	entry, ok := value.(*cacheEntry)
	if !ok {
		cache.Remove(params.Key)

		return errors.New("invalid value found in cache")
	}

	entry.value = params.Value

	return cache.addInCache(params.Key, entry)
}

// Add stores a value in the cache. If the key already exists it is overwritten.
// Per-key Expiry overrides the cache-level expiry when > 0.
// Callers that concurrently call UpdateTime must hold RLock() before calling
// Add to avoid a data race on the cache-level expiry field.
func (cache *Cache) Add(params *AddCacheParams) error {
	value := &cacheEntry{
		value:         params.Value,
		expiry:        cache.expiry,
		insertionTime: time.Now(),
	}

	// override the expiry for the key provided by the user
	if params.Expiry > 0 {
		value.expiry = params.Expiry
	}

	return cache.addInCache(params.Key, value)
}

func (cache *Cache) Get(key any, value any) error {
	if _, found := cache.get(key, value); !found {
		return errors.New("key not found in the cache")
	}

	return nil
}

// Remove the provided key from the cache.
func (cache *Cache) Remove(key any) {
	cache.cacheMap.Delete(key)
}

// Clean cancels the background cleaner goroutine, wipes all cached entries,
// and removes the obfuscator.
//
// IMPORTANT: Clean is a terminal operation. The cache MUST NOT be used after
// calling Clean; the background cleaner goroutine will not restart. Create a
// fresh instance with NewCache if further caching is required.
func (cache *Cache) Clean() {
	cache.cacheCtx.cancelFunc()
	cache.cacheMap.Clear()

	cache.obfuscator = nil
}

func (cache *Cache) RLock() {
	cache.lock.RLock()
}

func (cache *Cache) RUnlock() {
	cache.lock.RUnlock()
}

func (cache *Cache) Lock() {
	cache.lock.Lock()
}

func (cache *Cache) Unlock() {
	cache.lock.Unlock()
}

// addInCache adds the value in the cache for the provided key
// It also obfuscates the value if cache is obfuscated
func (cache *Cache) addInCache(key any, value *cacheEntry) error {
	if cache.obfuscator != nil {
		insertValue, err := json.Marshal(&value.value)
		if err != nil {
			return err
		}

		if value.value, err = cache.obfuscator.Obfuscate(insertValue); err != nil {
			return err
		}
	}

	cache.cacheMap.Store(key, value)

	return nil
}

// clean removes the expired entries from the cache after a given interval.
//
// Uses time.NewTicker instead of time.After to avoid allocating a new timer on
// every iteration. UpdateTime signals interval changes via intervalCh so the
// ticker is reset immediately — no waiting for the current tick to expire.
//
// If no cleanInterval is configured at construction, the goroutine blocks on
// intervalCh until UpdateTime provides one, keeping the goroutine alive without
// risking a time.NewTicker(0) panic.
func (cache *Cache) clean() {
	interval := cache.cleanInterval

	// No interval configured yet — block until UpdateTime provides one or the
	// cache is shut down.
	if interval <= 0 {
		select {
		case <-cache.ctx.Done():
			return
		case interval = <-cache.intervalCh:
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-cache.ctx.Done():
			return

		// UpdateTime sent a new interval — reset the ticker immediately so
		// the change takes effect on the very next cycle.
		case newInterval := <-cache.intervalCh:
			if newInterval > 0 && newInterval != interval {
				interval = newInterval
				ticker.Reset(interval)
			}

		case <-ticker.C:
			cache.cacheMap.Range(func(key, value any) bool {
				entry, ok := value.(*cacheEntry)

				// Use > 0 (not != defaultExpiry) so a zero-duration expiry is
				// treated as "no expiry" rather than "immediately expired".
				if ok && entry.expiry > 0 && time.Since(entry.insertionTime) > entry.expiry {
					cache.Remove(key)
				}

				// Always return true to continue iterating over all entries.
				// Returning false would stop Range after the first non-expired key.
				return true
			})
		}
	}
}

func (cache *Cache) get(key any, value any) (*GetCacheResponse, bool) {
	valueFromCache, found := cache.cacheMap.Load(key)
	if !found {
		return nil, false
	}

	entry, ok := valueFromCache.(*cacheEntry)
	if !ok {
		cache.Remove(key)

		return nil, false
	}

	// Use > 0 (not > defaultExpiry) so a zero-duration expiry is treated as
	// "no expiry" rather than "immediately expired" (defaultExpiry is -1, so
	// > defaultExpiry would also be true for expiry == 0).
	if entry.expiry > 0 && time.Since(entry.insertionTime) > entry.expiry {
		cache.Remove(key)

		return nil, false
	}

	if cache.obfuscator == nil {
		// Populate *any dest for non-JSON types (e.g. CGo cipher objects).
		if ptr, ok := value.(*any); ok && ptr != nil {
			*ptr = entry.value
		}

		return &GetCacheResponse{
			Value: entry.value,
		}, true
	}

	var err error

	insertedValue := entry.value.([]byte)
	if insertedValue, err = cache.obfuscator.Deobfuscate(insertedValue); err != nil {
		cache.Remove(key)

		return nil, false
	}

	if value != nil {
		if err = json.Unmarshal(insertedValue, value); err != nil {
			return nil, false
		}
	}

	return &GetCacheResponse{
		Value: insertedValue,
	}, true
}
