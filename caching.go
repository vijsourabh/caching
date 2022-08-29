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
	}
)

// NewCache creates a cache Instance and triggers a goroutine to Clean the cache
func NewCache(expiry time.Duration, cleanInterval time.Duration) *Cache {
	cache := &Cache{
		cacheMap:      sync.Map{},
		expiry:        expiry,
		cleanInterval: cleanInterval,
	}

	// call goroutine to clean cache
	go cache.clean()

	return cache
}

// NewObfuscatedCache creates an obfuscated cache Instance and triggers a goroutine to Clean the cache
// The cache will obfuscate the input before adding it in the map
func NewObfuscatedCache(expiry time.Duration, cleanInterval time.Duration) *Cache {
	cache := &Cache{
		cacheMap:      sync.Map{},
		expiry:        expiry,
		cleanInterval: cleanInterval,
		obfuscator:    NewObfuscator(),
	}

	// call goroutine to clean cache
	go cache.clean()

	return cache
}

// Add a value to the cache.
// The value must be a pointer to a json struct
func (cache *Cache) Add(key interface{}, value interface{}) error {
	insertValue, err := json.Marshal(&value)
	if err != nil {
		return err
	}

	if cache.obfuscator != nil {
		if insertValue, err = cache.obfuscator.Obfuscate(insertValue); err != nil {
			return err
		}
	}

	cache.cacheMap.Store(key, &cacheEntry{
		value:         insertValue,
		insertionTime: time.Now(),
	})

	return nil
}

// Get looks up a key's value from the cache.
func (cache *Cache) Get(key interface{}) ([]byte, time.Time, bool) {
	value, found := cache.cacheMap.Load(key)
	if !found {
		return nil, time.Time{}, false
	}

	entry, ok := value.(*cacheEntry)
	if !ok {
		cache.Remove(key)
		return nil, time.Time{}, false
	}

	if cache.expiry < 0 || time.Since(entry.insertionTime) <= cache.expiry {
		insertedValue := entry.value.([]byte)
		var err error

		// deobfuscate the entry if cache is obfuscated
		if cache.obfuscator != nil {
			insertedValue, err = cache.obfuscator.Deobfuscate(insertedValue)
			if err != nil {
				cache.Remove(key)
				return nil, time.Time{}, false
			}
		}

		return insertedValue, entry.insertionTime, true
	}

	// since the entry in the cache is expired, so removing it from cache
	cache.Remove(key)

	return nil, time.Time{}, false
}

// Remove the provided key from the cache.
func (cache *Cache) Remove(key interface{}) {
	cache.cacheMap.Delete(key)
}

// clean removes the expired entries from the cache after a given interval
func (cache *Cache) clean() {
	if cache.expiry < 0 {
		return
	}

	// infinite loop
	for {
		time.Sleep(cache.cleanInterval)

		cache.cacheMap.Range(func(key, value interface{}) bool {
			entry, ok := value.(*cacheEntry)

			if ok && time.Since(entry.insertionTime) > cache.expiry {
				cache.Remove(key)
				return true
			}

			return false
		})
	}
}
