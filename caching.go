package caching

import (
	"encoding/json"
	"sync"
	"time"
)

type (
	Cache struct {
		cacheMap      sync.Map
		expiry        time.Duration
		cleanInterval time.Duration
		obfuscator    *Obfuscator
	}

	cacheEntry struct {
		value         interface{}
		insertionTime time.Time
		expiry        time.Duration
	}

	CreateCacheParams struct {
		Expiry            time.Duration
		CleanInterval     time.Duration
		IsCacheObfuscated bool
	}

	AddCacheParams struct {
		Key    interface{}
		Value  interface{}
		Expiry time.Duration
	}

	GetCacheResponse struct {
		Value         []byte
		InsertionTime time.Time
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

// UpdateExpiry updates the expiry time of the cache.
func (cache *Cache) UpdateExpiry(expiry time.Duration) {
	cache.expiry = expiry
}

// Add a value to the cache and the expiry time of the entry will be overridden.
// The value must be a pointer to a json struct
func (cache *Cache) Add(params *AddCacheParams) error {
	insertValue, err := json.Marshal(&params.Value)
	if err != nil {
		return err
	}

	if cache.obfuscator != nil {
		if insertValue, err = cache.obfuscator.Obfuscate(insertValue); err != nil {
			return err
		}
	}

	entry := &cacheEntry{
		value:         insertValue,
		expiry:        cache.expiry,
		insertionTime: time.Now(),
	}

	// override the expiry for the key provided by the user
	if params.Expiry > 0 {
		entry.expiry = params.Expiry
	}

	cache.cacheMap.Store(params.Key, entry)

	return nil
}

// Get looks up a key's value from the cache.
func (cache *Cache) Get(key interface{}) (*GetCacheResponse, bool) {
	value, found := cache.cacheMap.Load(key)
	if !found {
		return nil, false
	}

	entry, ok := value.(*cacheEntry)
	if !ok {
		cache.Remove(key)
		return nil, false
	}

	if entry.expiry <= defaultExpiry || time.Since(entry.insertionTime) <= entry.expiry {
		insertedValue := entry.value.([]byte)
		var err error

		// deobfuscate the entry if cache is obfuscated
		if cache.obfuscator != nil {
			insertedValue, err = cache.obfuscator.Deobfuscate(insertedValue)
			if err != nil {
				cache.Remove(key)
				return nil, false
			}
		}

		return &GetCacheResponse{
			Value:         insertedValue,
			InsertionTime: entry.insertionTime,
		}, true
	}

	// since the entry in the cache is expired, so removing it from cache
	cache.Remove(key)

	return nil, false
}

// Remove the provided key from the cache.
func (cache *Cache) Remove(key interface{}) {
	cache.cacheMap.Delete(key)
}

// clean removes the expired entries from the cache after a given interval
func (cache *Cache) clean() {
	// infinite loop
	for {
		time.Sleep(cache.cleanInterval)

		cache.cacheMap.Range(func(key, value interface{}) bool {
			entry, ok := value.(*cacheEntry)

			if ok && entry.expiry != defaultExpiry && time.Since(entry.insertionTime) > entry.expiry {
				cache.Remove(key)
				return true
			}

			return false
		})
	}
}
