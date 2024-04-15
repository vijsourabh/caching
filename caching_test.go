package caching

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gemalto/flume/flumetest"
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
		require.Nil(test, cachedInfo)
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
		require.Equal(test, testCacheValue.Value, expectedValue.Value)
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
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

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
		require.Equal(test, testUpdatedCacheValue.Value, expectedUpdatedValue.Value)
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

	test.Run("GetValue of string entry from the cache", func(test *testing.T) {
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

		cachedValue, err := cache.GetValue(testCacheKey)
		require.NoError(test, err)
		require.Equal(test, value, cachedValue.(string))
	})

	test.Run("Getvalue of int entry from the cache", func(test *testing.T) {
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

		cachedValue, err := cache.GetValue(testCacheKey)
		require.NoError(test, err)
		require.Equal(test, value, cachedValue.(int))
	})
}
