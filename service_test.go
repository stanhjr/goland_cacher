package main

import (
	"back_office_cacher/services"
	"github.com/go-redis/redis"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestGetCacheKey(t *testing.T) {

	partnerID := "123"
	authorization := "token123"
	url := "v1/bet_insights/partner-sports"

	os.Setenv("REDIS_KEY_VERSION", "2")
	defer os.Unsetenv("REDIS_KEY_VERSION")

	cacheService := services.NewCacheKeyService(partnerID, authorization, url)

	cacheKey, err := cacheService.GetCacheKey()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedCacheKey := ":2:123_token123_v1_bet_insights_partner_sports"
	if cacheKey != expectedCacheKey {
		t.Errorf("Expected cache key: %s, but got: %s", expectedCacheKey, cacheKey)
	}

	cacheService.Url = "unknown/url"
	cacheKey, err = cacheService.GetCacheKey()

	if err == nil {
		t.Error("Expected error for unknown URL, but got none.")
	}
}

func TestNewCacheKeyService(t *testing.T) {
	partnerID := "123"
	authorization := "token123"
	url := "v1/bet_insights/partner-sports"

	cacheService := services.NewCacheKeyService(partnerID, authorization, url)

	if cacheService.PartnerId != partnerID || cacheService.Authorization != authorization || cacheService.Url != url {
		t.Error("Values not set correctly in NewCacheKeyService.")
	}

	expectedUrlMap := map[string]string{
		"v1/bet_insights/partner-sports":           "v1_bet_insights_partner_sports",
		"v1/bet_insights/partner-widgets_settings": "v1_bet_insights_widgets_settings",
	}

	for key, value := range expectedUrlMap {
		if cacheService.UrlMap[key] != value {
			t.Errorf("Expected UrlMap[%s]: %s, but got: %s", key, value, cacheService.UrlMap[key])
		}
	}
}

func TestCacheResponseAndRetrieveFromCache(t *testing.T) {
	fakeResponse := httptest.NewRecorder()
	fakeResponse.WriteHeader(http.StatusOK)
	fakeResponse.WriteString("Mock data")
	fakeHeaders := fakeResponse.Header()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cacheKey := "test_cache_key"

	body := "Hello, World!"

	keyTtl := 5 * time.Minute

	err := services.CacheResponse(redisClient, cacheKey, fakeResponse.Result(), body, keyTtl)
	if err != nil {
		t.Fatalf("Error caching response: %v", err)
	}

	cachedResponse, err := services.RetrieveFromCache(redisClient, cacheKey)
	if err != nil {
		t.Fatalf("Error retrieving response from cache: %v", err)
	}

	if len(cachedResponse.Headers) != len(fakeHeaders) {
		t.Errorf("Expected %d headers, but got %d", len(fakeHeaders), len(cachedResponse.Headers))
	}

	for key, expectedValues := range fakeHeaders {
		actualValues, exists := cachedResponse.Headers[key]
		if !exists {
			t.Errorf("Header %s is missing", key)
			continue
		}

		for i, expectedValue := range expectedValues {
			if i >= len(actualValues) || actualValues[i] != expectedValue {
				t.Errorf("Header %s, value %s does not match", key, expectedValue)
			}
		}
	}

	if cachedResponse.Body != body {
		t.Errorf("Expected body: %s, but got: %s", body, cachedResponse.Body)
	}

	if cachedResponse.Status != http.StatusOK {
		t.Errorf("Expected status code %d, but got: %d", http.StatusOK, cachedResponse.Status)
	}
	redisClient.Del(cacheKey)
}
