package main

import (
	"back_office_cacher/utils"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

var redisClient *redis.Client

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	prometheus.MustRegister(httpRequestsTotal)
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()
	partnerID := r.Header.Get("Partner-Id")
	authorization := r.Header.Get("Authorization")

	if partnerID == "" || authorization == "" {
		http.Error(w, "Authentication credentials were not provided.", http.StatusUnauthorized)
		return
	}
	cacheService := utils.NewCacheService(partnerID, authorization, r.URL.Path)
	cacheKey, err := cacheService.GetCacheKey()
	if err != nil {
		http.Error(w, "Resource not found.", http.StatusNotFound)
		return
	}

	cachedResponse, err := utils.RetrieveFromCache(redisClient, cacheKey)

	if err == nil {
		for key, values := range cachedResponse.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(cachedResponse.Status)
		w.Write([]byte(cachedResponse.Body))
		return
	}

	remoteURL := os.Getenv("PROXY_URL")
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

	keyTtl := 30 * time.Minute

	if resp.StatusCode < 500 {
		err := utils.CacheResponse(
			redisClient,
			cacheKey,
			resp,
			string(body),
			keyTtl,
		)

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
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", proxyHandler)

	port := 8080
	fmt.Printf("Proxy server listening on :%d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Error starting proxy server:", err)
	}
}
