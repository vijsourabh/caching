package caching

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Value string
}

var (
	cacheExpiry        = 5
	cacheCleanInterval = 5
	cacheKey           = "key"
	cacheValue         = &testStruct{
		Value: "value",
	}
)

func TestService_CacheWithExpiry(test *testing.T) {
	test.Run("get entry in the cache", func(test *testing.T) {
		cache := NewCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		defer cache.Remove(cacheKey)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)
	})

	test.Run("not able to get entry from after removal", func(test *testing.T) {
		cache := NewCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		cache.Remove(cacheKey)

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("not able to get expired entry", func(test *testing.T) {
		expiryTime := 1

		cache := NewCacheWithExpiry(time.Second*time.Duration(expiryTime), time.Second*time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(expiryTime))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("get invalid entry from cache", func(test *testing.T) {
		cache := NewCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cacheCleanInterval))

		value, _, found := cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("adding same key in cache multiple times with different value", func(test *testing.T) {
		cache := NewCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		newValue := &testStruct{
			Value: "newValue",
		}
		_ = cache.Add(cacheKey, newValue)

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, newValue.Value, expectedValue.Value)
	})

	test.Run("fetch the value after cleaning of cache", func(test *testing.T) {
		cleanInterval := 1

		cache := NewCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)
	})

	test.Run("unable to fetch the expired value after cleaning of cache", func(test *testing.T) {
		cleanInterval := 1
		expiry := 2

		cache := NewCacheWithExpiry(time.Second*time.Duration(expiry), time.Second*time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("the entry gets expired if expiry of the cache is decreased", func(test *testing.T) {
		cleanInterval := 1
		expiry := 10

		cache := NewCacheWithExpiry(time.Second*time.Duration(expiry), time.Second*time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		cache.UpdateExpiry(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("the entry will not get expired if expiry of the cache is increased", func(test *testing.T) {
		cleanInterval := 3
		expiry := 1

		cache := NewCacheWithExpiry(time.Second*time.Duration(expiry), time.Second*time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(expiry-1))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		cache.UpdateExpiry(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

}

func TestService_Cache(test *testing.T) {
	test.Run("get entry in the cache", func(test *testing.T) {
		cache := NewCache(time.Second * time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		defer cache.Remove(cacheKey)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)
	})

	test.Run("not able to get entry from after removal", func(test *testing.T) {
		cache := NewCache(time.Second * time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		cache.Remove(cacheKey)

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("get invalid entry from cache", func(test *testing.T) {
		cache := NewCache(time.Second * time.Duration(cacheCleanInterval))

		value, _, found := cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("adding same key in cache multiple times with different value", func(test *testing.T) {
		cache := NewCache(time.Second * time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		newValue := &testStruct{
			Value: "newValue",
		}
		_ = cache.Add(cacheKey, newValue)

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, newValue.Value, expectedValue.Value)
	})

	test.Run("fetch the value after cleaning of cache", func(test *testing.T) {
		cleanInterval := 1

		cache := NewCache(time.Second * time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)
	})

	test.Run("entry gets expired after cleaning of cache if the expiry is updated", func(test *testing.T) {
		cleanInterval := 1

		cache := NewCache(time.Second * time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		cache.UpdateExpiry(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})
}

func TestService_ObfuscatedCacheWithExpiry(test *testing.T) {
	test.Run("get entry in the cache", func(test *testing.T) {
		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		defer cache.Remove(cacheKey)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)
	})

	test.Run("not able to get entry from after removal", func(test *testing.T) {
		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		cache.Remove(cacheKey)

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("not able to get expired entry", func(test *testing.T) {
		expiryTime := 1

		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(expiryTime), time.Second*time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(expiryTime))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("get invalid entry from cache", func(test *testing.T) {
		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cacheCleanInterval))

		value, _, found := cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("adding same key in cache multiple times with different value", func(test *testing.T) {
		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		newValue := &testStruct{
			Value: "newValue",
		}
		_ = cache.Add(cacheKey, newValue)

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, newValue.Value, expectedValue.Value)
	})

	test.Run("fetch the value after cleaning of cache", func(test *testing.T) {
		cleanInterval := 1

		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(cacheExpiry), time.Second*time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)
	})

	test.Run("unable to fetch the expired value after cleaning of cache", func(test *testing.T) {
		cleanInterval := 1
		expiry := 2

		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(expiry), time.Second*time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("the entry gets expired if expiry of the cache is decreased", func(test *testing.T) {
		cleanInterval := 1
		expiry := 10

		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(expiry), time.Second*time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		cache.UpdateExpiry(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("the entry will not get expired if expiry of the cache is increased", func(test *testing.T) {
		cleanInterval := 3
		expiry := 1

		cache := NewObfuscatedCacheWithExpiry(time.Second*time.Duration(expiry), time.Second*time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(expiry-1))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		cache.UpdateExpiry(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})
}

func TestService_ObfuscatedCach(test *testing.T) {
	test.Run("get entry in the cache", func(test *testing.T) {
		cache := NewObfuscatedCache(time.Second * time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		defer cache.Remove(cacheKey)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)
	})

	test.Run("not able to get entry from after removal", func(test *testing.T) {
		cache := NewObfuscatedCache(time.Second * time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		cache.Remove(cacheKey)

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("get invalid entry from cache", func(test *testing.T) {
		cache := NewObfuscatedCache(time.Second * time.Duration(cacheCleanInterval))

		value, _, found := cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})

	test.Run("adding same key in cache multiple times with different value", func(test *testing.T) {
		cache := NewObfuscatedCache(time.Second * time.Duration(cacheCleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		newValue := &testStruct{
			Value: "newValue",
		}
		_ = cache.Add(cacheKey, newValue)

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, newValue.Value, expectedValue.Value)
	})

	test.Run("fetch the value after cleaning of cache", func(test *testing.T) {
		cleanInterval := 1

		cache := NewObfuscatedCache(time.Second * time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)
	})

	test.Run("entry gets expired after cleaning of cache if the expiry is updated", func(test *testing.T) {
		cleanInterval := 1

		cache := NewObfuscatedCache(time.Second * time.Duration(cleanInterval))

		err := cache.Add(cacheKey, cacheValue)
		require.NoError(test, err)

		value, _, found := cache.Get(cacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.True(test, found)

		err = json.Unmarshal(value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, cacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		cache.UpdateExpiry(time.Second * time.Duration(cleanInterval))

		value, _, found = cache.Get(cacheKey)
		require.False(test, found)
		require.Nil(test, value)
	})
}
