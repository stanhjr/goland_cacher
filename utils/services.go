package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"net/http"
	"strings"
	"time"
)

type CacheService struct {
	PartnerId     string
	Authorization string
	Url           string
	UrlMap        map[string]string
}

func NewCacheService(
	PartnerId string,
	Authorization string,
	Url string,
) *CacheService {

	urlMap := map[string]string{
		"v1/bet_insights/partner-sports":           "v1_bet_insights_partner_sports",
		"v1/bet_insights/partner-widgets_settings": "v1_bet_insights_widgets_settings",
	}

	return &CacheService{
		PartnerId:     PartnerId,
		Authorization: Authorization,
		Url:           Url,
		UrlMap:        urlMap,
	}
}

func (c *CacheService) GetCacheKey() (string, error) {
	if mappedKey, ok := c.UrlMap[c.Url]; ok {
		return fmt.Sprintf("%s_%s_%s", c.PartnerId, c.Authorization, mappedKey), nil
	}
	for key, value := range c.UrlMap {
		if strings.Contains(c.Url, key) {
			return fmt.Sprintf("%s_%s_%s", c.PartnerId, c.Authorization, value), nil
		}

	}

	return "", errors.New("url not found in UrlMap")
}

type CachedResponse struct {
	Headers http.Header
	Body    string
	Status  int
}

func CacheResponse(redisClient *redis.Client, cacheKey string, resp *http.Response, body string, keyTtl time.Duration) error {

	cachedResponse := CachedResponse{
		Headers: resp.Header,
		Body:    body,
		Status:  resp.StatusCode,
	}

	cachedResponseJSON, err := json.Marshal(cachedResponse)
	if err != nil {
		return fmt.Errorf("failed to serialize response to JSON: %v", err)
	}

	err = redisClient.Set(cacheKey, string(cachedResponseJSON), keyTtl).Err()
	if err != nil {
		return fmt.Errorf("failed to cache response in Redis: %v", err)
	}

	return nil
}

func RetrieveFromCache(redisClient *redis.Client, cacheKey string) (*CachedResponse, error) {

	cachedResponseJSON, err := redisClient.Get(cacheKey).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve response from Redis: %v", err)
	}

	var cachedResponse CachedResponse
	err = json.Unmarshal([]byte(cachedResponseJSON), &cachedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	return &cachedResponse, nil
}
