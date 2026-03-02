package caching

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ThalesGroup/flume/v2/flumetest"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Value string
}

var (
	testCacheExpiry        = 500
	testCacheCleanInterval = 500
	testCacheKey           = "key"
	testCacheValue         = &testStruct{
		Value: "value",
	}
)

func TestService_Cache(test *testing.T) {
	defer flumetest.Start(test)

	test.Run("get entry from the cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})
		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	test.Run("get all cache info from the cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		var err error
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})
		cacheKeyValue := make(map[string]string)
		cacheKeyValue["key1"] = "val1"
		cacheKeyValue["key2"] = "val2"
		cacheKeyValue["key3"] = "val3"
		cacheKeyValue["key4"] = "val4"
		for key, val := range cacheKeyValue {
			err = cache.Add(&AddCacheParams{
				Key:   key,
				Value: val,
			})
		}
		require.NoError(test, err)

		cachedInfo := cache.GetAllCacheInfo()
		require.Len(test, cachedInfo, len(cacheKeyValue))

		for key, value := range cachedInfo {
			cacheValue, found := cacheKeyValue[key.(string)]
			require.True(test, found)

			require.Equal(test, cacheValue, value.Value.(string))
		}
	})

	test.Run("fetch all cache after expiring the cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cacheExpiry := 5
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(cacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		var err error
		cacheKeyValue := make(map[string]string)
		cacheKeyValue["key1"] = "val11"
		cacheKeyValue["key2"] = "val12"
		cacheKeyValue["key3"] = "val13"
		cacheKeyValue["key4"] = "val14"
		for key, val := range cacheKeyValue {
			err = cache.Add(&AddCacheParams{
				Key:   key,
				Value: val,
			})
			require.NoError(test, err)
		}
		time.Sleep(time.Second * time.Duration(cacheExpiry))

		cachedInfo := cache.GetAllCacheInfo()
		// GetAllCacheInfo returns an empty map (not nil) when no live entries remain.
		require.Empty(test, cachedInfo)
	})

	test.Run("fetch cache info after expiring some keys from the cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cacheExpiry := 5
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(cacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		var err error
		cacheKeyValue := make(map[string]string)
		cacheKeyValue["key1"] = "val21"
		cacheKeyValue["key2"] = "val22"
		cacheKeyValue["key3"] = "val23"
		cacheKeyValue["key4"] = "val24"
		for key, val := range cacheKeyValue {
			err = cache.Add(&AddCacheParams{
				Key:   key,
				Value: val,
			})
			require.NoError(test, err)
		}

		time.Sleep(time.Second * time.Duration(cacheExpiry))

		cacheKeyValue2 := make(map[string]string)
		cacheKeyValue2["key5"] = "val25"
		cacheKeyValue2["key6"] = "val26"

		for key, val := range cacheKeyValue2 {
			err = cache.Add(&AddCacheParams{
				Key:   key,
				Value: val,
			})
			require.NoError(test, err)
		}

		cachedInfo := cache.GetAllCacheInfo()
		require.Len(test, cachedInfo, len(cacheKeyValue2))

		for key, value := range cachedInfo {
			cacheValue, found := cacheKeyValue2[key.(string)]
			require.True(test, found)

			require.Equal(test, cacheValue, value.Value.(string))
		}
	})

	test.Run("get entry from the obfuscated cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:            time.Second * time.Duration(testCacheExpiry),
			CleanInterval:     time.Second * time.Duration(testCacheCleanInterval),
			IsCacheObfuscated: true,
		})
		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value.([]byte), &expectedValue)
		require.NoError(test, err)
		require.Equal(test, expectedValue.Value, testCacheValue.Value)
	})

	test.Run("not able to get entry from cache after removal", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		cache.Remove(testCacheKey)

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("not able to get expired entry", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		expiryTime := 1
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(expiryTime),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(expiryTime))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("get invalid entry from cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})
		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("adding same key in cache multiple times with different value", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		newValue := &testStruct{
			Value: "newValue",
		}
		err = cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: newValue,
		})
		require.NoError(test, err)

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, newValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	test.Run("fetch the value after cleaning of cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cleanInterval := 1
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(cleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	test.Run("unable to fetch the expired value after cleaning of cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cleanInterval := 1
		expiry := 2
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(expiry),
			CleanInterval: time.Second * time.Duration(cleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("the entry shouldn't get expired even if expiry of the cache is decreased", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cleanInterval := 1
		expiry := 10
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(expiry),
			CleanInterval: time.Second * time.Duration(cleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		cache.UpdateTime(&UpdateCacheTimeParams{
			Expiry: time.Second * time.Duration(cleanInterval),
		})
		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	test.Run("the entry will get expired even if expiry of the cache is increased", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cleanInterval := 3
		expiry := 2
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(expiry),
			CleanInterval: time.Second * time.Duration(cleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(expiry-1))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		cache.UpdateTime(&UpdateCacheTimeParams{
			Expiry: time.Second * time.Duration(cleanInterval),
		})
		time.Sleep(time.Second * time.Duration(1))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("entry not get expired after cleaning if cache expiry is not set", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cleanInterval := 1
		cache := NewCache(&CreateCacheParams{
			CleanInterval: time.Second * time.Duration(cleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	test.Run("entry will get expired after cleaning if cache expiry is added later", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cleanInterval := 1
		expiry := 1
		cache := NewCache(&CreateCacheParams{
			CleanInterval: time.Second * time.Duration(cleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		cache.UpdateTime(&UpdateCacheTimeParams{
			Expiry: time.Second * time.Duration(expiry),
		})

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		// add the entry with new expiry
		newCacheKey := "abc"
		err = cache.Add(&AddCacheParams{
			Key:   newCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found = cache.get(newCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		getCachedValue, found = cache.get(newCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	test.Run("expire entry after overriding the value of cache expiry", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cleanInterval := 2
		expiry := 2
		cache := NewCache(&CreateCacheParams{
			CleanInterval: time.Second * time.Duration(cleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:    testCacheKey,
			Value:  testCacheValue,
			Expiry: time.Duration(expiry) * time.Second,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval-1))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("exception when updating an unknown key", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:            time.Second * time.Duration(testCacheExpiry),
			CleanInterval:     time.Second * time.Duration(testCacheCleanInterval),
			IsCacheObfuscated: true,
		})

		err := cache.Update(&UpdateCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.Error(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("update the value for a key", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:            time.Second * time.Duration(testCacheExpiry),
			CleanInterval:     time.Second * time.Duration(testCacheCleanInterval),
			IsCacheObfuscated: true,
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value.([]byte), &expectedValue)
		require.NoError(test, err)
		require.Equal(test, expectedValue.Value, testCacheValue.Value)

		testUpdatedCacheValue := &testStruct{
			Value: "updatedValue",
		}
		err = cache.Update(&UpdateCacheParams{
			Key:   testCacheKey,
			Value: testUpdatedCacheValue,
		})
		require.NoError(test, err)

		getUpdatedCacheValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)

		var expectedUpdatedValue testStruct
		err = json.Unmarshal(getUpdatedCacheValue.Value.([]byte), &expectedUpdatedValue)
		require.NoError(test, err)
		require.Equal(test, expectedUpdatedValue.Value, testUpdatedCacheValue.Value)
	})

	test.Run("Get string entry from the cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:            time.Second * time.Duration(testCacheExpiry),
			CleanInterval:     time.Second * time.Duration(testCacheCleanInterval),
			IsCacheObfuscated: true,
		})
		value := "aa"
		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: value,
		})
		require.NoError(test, err)

		var getValue string
		err = cache.Get(testCacheKey, &getValue)
		require.NoError(test, err)
		require.Equal(test, value, getValue)
	})

	test.Run("Get int entry from the cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:            time.Second * time.Duration(testCacheExpiry),
			CleanInterval:     time.Second * time.Duration(testCacheCleanInterval),
			IsCacheObfuscated: true,
		})
		value := 50
		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: value,
		})
		require.NoError(test, err)

		var getValue int
		err = cache.Get(testCacheKey, &getValue)
		require.NoError(test, err)
		require.Equal(test, value, getValue)
	})

	test.Run("Get string entry from non-obfuscated cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})
		value := "aa"
		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: value,
		})
		require.NoError(test, err)

		var cachedValue any
		err = cache.Get(testCacheKey, &cachedValue)
		require.NoError(test, err)
		require.Equal(test, value, cachedValue.(string))
	})

	test.Run("Get int entry from non-obfuscated cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})
		value := 50
		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: value,
		})
		require.NoError(test, err)

		var cachedValue any
		err = cache.Get(testCacheKey, &cachedValue)
		require.NoError(test, err)
		require.Equal(test, value, cachedValue.(int))
	})

	test.Run("Cache should be clean after Clean", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		var cachedValue any
		err = cache.Get(testCacheKey, &cachedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue, cachedValue)

		cache.Clean()

		cachedValue = nil
		err = cache.Get(testCacheKey, &cachedValue)
		require.Error(test, err, "key not found in the cache")
		require.Nil(test, cachedValue)
	})

	// -----------------------------------------------------------------------
	// NEW TEST CASES
	// -----------------------------------------------------------------------

	// 1. NewCache with Expiry == 0 must fall back to defaultExpiry (-1) so
	//    entries are never expired by the background cleaner.
	test.Run("NewCache with zero expiry uses defaultExpiry and entries never expire", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		shortClean := 1
		cache := NewCache(&CreateCacheParams{
			Expiry:        0, // explicitly zero → must use defaultExpiry = -1
			CleanInterval: time.Second * time.Duration(shortClean),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)
		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		// Wait for several clean cycles — entry must still be present.
		time.Sleep(time.Second * time.Duration(shortClean*3))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)
		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	// 2. UpdateTime with a new CleanInterval must reset the background cleaner
	//    ticker so the new interval takes effect immediately.
	test.Run("UpdateTime with CleanInterval resets background cleaner ticker", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		// Start with a very long clean interval so the cleaner won't fire on its own.
		longClean := 60
		shortExpiry := 1
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(shortExpiry),
			CleanInterval: time.Second * time.Duration(longClean),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		// Entry is live right after insertion.
		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)
		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		// Switch to a short clean interval while also lowering the expiry so
		// the next fresh entry we add will expire quickly.
		newClean := 1
		cache.Lock()
		cache.UpdateTime(&UpdateCacheTimeParams{
			Expiry:        time.Second * time.Duration(shortExpiry),
			CleanInterval: time.Second * time.Duration(newClean),
		})
		cache.Unlock()

		// Verify the cache field was updated.
		require.Equal(test, time.Second*time.Duration(newClean), cache.cleanInterval)

		// Add a fresh entry that will expire after shortExpiry.
		freshKey := "freshKey"
		err = cache.Add(&AddCacheParams{
			Key:   freshKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		// Wait for the new (short) clean interval to fire and remove the expired entry.
		time.Sleep(time.Second * time.Duration(shortExpiry+newClean+1))

		// Background cleaner should have evicted the fresh entry.
		getCachedValue, found = cache.get(freshKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	// 3. When no CleanInterval is set at construction, the clean() goroutine blocks
	//    on intervalCh.  Supplying one via UpdateTime should start the ticker.
	test.Run("cache with no initial CleanInterval starts cleaning after UpdateTime provides one", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		expiry := 1
		// No CleanInterval → clean() goroutine blocks on intervalCh.
		cache := NewCache(&CreateCacheParams{
			Expiry: time.Second * time.Duration(expiry),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		// Entry is live immediately.
		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)
		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		// Wait past expiry — get() itself detects and removes the expired entry.
		time.Sleep(time.Second * time.Duration(expiry+1))

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)

		// Add a new entry and now supply a CleanInterval via UpdateTime so the
		// goroutine wakes up and starts ticking.
		newKey := "newKey"
		err = cache.Add(&AddCacheParams{
			Key:   newKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		newClean := 1
		cache.Lock()
		cache.UpdateTime(&UpdateCacheTimeParams{
			Expiry:        time.Second * time.Duration(expiry),
			CleanInterval: time.Second * time.Duration(newClean),
		})
		cache.Unlock()

		// Wait for the goroutine to run one tick and evict the expired entry.
		time.Sleep(time.Second * time.Duration(expiry+newClean+1))

		getCachedValue, found = cache.get(newKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	// 4. Clean() must properly shut down the goroutine even when it is still
	//    blocking on intervalCh (i.e. no CleanInterval was ever configured).
	test.Run("Clean terminates goroutine when no CleanInterval was set", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		// Goroutine will block on intervalCh; Clean() must cancel the context.
		cache := NewCache(&CreateCacheParams{
			Expiry: time.Second * time.Duration(testCacheExpiry),
			// No CleanInterval.
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		// Clean() should cancel the context, unblocking the goroutine.
		cache.Clean()

		// All entries must be wiped.
		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	// 5. Update must work on a non-obfuscated cache (existing tests only use obfuscated).
	test.Run("Update value in non-obfuscated cache", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
			// IsCacheObfuscated: false (default)
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)
		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)

		updatedValue := &testStruct{Value: "updatedNonObfuscated"}
		err = cache.Update(&UpdateCacheParams{
			Key:   testCacheKey,
			Value: updatedValue,
		})
		require.NoError(test, err)

		getCachedValue, found = cache.get(testCacheKey, &testCacheValue)
		require.True(test, found)
		require.Equal(test, updatedValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	// 6. The public Get method must return an error when the key is not found.
	test.Run("public Get returns error for missing key", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		var dest any
		err := cache.Get("nonExistentKey", &dest)
		require.Error(test, err)
		require.EqualError(test, err, "key not found in the cache")
		require.Nil(test, dest)
	})

	// 7. A per-key Expiry that is shorter than the cache-level expiry must
	//    cause that entry to expire before entries using the cache-level expiry.
	test.Run("per-key expiry shorter than cache-level expiry expires sooner", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cacheExpiry := 10
		perKeyExpiry := 1
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(cacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		shortKey := "shortLivedKey"
		longKey := "longLivedKey"

		// shortKey uses a per-key expiry shorter than the cache-level expiry.
		err := cache.Add(&AddCacheParams{
			Key:    shortKey,
			Value:  testCacheValue,
			Expiry: time.Second * time.Duration(perKeyExpiry),
		})
		require.NoError(test, err)

		// longKey uses the cache-level expiry.
		err = cache.Add(&AddCacheParams{
			Key:   longKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		// Both are live immediately.
		_, found := cache.get(shortKey, &testCacheValue)
		require.True(test, found)
		_, found = cache.get(longKey, &testCacheValue)
		require.True(test, found)

		// After the short per-key expiry, shortKey must be gone but longKey survives.
		time.Sleep(time.Second * time.Duration(perKeyExpiry+1))

		getCachedValue, found := cache.get(shortKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)

		getCachedValue, found = cache.get(longKey, &testCacheValue)
		require.True(test, found)
		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	// 8. A per-key Expiry that is longer than the cache-level expiry must
	//    keep that entry alive after entries using the cache-level expiry expire.
	test.Run("per-key expiry longer than cache-level expiry outlives other entries", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cacheExpiry := 1
		perKeyExpiry := 5
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(cacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		shortKey := "shortLivedKey"
		longKey := "longLivedKey"

		// shortKey uses the cache-level (shorter) expiry.
		err := cache.Add(&AddCacheParams{
			Key:   shortKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		// longKey uses a per-key expiry that exceeds the cache-level expiry.
		err = cache.Add(&AddCacheParams{
			Key:    longKey,
			Value:  testCacheValue,
			Expiry: time.Second * time.Duration(perKeyExpiry),
		})
		require.NoError(test, err)

		// Both live initially.
		_, found := cache.get(shortKey, &testCacheValue)
		require.True(test, found)
		_, found = cache.get(longKey, &testCacheValue)
		require.True(test, found)

		// After the cache-level expiry, shortKey is gone but longKey (per-key) still lives.
		time.Sleep(time.Second * time.Duration(cacheExpiry+1))

		getCachedValue, found := cache.get(shortKey, &testCacheValue)
		require.False(test, found)
		require.Nil(test, getCachedValue)

		getCachedValue, found = cache.get(longKey, &testCacheValue)
		require.True(test, found)
		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})

	// 9. The four mutex-delegation helpers (Lock/Unlock/RLock/RUnlock) must
	//    not deadlock and must protect concurrent access correctly.
	test.Run("Lock RLock Unlock RUnlock helpers delegate to embedded mutex", func(test *testing.T) {
		defer flumetest.Start(test)
		test.Parallel()

		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		// Write lock: Lock / Unlock must not deadlock.
		cache.Lock()
		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		cache.Unlock()
		require.NoError(test, err)

		// Read lock: RLock / RUnlock must not deadlock and must allow the read.
		cache.RLock()
		getCachedValue, found := cache.get(testCacheKey, &testCacheValue)
		cache.RUnlock()
		require.True(test, found)
		require.Equal(test, testCacheValue.Value, getCachedValue.Value.(*testStruct).Value)
	})
}
