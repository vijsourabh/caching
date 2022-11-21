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
	testCacheExpiry        = 5
	testCacheCleanInterval = 5
	testCacheKey           = "key"
	testCacheValue         = &testStruct{
		Value: "value",
	}
)

func TestService_Cache(test *testing.T) {
	test.Run("get entry from the cache", func(test *testing.T) {
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		defer cache.Remove(testCacheKey)

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)
	})

	test.Run("get entry from the obfuscated cache", func(test *testing.T) {
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

		defer cache.Remove(testCacheKey)

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)
	})

	test.Run("not able to get entry from cache after removal", func(test *testing.T) {
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		cache.Remove(testCacheKey)

		getCachedValue, found = cache.Get(testCacheKey)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("not able to get expired entry", func(test *testing.T) {
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

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(expiryTime))

		getCachedValue, found = cache.Get(testCacheKey)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("get invalid entry from cache", func(test *testing.T) {
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		getCachedValue, found := cache.Get(testCacheKey)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("adding same key in cache multiple times with different value", func(test *testing.T) {
		cache := NewCache(&CreateCacheParams{
			Expiry:        time.Second * time.Duration(testCacheExpiry),
			CleanInterval: time.Second * time.Duration(testCacheCleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		newValue := &testStruct{
			Value: "newValue",
		}
		err = cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: newValue,
		})
		require.NoError(test, err)

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, newValue.Value, expectedValue.Value)
	})

	test.Run("fetch the value after cleaning of cache", func(test *testing.T) {
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

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)
	})

	test.Run("unable to fetch the expired value after cleaning of cache", func(test *testing.T) {
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

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		getCachedValue, found = cache.Get(testCacheKey)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("the entry shouldn't get expired even if expiry of the cache is decreased", func(test *testing.T) {
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

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		cache.UpdateExpiry(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)
	})

	test.Run("the entry will get expired even if expiry of the cache is increased", func(test *testing.T) {
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

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(expiry-1))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		cache.UpdateExpiry(time.Second * time.Duration(cleanInterval))

		time.Sleep(time.Second * time.Duration(1))

		getCachedValue, found = cache.Get(testCacheKey)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})

	test.Run("entry not get expired after cleaning if cache expiry is not set", func(test *testing.T) {
		cleanInterval := 1

		cache := NewCache(&CreateCacheParams{
			CleanInterval: time.Second * time.Duration(cleanInterval),
		})

		err := cache.Add(&AddCacheParams{
			Key:   testCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)
	})

	test.Run("entry will get expired after cleaning if cache expiry is added later", func(test *testing.T) {
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

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		cache.UpdateExpiry(time.Second * time.Duration(expiry))

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		// add the entry with new expiry
		newCacheKey := "abc"
		err = cache.Add(&AddCacheParams{
			Key:   newCacheKey,
			Value: testCacheValue,
		})
		require.NoError(test, err)

		getCachedValue, found = cache.Get(newCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval+1))

		getCachedValue, found = cache.Get(newCacheKey)
		require.False(test, found)
		require.Nil(test, getCachedValue)

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)
	})

	test.Run("expire entry after overriding the value of cache expiry", func(test *testing.T) {
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

		getCachedValue, found := cache.Get(testCacheKey)
		require.True(test, found)

		var expectedValue testStruct
		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval-1))

		getCachedValue, found = cache.Get(testCacheKey)
		require.True(test, found)

		err = json.Unmarshal(getCachedValue.Value, &expectedValue)
		require.NoError(test, err)
		require.Equal(test, testCacheValue.Value, expectedValue.Value)

		time.Sleep(time.Second * time.Duration(cleanInterval))

		getCachedValue, found = cache.Get(testCacheKey)
		require.False(test, found)
		require.Nil(test, getCachedValue)
	})
}
