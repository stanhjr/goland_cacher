// Package utils provides utility functions for handling caching and cache key generation.
package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"net/http"
	"os"
	"strings"
	"time"
)

// CacheKeyService is a service for generating cache keys based on specific criteria.
type CacheKeyService struct {
	PartnerId     string            // Partner ID associated with the cache key.
	Authorization string            // Authorization token associated with the cache key.
	Url           string            // URL used to determine the cache key.
	UrlMap        map[string]string // Mapping of specific URL patterns to their corresponding cache key representations.
}

// NewCacheKeyService creates a new CacheKeyService instance.
func NewCacheKeyService(
	PartnerId string,
	Authorization string,
	Url string,
) *CacheKeyService {

	// Define a mapping of URL patterns to cache key representations.
	urlMap := map[string]string{
		"v1/bet_insights/partner-sports":           "v1_bet_insights_partner_sports",
		"v1/bet_insights/partner-widgets_settings": "v1_bet_insights_widgets_settings",
	}

	return &CacheKeyService{
		PartnerId:     PartnerId,
		Authorization: Authorization,
		Url:           Url,
		UrlMap:        urlMap,
	}
}

// GetCacheKey generates a cache key based on the provided URL and the configured mapping.
func (c *CacheKeyService) GetCacheKey() (string, error) {
	for key, value := range c.UrlMap {
		if strings.Contains(c.Url, key) {
			keyVersion := os.Getenv("REDIS_KEY_VERSION")
			if keyVersion == "" {
				keyVersion = "1"
			}
			return fmt.Sprintf(":%s:%s_%s_%s", keyVersion, c.PartnerId, c.Authorization, value), nil
		}
	}

	return "", errors.New("url not found in UrlMap")
}

// CachedResponse represents a cached HTTP response, including headers, body, and status code.
type CachedResponse struct {
	Headers http.Header // HTTP headers of the cached response.
	Body    string      // Body content of the cached response.
	Status  int         // HTTP status code of the cached response.
}

// CacheResponse stores a given HTTP response and its body in Redis using the provided cache key.
func CacheResponse(
	redisClient *redis.Client,
	cacheKey string,
	resp *http.Response,
	body string,
	keyTtl time.Duration,
) error {

	// Create a CachedResponse instance to store in Redis.
	cachedResponse := CachedResponse{
		Headers: resp.Header,
		Body:    body,
		Status:  resp.StatusCode,
	}

	// Serialize the CachedResponse to JSON.
	cachedResponseJSON, err := json.Marshal(cachedResponse)
	if err != nil {
		return fmt.Errorf("failed to serialize response to JSON: %v", err)
	}

	// Store the JSON representation in Redis with the specified TTL.
	err = redisClient.Set(cacheKey, string(cachedResponseJSON), keyTtl).Err()
	if err != nil {
		return fmt.Errorf("failed to cache response in Redis: %v", err)
	}

	return nil
}

// RetrieveFromCache retrieves a CachedResponse from Redis based on the provided cache key.
func RetrieveFromCache(redisClient *redis.Client, cacheKey string) (*CachedResponse, error) {

	// Retrieve the JSON representation of the cached response from Redis.
	cachedResponseJSON, err := redisClient.Get(cacheKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve response from Redis: %v", err)
	}

	// Deserialize the JSON representation into a CachedResponse instance.
	var cachedResponse CachedResponse
	err = json.Unmarshal([]byte(cachedResponseJSON), &cachedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	return &cachedResponse, nil
}
