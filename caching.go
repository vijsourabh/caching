package caching

import (
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

	UpdateCacheParams struct {
		Key   interface{}
		Value interface{}
	}

	UpdateCacheTimeParams struct {
		Expiry        time.Duration
		CleanInterval time.Duration
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

// UpdateTime updates the expiry time of the cache.
func (cache *Cache) UpdateTime(params *UpdateCacheTimeParams) {
	cache.expiry = params.Expiry
	cache.cleanInterval = params.CleanInterval
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

// addInCache adds the value in the cache for the provided key
// It also obfuscates the value if cache is obfuscated
func (cache *Cache) addInCache(key interface{}, value *cacheEntry) error {
	insertValue, err := json.Marshal(&value.value)
	if err != nil {
		return err
	}

	if cache.obfuscator != nil {
		if insertValue, err = cache.obfuscator.Obfuscate(insertValue); err != nil {
			return err
		}
	}

	value.value = insertValue

	cache.cacheMap.Store(key, value)

	return nil
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
