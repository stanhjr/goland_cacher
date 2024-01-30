package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"io"
	"net/http"
	"net/url"
)

var redisClient *redis.Client

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "8QQnjhHdnAj9ZgVpB5AZSDpZCHpLne",
	})
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	partnerID := r.Header.Get("Partner-Id")
	authorization := r.Header.Get("Authorization")

	cacheKey := fmt.Sprintf("%s:%s", partnerID, authorization)
	cachedResponse, err := redisClient.Get(cacheKey).Result()

	if err == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedResponse))
		return
	}

	remoteURL := "https://backoffice-backend.dev.p13r.bcua.io"
	remote, err := url.Parse(remoteURL)
	if err != nil {
		http.Error(w, "Failed to parse remote URL", http.StatusInternalServerError)
		return
	}

	r.URL.Host = remote.Host
	r.URL.Scheme = remote.Scheme
	r.Host = remote.Host

	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, "Failed to make request to remote server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode == http.StatusOK {
		err := redisClient.Set(cacheKey, string(body), 0).Err()
		if err != nil {
			fmt.Println("Failed to cache response in Redis:", err)
		}
	}

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func main() {
	http.HandleFunc("/", proxyHandler)

	port := 8080
	fmt.Printf("Proxy server listening on :%d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Error starting proxy server:", err)
	}
}
