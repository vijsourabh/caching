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
		Value interface{}
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

// GetAllCacheInfo  fetch the all cache info
func (cache *Cache) GetAllCacheInfo() map[interface{}]*GetCacheResponse {
	res := make(map[interface{}]*GetCacheResponse)
	cache.cacheMap.Range(func(key, value interface{}) bool {
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

// addInCache adds the value in the cache for the provided key
// It also obfuscates the value if cache is obfuscated
func (cache *Cache) addInCache(key interface{}, value *cacheEntry) error {
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

// nolint
func (cache *Cache) get(key interface{}, value interface{}) (*GetCacheResponse, bool) {
	valueFromCache, found := cache.cacheMap.Load(key)
	if !found {
		return nil, false
	}

	entry, ok := valueFromCache.(*cacheEntry)
	if !ok {
		cache.Remove(key)
		return nil, false
	}

	if entry.expiry <= defaultExpiry || time.Since(entry.insertionTime) <= entry.expiry {
		var err error

		if cache.obfuscator != nil {
			insertedValue := entry.value.([]byte)
			insertedValue, err = cache.obfuscator.Deobfuscate(insertedValue)
			if err != nil {
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

		return &GetCacheResponse{
			Value: entry.value,
		}, true
	}

	// since the entry in the cache is expired, so removing it from cache
	cache.Remove(key)

	return nil, false
}

func (cache *Cache) Get(key interface{}, value interface{}) error {
	if _, found := cache.get(key, value); !found {
		return errors.New("key not found in the cache")
	}

	return nil
}

func (cache *Cache) GetValue(key interface{}) (interface{}, error) {
	cachedValue, found := cache.get(key, nil)
	if !found {
		return nil, errors.New("key not found in the cache")
	}

	return cachedValue.Value, nil
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
