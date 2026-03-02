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

// NewCache creates a cache Instance and triggers a goroutine to Clean the cache on the basis of provided cleanInterval
func NewCache(params *CreateCacheParams) *Cache {
	cache := &Cache{
		cacheMap:      sync.Map{},
		cleanInterval: params.CleanInterval,
		expiry:        defaultExpiry,
	}

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

// UpdateTime updates the expiry time of the cache.
func (cache *Cache) UpdateTime(params *UpdateCacheTimeParams) {
	cache.expiry = params.Expiry
	cache.cleanInterval = params.CleanInterval
}

// GetAllCacheInfo  fetch the all cache info
func (cache *Cache) GetAllCacheInfo() map[any]*GetCacheResponse {
	res := make(map[any]*GetCacheResponse)
	cache.cacheMap.Range(func(key, value any) bool {
		insertedVal, found := cache.get(key, value)
		if found {
			res[key] = insertedVal
		}

		return true
	})

	if len(res) > 0 {
		return res
	}

	return nil
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

// Add a value to the cache and the expiry time of the entry will be overridden.
// The value must be a pointer to a json struct
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

// clean removes the expired entries from the cache after a given interval
func (cache *Cache) clean() {
	for {
		select {
		case <-cache.ctx.Done():
			return

		case <-time.After(cache.cleanInterval):
			cache.cacheMap.Range(func(key, value any) bool {
				entry, ok := value.(*cacheEntry)

				if ok && entry.expiry != defaultExpiry && time.Since(entry.insertionTime) > entry.expiry {
					cache.Remove(key)

					return true
				}

				return false
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

	if entry.expiry > defaultExpiry && time.Since(entry.insertionTime) > entry.expiry {
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
